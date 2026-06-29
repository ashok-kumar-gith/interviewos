package content

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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

// parseSort parses the `sort` query param ("-created_at,title") into validated
// SortField instructions, dropping any column not in the allowlist (guards
// against SQL injection via ORDER BY).
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

// parseFilter parses the RHS-colon `filter` query param
// ("difficulty:hard,priority:high") into a key→value map. Unknown keys are kept
// and ignored by callers that don't recognize them.
func parseFilter(raw string) map[string]string {
	out := map[string]string{}
	if raw == "" {
		return out
	}
	for _, part := range strings.Split(raw, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(kv[0]))
		v := strings.TrimSpace(kv[1])
		if k != "" && v != "" {
			out[k] = v
		}
	}
	return out
}

// queryUUID parses an optional UUID query param. Returns nil when absent or
// malformed so a bad value simply drops the filter rather than erroring.
func queryUUID(c *gin.Context, key string) *uuid.UUID {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &id
}

// filterUUID parses an optional UUID drawn from the parsed filter map.
func filterUUID(filters map[string]string, key string) *uuid.UUID {
	raw, ok := filters[key]
	if !ok || raw == "" {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &id
}

// Sort allowlists per resource. Only these columns may be used in ORDER BY.
var (
	trackSortable    = set("name", "slug", "sort_order", "created_at")
	pillarSortable   = set("name", "sort_order", "weight", "created_at")
	topicSortable    = set("name", "slug", "difficulty", "priority", "sort_order", "estimated_hours", "created_at")
	resourceSortable = set("title", "type", "priority", "estimated_minutes", "created_at")
	patternSortable  = set("name", "slug", "sort_order", "created_at")
	problemSortable  = set("title", "slug", "difficulty", "frequency_score", "estimated_minutes", "created_at")
	companySortable  = set("name", "slug", "sort_order", "created_at")
)

func set(keys ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		m[k] = struct{}{}
	}
	return m
}
