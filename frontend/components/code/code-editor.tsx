"use client";

import * as React from "react";
import { cn } from "@/lib/utils";

/**
 * A dependency-free code editor: a transparent <textarea> layered over a
 * syntax-highlighted <pre>. This avoids CodeMirror/Monaco (which need CDN
 * assets the Artifact/app CSP would block) while still giving language-aware
 * coloring, auto-indentation, Tab-to-indent, and auto-closing brackets.
 *
 * Highlighting is a lightweight regex tokenizer — good enough to read code at a
 * glance, not a full parser. The textarea remains the source of truth so
 * selection, IME, and accessibility behave natively.
 */

export type EditorLanguage =
  | "python"
  | "javascript"
  | "typescript"
  | "go"
  | "java"
  | "cpp"
  | "c";

interface CodeEditorProps {
  value: string;
  onChange: (value: string) => void;
  language: EditorLanguage;
  id?: string;
  rows?: number;
  placeholder?: string;
  className?: string;
  disabled?: boolean;
  "aria-label"?: string;
}

const KEYWORDS: Record<EditorLanguage, string[]> = {
  python: ["def", "return", "if", "elif", "else", "for", "while", "in", "not", "and", "or", "import", "from", "as", "class", "try", "except", "finally", "raise", "with", "lambda", "yield", "pass", "break", "continue", "True", "False", "None", "self", "print", "len", "range"],
  javascript: ["const", "let", "var", "function", "return", "if", "else", "for", "while", "of", "in", "new", "class", "extends", "this", "try", "catch", "finally", "throw", "await", "async", "import", "from", "export", "default", "true", "false", "null", "undefined", "console"],
  typescript: ["const", "let", "var", "function", "return", "if", "else", "for", "while", "of", "in", "new", "class", "extends", "this", "try", "catch", "finally", "throw", "await", "async", "import", "from", "export", "default", "interface", "type", "enum", "implements", "public", "private", "readonly", "number", "string", "boolean", "void", "true", "false", "null", "undefined", "console"],
  go: ["package", "import", "func", "return", "if", "else", "for", "range", "var", "const", "type", "struct", "interface", "map", "chan", "go", "defer", "select", "switch", "case", "default", "break", "continue", "nil", "true", "false", "string", "int", "bool", "error", "fmt"],
  java: ["public", "private", "protected", "class", "interface", "extends", "implements", "static", "final", "void", "return", "if", "else", "for", "while", "new", "this", "try", "catch", "finally", "throw", "throws", "import", "package", "int", "long", "double", "boolean", "String", "true", "false", "null", "System"],
  cpp: ["include", "int", "char", "double", "float", "bool", "void", "auto", "const", "return", "if", "else", "for", "while", "class", "struct", "public", "private", "namespace", "using", "new", "delete", "template", "typename", "true", "false", "nullptr", "std", "cout", "cin", "endl", "vector", "string"],
  c: ["include", "int", "char", "double", "float", "void", "const", "return", "if", "else", "for", "while", "struct", "typedef", "sizeof", "static", "printf", "scanf", "malloc", "free", "NULL"],
};

function escapeHTML(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

// Token classes → CSS color via inline styles keyed off CSS vars already in the
// theme, with sensible fallbacks. One pass, ordered so earlier matches win.
function highlight(src: string, language: EditorLanguage): string {
  const kw = new Set(KEYWORDS[language]);
  // Match: line comment | block comment | string | number | identifier | other.
  const re = /(\/\/[^\n]*|#[^\n]*)|(\/\*[\s\S]*?\*\/)|("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|`(?:\\.|[^`\\])*`)|(\b\d[\d._]*\b)|([A-Za-z_]\w*)|([\s\S])/g;
  let out = "";
  let m: RegExpExecArray | null;
  const comment = language === "python" || language === "c" || language === "cpp" || language === "go" || language === "java" || language === "javascript" || language === "typescript";
  while ((m = re.exec(src)) !== null) {
    if (m[1] !== undefined && comment) out += `<span class="tok-com">${escapeHTML(m[1])}</span>`;
    else if (m[1] !== undefined) out += escapeHTML(m[1]);
    else if (m[2] !== undefined) out += `<span class="tok-com">${escapeHTML(m[2])}</span>`;
    else if (m[3] !== undefined) out += `<span class="tok-str">${escapeHTML(m[3])}</span>`;
    else if (m[4] !== undefined) out += `<span class="tok-num">${escapeHTML(m[4])}</span>`;
    else if (m[5] !== undefined) {
      out += kw.has(m[5]) ? `<span class="tok-kw">${escapeHTML(m[5])}</span>` : escapeHTML(m[5]);
    } else out += escapeHTML(m[6] ?? "");
  }
  return out;
}

const OPEN_TO_CLOSE: Record<string, string> = { "(": ")", "[": "]", "{": "}" };

export function CodeEditor({
  value,
  onChange,
  language,
  id,
  rows = 12,
  placeholder,
  className,
  disabled,
  "aria-label": ariaLabel,
}: CodeEditorProps) {
  const taRef = React.useRef<HTMLTextAreaElement>(null);
  const preRef = React.useRef<HTMLPreElement>(null);
  // Caret position to apply after a programmatic edit. React resets the caret to
  // the end when a controlled value changes, so we restore it synchronously in a
  // layout effect (after commit, before paint) to avoid the rAF race that made
  // subsequently-typed characters land at the wrong offset.
  const pendingCaret = React.useRef<number | null>(null);

  // Keep the highlight layer scrolled in lockstep with the textarea.
  const syncScroll = React.useCallback(() => {
    if (taRef.current && preRef.current) {
      preRef.current.scrollTop = taRef.current.scrollTop;
      preRef.current.scrollLeft = taRef.current.scrollLeft;
    }
  }, []);

  React.useLayoutEffect(() => {
    if (pendingCaret.current !== null && taRef.current) {
      const c = pendingCaret.current;
      taRef.current.selectionStart = taRef.current.selectionEnd = c;
      pendingCaret.current = null;
      syncScroll();
    }
  });

  function setValueAndCaret(next: string, caret: number) {
    pendingCaret.current = caret;
    onChange(next);
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    const ta = e.currentTarget;
    const { selectionStart: s, selectionEnd: en } = ta;

    // Tab inserts two spaces (indent); Shift+Tab outdents the current line.
    if (e.key === "Tab") {
      e.preventDefault();
      if (e.shiftKey) {
        const lineStart = value.lastIndexOf("\n", s - 1) + 1;
        if (value.slice(lineStart, lineStart + 2) === "  ") {
          const next = value.slice(0, lineStart) + value.slice(lineStart + 2);
          setValueAndCaret(next, Math.max(lineStart, s - 2));
        }
      } else {
        const next = value.slice(0, s) + "  " + value.slice(en);
        setValueAndCaret(next, s + 2);
      }
      return;
    }

    // Enter: preserve the current line's indentation; add one level after an
    // opening brace/colon.
    if (e.key === "Enter") {
      e.preventDefault();
      const lineStart = value.lastIndexOf("\n", s - 1) + 1;
      const line = value.slice(lineStart, s);
      const indent = (line.match(/^[ \t]*/) ?? [""])[0];
      const prevChar = value[s - 1];
      const opensBlock = prevChar === "{" || prevChar === "[" || prevChar === "(" || (language === "python" && prevChar === ":");
      const extra = opensBlock ? "  " : "";
      // If we're between a matching pair like {|}, put the close on its own line.
      const nextChar = value[s];
      const closer = OPEN_TO_CLOSE[prevChar ?? ""];
      if (opensBlock && closer && nextChar === closer) {
        const insert = "\n" + indent + extra + "\n" + indent;
        const next = value.slice(0, s) + insert + value.slice(en);
        setValueAndCaret(next, s + 1 + indent.length + extra.length);
      } else {
        const insert = "\n" + indent + extra;
        const next = value.slice(0, s) + insert + value.slice(en);
        setValueAndCaret(next, s + insert.length);
      }
      return;
    }

    // Auto-close brackets/quotes around the caret.
    if (OPEN_TO_CLOSE[e.key] && s === en) {
      e.preventDefault();
      const close = OPEN_TO_CLOSE[e.key];
      const next = value.slice(0, s) + e.key + close + value.slice(en);
      setValueAndCaret(next, s + 1);
      return;
    }
  }

  const highlighted = React.useMemo(() => highlight(value, language), [value, language]);

  return (
    <div
      className={cn(
        "code-editor relative overflow-hidden rounded-md border border-border bg-surface focus-within:ring-2 focus-within:ring-ring",
        className,
      )}
    >
      <pre
        ref={preRef}
        aria-hidden
        className="code-layer pointer-events-none m-0 overflow-auto whitespace-pre px-3.5 py-3 font-mono text-[13px] leading-relaxed"
        // Trailing newline keeps the last line visible while the caret sits on it.
        dangerouslySetInnerHTML={{ __html: highlighted + "\n" }}
      />
      <textarea
        ref={taRef}
        id={id}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onScroll={syncScroll}
        onKeyDown={onKeyDown}
        spellCheck={false}
        autoCapitalize="off"
        autoCorrect="off"
        rows={rows}
        disabled={disabled}
        placeholder={placeholder}
        aria-label={ariaLabel}
        className="code-layer absolute inset-0 m-0 resize-none overflow-auto whitespace-pre border-0 bg-transparent px-3.5 py-3 font-mono text-[13px] leading-relaxed text-transparent caret-foreground outline-none placeholder:text-muted-foreground"
      />
    </div>
  );
}
