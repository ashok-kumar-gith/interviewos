package ai

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// datatypesJSONB scans a JSONB column into raw bytes for lazy decoding. It is a
// minimal local helper so the readers can pull JSONB arrays/objects (skills,
// target_keywords, metrics, pillar_strengths) without depending on a wider ORM
// datatypes package.
type datatypesJSONB []byte

// Scan implements sql.Scanner.
func (j *datatypesJSONB) Scan(src any) error {
	if src == nil {
		*j = nil
		return nil
	}
	switch v := src.(type) {
	case []byte:
		*j = append((*j)[:0], v...)
	case string:
		*j = []byte(v)
	default:
		return fmt.Errorf("ai: cannot scan %T into JSONB", src)
	}
	return nil
}

// Value implements driver.Valuer (read-only use, but kept for completeness).
func (j datatypesJSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

// toStrings decodes a JSONB array of strings, returning an empty slice on
// null/empty/invalid input (defensive: never panics on dirty data).
func (j datatypesJSONB) toStrings() []string {
	if len(j) == 0 {
		return nil
	}
	var out []string
	if err := json.Unmarshal(j, &out); err != nil {
		return nil
	}
	return out
}

// toIntMap decodes a JSONB object of string->int (e.g. pillar_strengths).
func (j datatypesJSONB) toIntMap() map[string]int {
	if len(j) == 0 {
		return nil
	}
	var out map[string]int
	if err := json.Unmarshal(j, &out); err != nil {
		return nil
	}
	return out
}
