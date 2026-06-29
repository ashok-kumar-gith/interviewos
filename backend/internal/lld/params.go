package lld

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// SortField is a single normalized ordering instruction.
type SortField struct {
	Column string
	Desc   bool
}

// problemSortable is the ORDER BY allowlist for LLD problems. Only these columns
// may appear in a sort clause, guarding against SQL injection via the param.
var problemSortable = set("title", "slug", "difficulty", "order_index", "created_at")

// parsePagination reads page/page_size query params (defaults applied later in
// the service via normalizePage).
func parsePagination(c *gin.Context) (page, pageSize int) {
	page = atoiDefault(c.Query("page"), defaultPage)
	pageSize = atoiDefault(c.Query("page_size"), defaultPageSize)
	return page, pageSize
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// parseSort parses the `sort` query param ("-order_index,title") into validated
// SortField instructions, dropping any column not in the allowlist.
func parseSort(raw string, allowed map[string]struct{}) []SortField {
	if raw == "" {
		return nil
	}
	var out []SortField
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		desc := false
		if strings.HasPrefix(part, "-") {
			desc = true
			part = part[1:]
		} else if strings.HasPrefix(part, "+") {
			part = part[1:]
		}
		col := strings.ToLower(strings.TrimSpace(part))
		if !sortColumnAllowed(col, allowed) {
			continue
		}
		out = append(out, SortField{Column: col, Desc: desc})
	}
	return out
}

// difficultyParam validates an optional difficulty query value, returning nil
// (drop the filter) when absent or malformed.
func difficultyParam(s string) *Difficulty {
	switch Difficulty(s) {
	case DifficultyEasy, DifficultyMedium, DifficultyHard:
		d := Difficulty(s)
		return &d
	}
	return nil
}

func sortColumnAllowed(col string, allowed map[string]struct{}) bool {
	_, ok := allowed[strings.ToLower(col)]
	return ok
}

func set(keys ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		m[k] = struct{}{}
	}
	return m
}
