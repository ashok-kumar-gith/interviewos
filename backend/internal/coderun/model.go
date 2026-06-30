package coderun

// Limits and timeouts for code execution. Kept as package constants so the
// service, handler, and tests share one source of truth.
const (
	// MaxSourceBytes caps the submitted source size (~64KB).
	MaxSourceBytes = 64 * 1024
	// MaxStdinBytes caps the submitted stdin size (~16KB).
	MaxStdinBytes = 16 * 1024
)

// RunInput is the validated, language-resolved request the service hands to the
// Piston client.
type RunInput struct {
	Language string
	Source   string
	Stdin    string
}

// RunOutput is the normalized result of a code execution, independent of the
// upstream executor's wire shape.
type RunOutput struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	Language string `json:"language"`
	Version  string `json:"version"`
	Ran      bool   `json:"ran"`
	Message  string `json:"message,omitempty"`
}

// runRequest is the request DTO for POST /code/run (mirrors openapi.yaml
// CodeRunRequest).
type runRequest struct {
	Language string `json:"language"`
	Source   string `json:"source"`
	Stdin    string `json:"stdin"`
}

// SupportedLanguages is the allowlist of languages the executor accepts. The
// keys are the public language identifiers; values are the corresponding Piston
// runtime language names (which happen to match for these, but are resolved
// explicitly so the mapping is intentional and not coincidental).
var SupportedLanguages = map[string]string{
	"python":     "python",
	"javascript": "javascript",
	"typescript": "typescript",
	"go":         "go",
	"java":       "java",
	"cpp":        "c++",
	"c":          "c",
}

// IsSupported reports whether lang is in the allowlist.
func IsSupported(lang string) bool {
	_, ok := SupportedLanguages[lang]
	return ok
}
