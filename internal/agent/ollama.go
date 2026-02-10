package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const defaultOllamaModel = "qwen2.5-coder:latest"

// OllamaAgent runs code reviews using Ollama servers
type OllamaAgent struct {
	BaseURL    string         // Ollama server URL (default: "http://localhost:11434")
	Model      string         // Model name (e.g., "qwen2.5-coder:latest")
	Reasoning  ReasoningLevel // Reasoning level
	Agentic    bool           // Whether agentic mode is enabled
	HTTPClient *http.Client   // HTTP client for requests (configurable for testing)
}

// NewOllamaAgent creates a new Ollama agent with default settings
func NewOllamaAgent(baseURL string) *OllamaAgent {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &OllamaAgent{
		BaseURL:   baseURL,
		Reasoning: ReasoningStandard,
	}
}

// ollamaChatMessage represents a message in the chat API
type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ollamaChatRequest represents a request to Ollama's /api/chat endpoint
type ollamaChatRequest struct {
	Model    string                 `json:"model"`
	Messages []ollamaChatMessage    `json:"messages"`
	Stream   bool                   `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// ollamaChatResponse represents a streaming response from Ollama
type ollamaChatResponse struct {
	Message ollamaChatMessage `json:"message"`
	Done    bool              `json:"done"`
	Error   string            `json:"error,omitempty"`
}

// Name returns the agent identifier
func (a *OllamaAgent) Name() string {
	return "ollama"
}

// buildRequest creates an Ollama API request with reasoning level mapped to temperature
func (a *OllamaAgent) buildRequest(prompt string) ollamaChatRequest {
	model := a.Model
	if model == "" {
		model = defaultOllamaModel
	}

	options := make(map[string]interface{})

	// Map reasoning level to temperature/top_p
	switch a.Reasoning {
	case ReasoningThorough:
		options["temperature"] = 0.3
		options["top_p"] = 0.9
	case ReasoningFast:
		options["temperature"] = 1.0
		options["top_p"] = 1.0
	default: // ReasoningStandard
		options["temperature"] = 0.7
		options["top_p"] = 0.95
	}

	// Create system message with review instructions and user message with prompt
	messages := []ollamaChatMessage{
		{
			Role:    "system",
			Content: "You are a code review assistant. Analyze the provided code changes and provide constructive feedback.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	return ollamaChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
		Options:  options,
	}
}

// augmentPromptForAgentic modifies the prompt to include tool descriptions when agentic mode is enabled.
// Note: Tool support depends on the model. Models like Qwen 2.5+ understand this syntax.
func (a *OllamaAgent) augmentPromptForAgentic(prompt string) string {
	if !a.Agentic {
		return prompt
	}

	// Append analysis-only tool descriptions. No write/execute tools are available
	// because Ollama lacks tool-call parsing and execution logic.
	toolDescriptions := `

You have access to the following analysis capabilities:

1. read_file(path: string) -> string
   Read the contents of a file at the given path.

Analyze the code thoroughly and describe any issues or suggestions clearly.
`

	return prompt + toolDescriptions
}

// getHTTPClient returns the HTTP client for requests
func (a *OllamaAgent) getHTTPClient() *http.Client {
	if a.HTTPClient != nil {
		return a.HTTPClient
	}
	return &http.Client{
		Timeout: 5 * time.Minute,
	}
}

// CheckHealth verifies that the Ollama server is reachable and responding.
// Returns nil if the server is healthy, or a descriptive error otherwise.
func (a *OllamaAgent) CheckHealth(ctx context.Context) error {
	// Create context with timeout to avoid hanging
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Make GET request to /api/tags endpoint
	healthURL := a.BaseURL + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return fmt.Errorf("create health check request: %w", err)
	}

	// Use the configured HTTP client so custom transport/TLS settings apply
	client := a.getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return a.classifyError(err, 0, "")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("ollama server health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// Review runs a code review and returns the output
func (a *OllamaAgent) Review(ctx context.Context, repoPath, commitSHA, prompt string, output io.Writer) (string, error) {
	// Fast-fail: check that the Ollama server is reachable
	if err := a.CheckHealth(ctx); err != nil {
		return "", err
	}

	// Augment prompt for agentic mode if enabled
	prompt = a.augmentPromptForAgentic(prompt)

	// Build the request
	reqData := a.buildRequest(prompt)

	// Validate model name
	if err := validateOllamaModel(reqData.Model); err != nil {
		return "", err
	}

	// Marshal to JSON
	reqBody, err := json.Marshal(reqData)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request to /api/chat
	url := a.BaseURL + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := a.getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return "", a.classifyError(err, 0, reqData.Model)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", a.classifyError(fmt.Errorf("%s", string(body)), resp.StatusCode, reqData.Model)
	}

	// Parse streaming response
	return a.parseStream(resp.Body, output)
}

// parseStream parses the NDJSON streaming response from Ollama
func (a *OllamaAgent) parseStream(reader io.Reader, output io.Writer) (string, error) {
	scanner := bufio.NewScanner(reader)
	var result strings.Builder
	sw := newSyncWriter(output)

	var consecutiveParseFailures int
	var sawValidJSON bool

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var resp ollamaChatResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			consecutiveParseFailures++
			// If we've never seen valid JSON and hit many failures,
			// the server is likely returning an error page (e.g. HTML from a proxy)
			if !sawValidJSON && consecutiveParseFailures >= 5 {
				return "", fmt.Errorf("ollama returned non-JSON response (first line: %s)", truncate(line, 200))
			}
			continue
		}
		sawValidJSON = true
		consecutiveParseFailures = 0

		// Check for error in response
		if resp.Error != "" {
			return result.String(), fmt.Errorf("ollama error: %s", resp.Error)
		}

		// Accumulate message content
		if resp.Message.Content != "" {
			result.WriteString(resp.Message.Content)
			// Stream progress to output if provided
			if sw != nil {
				if _, err := sw.Write([]byte(resp.Message.Content)); err != nil {
					return result.String(), fmt.Errorf("write output: %w", err)
				}
			}
		}

		// Stop when done
		if resp.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return result.String(), fmt.Errorf("read stream: %w", err)
	}

	finalResult := result.String()
	if finalResult == "" {
		return "No review output generated", nil
	}

	return finalResult, nil
}

// truncate returns s truncated to maxLen, appending "..." if truncated
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// classifyError converts raw errors into user-friendly messages with actionable next steps
func (a *OllamaAgent) classifyError(err error, statusCode int, model string) error {
	// Check for timeout
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("ollama request timed out. Try: 1) Use a smaller/faster model, 2) Increase timeout in config")
	}

	// Check for connection refused (server not running)
	errMsg := err.Error()
	if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "connect: connection refused") {
		return fmt.Errorf("ollama server not reachable at %s. Is Ollama running? Start with: ollama serve", a.BaseURL)
	}

	// Handle HTTP status codes
	switch statusCode {
	case 404:
		return fmt.Errorf("model %q not found. Pull it with: ollama pull %s\nList available models: ollama list", model, model)
	case 500, 502, 503, 504:
		return fmt.Errorf("ollama server error (status %d): %s\nCheck Ollama logs: journalctl -u ollama (Linux) or check console output", statusCode, errMsg)
	}

	// Return original error with context
	if statusCode > 0 {
		return fmt.Errorf("ollama request failed (status %d): %w", statusCode, err)
	}
	return fmt.Errorf("ollama request failed: %w", err)
}

// validateOllamaModel checks if a model name is valid
func validateOllamaModel(model string) error {
	if model == "" {
		return fmt.Errorf("model name cannot be empty")
	}
	// Ollama model names are alphanumeric with optional : - _ . / characters
	// Examples: llama3, qwen2.5-coder:latest, mistral:7b-instruct, library/llama3:latest
	for i, r := range model {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		if r == ':' || r == '-' || r == '_' || r == '.' || r == '/' {
			continue
		}
		return fmt.Errorf("invalid model name %q: contains invalid character %q at position %d", model, r, i)
	}
	return nil
}

// ResolveOllamaBaseURL determines the Ollama base URL from config or environment.
// Priority (highest to lowest):
// 1. Config TOML: cfg.OllamaBaseURL (if cfg != nil and non-empty)
// 2. OLLAMA_HOST environment variable
// 3. Default: http://localhost:11434
func ResolveOllamaBaseURL(cfg interface{}) string {
	// Check if config provides base URL
	type ollamaConfigGetter interface {
		GetOllamaBaseURL() string
	}
	if c, ok := cfg.(ollamaConfigGetter); ok {
		if u := c.GetOllamaBaseURL(); u != "" {
			return u
		}
	}

	// Check environment variable
	if envURL := os.Getenv("OLLAMA_HOST"); envURL != "" {
		return normalizeOllamaURL(envURL)
	}

	// Default
	return "http://localhost:11434"
}

// normalizeOllamaURL ensures a URL has a scheme, preventing malformed URLs
// like "myserver/api/chat" when users set OLLAMA_HOST=myserver.
func normalizeOllamaURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	// If no scheme, assume http
	if parsed.Scheme == "" {
		return "http://" + rawURL
	}
	return rawURL
}

// CommandLine returns a representative command line for this agent.
// Ollama uses an HTTP API rather than a CLI binary.
func (a *OllamaAgent) CommandLine() string {
	model := a.Model
	if model == "" {
		model = defaultOllamaModel
	}
	return fmt.Sprintf("ollama run %s (via %s)", model, a.BaseURL)
}

func init() {
	Register(NewOllamaAgent(""))
}

// WithReasoning returns a copy of the agent with the specified reasoning level
func (a *OllamaAgent) WithReasoning(level ReasoningLevel) Agent {
	return &OllamaAgent{
		BaseURL:    a.BaseURL,
		Model:      a.Model,
		Reasoning:  level,
		Agentic:    a.Agentic,
		HTTPClient: a.HTTPClient,
	}
}

// WithAgentic returns a copy of the agent configured for agentic mode
func (a *OllamaAgent) WithAgentic(agentic bool) Agent {
	return &OllamaAgent{
		BaseURL:    a.BaseURL,
		Model:      a.Model,
		Reasoning:  a.Reasoning,
		Agentic:    agentic,
		HTTPClient: a.HTTPClient,
	}
}

// WithModel returns a copy of the agent configured to use the specified model
func (a *OllamaAgent) WithModel(model string) Agent {
	if model == "" {
		return a
	}
	return &OllamaAgent{
		BaseURL:    a.BaseURL,
		Model:      model,
		Reasoning:  a.Reasoning,
		Agentic:    a.Agentic,
		HTTPClient: a.HTTPClient,
	}
}
