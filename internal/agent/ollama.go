package agent

import (
	"context"
	"io"
	"net/http"
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

// Name returns the agent identifier
func (a *OllamaAgent) Name() string {
	return "ollama"
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
