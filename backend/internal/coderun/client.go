package coderun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// defaultPistonBaseURL is the public Piston API. Execution runs server-side
// against this endpoint so no local Docker is required.
const defaultPistonBaseURL = "https://emkc.org/api/v2/piston"

// Executor is the narrow port the service depends on to run code. Keeping it an
// interface lets the service be unit-tested against a fake (see service_test.go).
type Executor interface {
	// Execute runs the resolved input and returns the normalized output. It
	// returns an error only for transport/upstream failures (unreachable,
	// timeout, malformed response, or no runtime version); a program that
	// compiled-and-ran but exited non-zero is NOT an error — it is returned with
	// Ran=true and the captured stdout/stderr/exit code.
	Execute(ctx context.Context, in RunInput) (RunOutput, error)
}

// PistonClient is an Executor backed by the public Piston HTTP API. Resolved
// runtime versions are cached in-memory (Piston exposes one "latest" version
// per language) so we hit /runtimes at most once per language per process
// lifetime under normal operation.
type PistonClient struct {
	baseURL string
	http    *http.Client

	mu       sync.RWMutex
	versions map[string]string // piston language name -> resolved version
}

// NewPistonClient constructs a PistonClient. A nil httpClient uses a client with
// the supplied per-call timeout enforced via context in the service; the
// http.Client timeout here is a generous backstop.
func NewPistonClient() *PistonClient {
	return &PistonClient{
		baseURL:  defaultPistonBaseURL,
		http:     &http.Client{Timeout: 15 * time.Second},
		versions: make(map[string]string),
	}
}

// Compile-time assertion that *PistonClient satisfies the Executor port.
var _ Executor = (*PistonClient)(nil)

// --- Piston wire shapes ---

type pistonRuntime struct {
	Language string   `json:"language"`
	Version  string   `json:"version"`
	Aliases  []string `json:"aliases"`
}

type pistonFile struct {
	Content string `json:"content"`
}

type pistonExecuteRequest struct {
	Language string       `json:"language"`
	Version  string       `json:"version"`
	Files    []pistonFile `json:"files"`
	Stdin    string       `json:"stdin,omitempty"`
}

type pistonStage struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Output string `json:"output"`
	Code   *int   `json:"code"`
}

type pistonExecuteResponse struct {
	Language string       `json:"language"`
	Version  string       `json:"version"`
	Run      pistonStage  `json:"run"`
	Compile  *pistonStage `json:"compile"`
}

// Execute runs the input against Piston, resolving the runtime version first.
func (c *PistonClient) Execute(ctx context.Context, in RunInput) (RunOutput, error) {
	pistonLang := in.Language
	version, err := c.resolveVersion(ctx, pistonLang)
	if err != nil {
		return RunOutput{}, err
	}

	body, err := json.Marshal(pistonExecuteRequest{
		Language: pistonLang,
		Version:  version,
		Files:    []pistonFile{{Content: in.Source}},
		Stdin:    in.Stdin,
	})
	if err != nil {
		return RunOutput{}, fmt.Errorf("%w: marshal request: %v", ErrExecutorUnavailable, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/execute", bytes.NewReader(body))
	if err != nil {
		return RunOutput{}, fmt.Errorf("%w: build request: %v", ErrExecutorUnavailable, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return RunOutput{}, fmt.Errorf("%w: %v", ErrExecutorUnavailable, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return RunOutput{}, fmt.Errorf("%w: read response: %v", ErrExecutorUnavailable, err)
	}

	if resp.StatusCode != http.StatusOK {
		return RunOutput{}, fmt.Errorf("%w: executor returned status %d", ErrExecutorUnavailable, resp.StatusCode)
	}

	var pr pistonExecuteResponse
	if err := json.Unmarshal(raw, &pr); err != nil {
		return RunOutput{}, fmt.Errorf("%w: decode response: %v", ErrExecutorUnavailable, err)
	}

	return normalizePistonResponse(pr, in.Language), nil
}

// normalizePistonResponse maps Piston's compile+run stages into a single
// normalized RunOutput. A compile-stage failure (non-zero exit) is surfaced as
// stderr with the compile exit code; otherwise the run stage wins. Either way
// Ran is true: the executor accepted and processed the submission.
func normalizePistonResponse(pr pistonExecuteResponse, publicLang string) RunOutput {
	out := RunOutput{
		Language: publicLang,
		Version:  pr.Version,
		Ran:      true,
		Stdout:   pr.Run.Stdout,
		Stderr:   pr.Run.Stderr,
	}
	if pr.Run.Code != nil {
		out.ExitCode = *pr.Run.Code
	}

	// If compilation failed, the program never ran; surface the compile diagnostics.
	if pr.Compile != nil && pr.Compile.Code != nil && *pr.Compile.Code != 0 {
		out.Stderr = pr.Compile.Stderr
		if out.Stderr == "" {
			out.Stderr = pr.Compile.Output
		}
		out.ExitCode = *pr.Compile.Code
		out.Message = "compilation failed"
	}
	return out
}

// resolveVersion returns the latest Piston runtime version for the given Piston
// language name, caching the result in-memory. On a cache miss it fetches
// /runtimes once and records every language's latest version it sees.
func (c *PistonClient) resolveVersion(ctx context.Context, pistonLang string) (string, error) {
	c.mu.RLock()
	v, ok := c.versions[pistonLang]
	c.mu.RUnlock()
	if ok {
		return v, nil
	}

	runtimes, err := c.fetchRuntimes(ctx)
	if err != nil {
		return "", err
	}

	// Build a language -> latest version map. Piston may list multiple versions
	// of a language; pick the highest by a simple semver-ish comparison. Aliases
	// (e.g. "node" for javascript) are recorded too so resolution is forgiving.
	latest := make(map[string]string)
	for _, rt := range runtimes {
		consider := func(name string) {
			if cur, exists := latest[name]; !exists || versionLess(cur, rt.Version) {
				latest[name] = rt.Version
			}
		}
		consider(rt.Language)
		for _, a := range rt.Aliases {
			consider(a)
		}
	}

	c.mu.Lock()
	for k, val := range latest {
		c.versions[k] = val
	}
	c.mu.Unlock()

	if v, ok := latest[pistonLang]; ok {
		return v, nil
	}
	return "", fmt.Errorf("%w: %q", ErrRuntimeUnavailable, pistonLang)
}

func (c *PistonClient) fetchRuntimes(ctx context.Context) ([]pistonRuntime, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/runtimes", nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build runtimes request: %v", ErrExecutorUnavailable, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExecutorUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: runtimes returned status %d", ErrExecutorUnavailable, resp.StatusCode)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("%w: read runtimes: %v", ErrExecutorUnavailable, err)
	}

	var runtimes []pistonRuntime
	if err := json.Unmarshal(raw, &runtimes); err != nil {
		return nil, fmt.Errorf("%w: decode runtimes: %v", ErrExecutorUnavailable, err)
	}
	if len(runtimes) == 0 {
		return nil, fmt.Errorf("%w: empty runtimes list", ErrRuntimeUnavailable)
	}
	return runtimes, nil
}

// versionLess reports whether version a is older than b using a dotted numeric
// comparison, falling back to lexical order for non-numeric components. It is a
// best-effort "pick the newest runtime" heuristic, not a full semver parser.
func versionLess(a, b string) bool {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var av, bv string
		if i < len(as) {
			av = as[i]
		}
		if i < len(bs) {
			bv = bs[i]
		}
		ai, aerr := atoiSafe(av)
		bi, berr := atoiSafe(bv)
		if aerr == nil && berr == nil {
			if ai != bi {
				return ai < bi
			}
			continue
		}
		if av != bv {
			return av < bv
		}
	}
	return false
}

func atoiSafe(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("non-numeric")
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

// sortedSupportedLanguages returns the allowlist keys in stable order (used for
// deterministic error messages / docs). Unused by hot paths.
func sortedSupportedLanguages() []string {
	out := make([]string, 0, len(SupportedLanguages))
	for k := range SupportedLanguages {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
