package agent

import (
	"context"
	"io"
	"net/http"
	"time"
)

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
		model = "qwen2.5-coder:latest"
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

// getHTTPClient returns the HTTP client for requests
func (a *OllamaAgent) getHTTPClient() *http.Client {
	if a.HTTPClient != nil {
		return a.HTTPClient
	}
	return &http.Client{
		Timeout: 5 * time.Minute,
	}
}

// Review runs a code review and returns the output
func (a *OllamaAgent) Review(ctx context.Context, repoPath, commitSHA, prompt string, output io.Writer) (string, error) {
	// TODO: Implement in Task 1.4
	return "", nil
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
