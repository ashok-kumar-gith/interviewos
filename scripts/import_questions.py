#!/usr/bin/env python3
"""
Import DSA questions from an Excel export into the InterviewOS `problems` table.

- Reads the workbook's Sheet1: category-header rows (a bare label like "Array")
  delimit sections; subsequent rows are problem URLs (LeetCode / GfG / other) with
  technique, complexity, solved and hint columns.
- Maps each category to a `patterns` slug (reusing the seeded ones; creating a few
  new patterns where the user's taxonomy has no equivalent) so the existing
  pattern/category filter and text search apply to the imported problems.
- Emits idempotent SQL: each problem is inserted only if no existing problem shares
  its slug OR url (so re-running, and overlap with the seeded set, never duplicates).

Usage:
    python3 scripts/import_questions.py "/path/Questions list.xlsx" > /tmp/import.sql
    psql "$DATABASE_URL" -f /tmp/import.sql
"""
import sys, re, openpyxl

# Category (as written in the sheet) -> pattern slug. Existing seeded patterns are
# reused; NEW_PATTERNS are created by this import so those categories are filterable.
CAT_TO_PATTERN = {
    "Array": "arrays-hashing",
    "Matrix": "matrix",
    "Hashing": "arrays-hashing",
    "Link List": "linked-list",
    "2 Pointer": "two-pointers",
    "Greedy": "greedy",
    "Recursion and Backtracking": "backtracking",
    "Binary Search": "binary-search",
    "Bit Operation": "bit-manipulation",
    "Stack and Queue": "stack",
    "String": "strings",
    "Binary Tree": "trees",
    "Binary Search Tree": "binary-search-tree",
    "TRIE": "tries",
    "Graph": "graphs",
    "Heap / Priority Queue": "heap",
    "Dynamic programming": "dynamic-programming",
    "Random Questions": "math",
}
# slug -> (name, description) for patterns this import must create if missing.
NEW_PATTERNS = {
    "matrix": ("Matrix", "2D grid traversal, rotation, and search problems."),
    "strings": ("Strings", "String manipulation, parsing, and pattern-matching problems."),
    "binary-search-tree": ("Binary Search Tree", "BST insertion, deletion, validation, and ordered traversal."),
}

def sqlstr(s):
    """SQL single-quoted literal (NULL for empty)."""
    if s is None or str(s).strip() == "":
        return "NULL"
    return "'" + str(s).replace("'", "''") + "'"

def slug_from_url(url):
    u = url.strip().rstrip("/")
    # leetcode.com/problems/<slug>[/...]
    m = re.search(r"/problems/([a-zA-Z0-9\-]+)", u)
    if m:
        return m.group(1).lower()
    # gfg practice: /problems/<slug-with-id>/1  -> already caught above; fallbacks:
    seg = re.sub(r"[?#].*$", "", u).rstrip("/").split("/")[-1]
    seg = re.sub(r"[^a-zA-Z0-9\-]+", "-", seg).strip("-").lower()
    return seg or "problem"

def title_from_slug(slug):
    s = re.sub(r"\d+$", "", slug)              # trim trailing GfG numeric ids
    s = s.replace("-", " ").strip()
    return " ".join(w.capitalize() for w in s.split()) or slug

def platform(url):
    if "leetcode.com" in url: return "leetcode"
    if "geeksforgeeks" in url: return "gfg"
    return "custom"

def difficulty(remark):
    r = (remark or "").strip().lower()
    return r if r in ("easy", "medium", "hard") else "medium"

def main():
    path = sys.argv[1] if len(sys.argv) > 1 else "/Users/ashokkumar/Downloads/Questions list.xlsx"
    ws = openpyxl.load_workbook(path, read_only=True, data_only=True)["Sheet1"]
    cur = None
    seen = set()
    items = []  # (slug, title, difficulty, platform, url, external_id, approach, hint, pattern_slug)
    for r in list(ws.iter_rows(values_only=True))[1:]:
        if not r or r[0] is None:
            continue
        c0 = str(r[0]).strip()
        if not c0:
            continue
        if not c0.startswith("http"):
            cur = c0
            continue
        url = c0.split()[0].strip()
        slug = slug_from_url(url)
        if slug in seen:            # dedup within the file
            continue
        seen.add(slug)
        remark   = str(r[1]).strip() if len(r) > 1 and r[1] else ""
        tech     = str(r[2]).strip() if len(r) > 2 and r[2] else ""
        tcx      = str(r[3]).strip() if len(r) > 3 and r[3] else ""
        scx      = str(r[4]).strip() if len(r) > 4 and r[4] else ""
        hint     = str(r[6]).strip() if len(r) > 6 and r[6] else ""
        approach = tech
        if tcx or scx:
            approach = (approach + f"\n\n**Complexity:** time {tcx or 'n/a'}, space {scx or 'n/a'}").strip()
        pat = CAT_TO_PATTERN.get(cur, "math")
        items.append((slug, title_from_slug(slug), difficulty(remark), platform(url),
                      url, slug, approach, hint, pat))

    out = []
    out.append("BEGIN;")
    out.append("-- 1. Ensure the new patterns for the imported taxonomy exist.")
    for slug, (name, desc) in NEW_PATTERNS.items():
        out.append(
            "INSERT INTO patterns (id, track_id, slug, name, description, sort_order, created_at, updated_at) "
            "SELECT gen_random_uuid(), (SELECT id FROM tracks WHERE slug='backend-sde3'), "
            f"{sqlstr(slug)}, {sqlstr(name)}, {sqlstr(desc)}, 100, now(), now() "
            f"WHERE NOT EXISTS (SELECT 1 FROM patterns WHERE slug={sqlstr(slug)});")
    out.append("-- 2. Insert problems idempotently (skip if slug OR url already present).")
    for slug, title, diff, plat, url, ext, approach, hint, pat in items:
        out.append(
            "INSERT INTO problems (id, track_id, slug, title, difficulty, platform, external_id, url, "
            "approach_md, common_mistakes, estimated_minutes, frequency_score, is_premium, created_at, updated_at) "
            "SELECT gen_random_uuid(), (SELECT id FROM tracks WHERE slug='backend-sde3'), "
            f"{sqlstr(slug)}, {sqlstr(title)}, {sqlstr(diff)}::difficulty, {sqlstr(plat)}::problem_platform, "
            f"{sqlstr(ext)}, {sqlstr(url)}, {sqlstr(approach)}, {sqlstr(hint)}, 30, 0, false, now(), now() "
            f"WHERE NOT EXISTS (SELECT 1 FROM problems WHERE slug={sqlstr(slug)} OR url={sqlstr(url)});")
    out.append("-- 3. Link each imported problem to its category pattern (idempotent).")
    vals = ",".join(f"({sqlstr(s)},{sqlstr(p)})" for s, *_ , p in [(i[0], i[8]) for i in items])
    # rebuild cleanly: (slug, pattern_slug) pairs
    pairs = ",".join(f"({sqlstr(i[0])},{sqlstr(i[8])})" for i in items)
    out.append(
        "INSERT INTO problem_patterns (id, problem_id, pattern_id, created_at, updated_at) "
        "SELECT gen_random_uuid(), p.id, pat.id, now(), now() "
        f"FROM (VALUES {pairs}) AS m(pslug, patslug) "
        "JOIN problems p ON p.slug = m.pslug "
        "JOIN patterns pat ON pat.slug = m.patslug "
        "WHERE NOT EXISTS (SELECT 1 FROM problem_patterns pp WHERE pp.problem_id=p.id AND pp.pattern_id=pat.id);")
    out.append("COMMIT;")
    print("\n".join(out))
    print(f"-- generated {len(items)} problem inserts across {len(set(i[8] for i in items))} patterns", file=sys.stderr)

if __name__ == "__main__":
    main()
