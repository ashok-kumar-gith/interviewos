package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Usage carries token accounting returned by the model provider. Fields are zero
// when the provider does not report them (or for the deterministic fallback,
// which never calls a provider).
type Usage struct {
	PromptTokens     int
	CompletionTokens int
}

// Client is the narrow port the orchestrator depends on to obtain a completion
// from a large language model. A single method keeps the seam small (ISP): the
// orchestrator owns prompt construction, parsing, fallback, and persistence;
// the Client is only responsible for "system + prompt -> text".
//
// Implementations: AnthropicClient (real HTTP call to the Claude Messages API).
// Tests inject a fake Client to exercise the success and error paths without any
// network access.
type Client interface {
	// Complete sends a single-turn request (system + user prompt) and returns the
	// model's text response plus token usage. It must honor ctx cancellation and
	// its own timeout, returning an error on transport failure, non-2xx status,
	// or an empty/malformed response so the orchestrator can fall back.
	Complete(ctx context.Context, system, prompt string, maxTokens int) (string, Usage, error)
}

// anthropicVersion is the pinned Anthropic API version header value.
const anthropicVersion = "2023-06-01"

// anthropicMessagesURL is the Claude Messages API endpoint.
const anthropicMessagesURL = "https://api.anthropic.com/v1/messages"

// AnthropicClient is the real Claude Messages API adapter. It is only
// constructed (and only used) when an API key is present; the orchestrator
// chooses the deterministic fallback otherwise.
type AnthropicClient struct {
	apiKey  string
	model   string
	http    *http.Client
	baseURL string
}

// AnthropicConfig configures an AnthropicClient.
type AnthropicConfig struct {
	APIKey string
	Model  string
	// Timeout bounds a single API call (per-call timeout, §9 cost controls).
	// Defaults to 30s when zero.
	Timeout time.Duration
	// HTTPClient lets callers/tests inject a transport. Defaults to a client with
	// the configured Timeout.
	HTTPClient *http.Client
	// BaseURL overrides the endpoint (tests). Defaults to anthropicMessagesURL.
	BaseURL string
}

// NewAnthropicClient constructs an AnthropicClient. It returns an error when the
// API key is empty so the composition root can fall back to deterministic mode.
func NewAnthropicClient(cfg AnthropicConfig) (*AnthropicClient, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("ai: anthropic api key is required")
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = anthropicMessagesURL
	}
	return &AnthropicClient{
		apiKey:  cfg.APIKey,
		model:   model,
		http:    httpClient,
		baseURL: baseURL,
	}, nil
}

// Model reports the configured Claude model id.
func (c *AnthropicClient) Model() string { return c.model }

// anthropicRequest is the Messages API request body.
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse is the relevant subset of the Messages API response.
type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Complete calls the Claude Messages API and returns the concatenated text of
// the response's text blocks plus token usage.
func (c *AnthropicClient) Complete(ctx context.Context, system, prompt string, maxTokens int) (string, Usage, error) {
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	body, err := json.Marshal(anthropicRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  []anthropicMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", Usage{}, fmt.Errorf("ai: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return "", Usage{}, fmt.Errorf("ai: build request: %w", err)
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("content-type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", Usage{}, fmt.Errorf("ai: anthropic request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", Usage{}, fmt.Errorf("ai: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", Usage{}, fmt.Errorf("ai: anthropic status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", Usage{}, fmt.Errorf("ai: decode response: %w", err)
	}

	var sb strings.Builder
	for _, block := range parsed.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	text := strings.TrimSpace(sb.String())
	if text == "" {
		return "", Usage{}, fmt.Errorf("ai: empty completion from anthropic")
	}
	return text, Usage{
		PromptTokens:     parsed.Usage.InputTokens,
		CompletionTokens: parsed.Usage.OutputTokens,
	}, nil
}
