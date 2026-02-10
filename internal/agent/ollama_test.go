package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOllamaAgent_Name(t *testing.T) {
	agent := NewOllamaAgent("")
	if got := agent.Name(); got != "ollama" {
		t.Errorf("Name() = %q, want %q", got, "ollama")
	}
}

func TestOllamaAgent_WithModel(t *testing.T) {
	agent := NewOllamaAgent("")
	agent2 := agent.WithModel("llama3:70b")

	// Verify model is set
	ollama2 := agent2.(*OllamaAgent)
	if ollama2.Model != "llama3:70b" {
		t.Errorf("WithModel() model = %q, want %q", ollama2.Model, "llama3:70b")
	}

	// Verify original is unchanged
	if agent.Model != "" {
		t.Errorf("Original agent was modified, model = %q", agent.Model)
	}

	// Verify empty model returns same agent instance
	agent3 := agent.WithModel("")
	if agent3 != agent {
		t.Error("WithModel(\"\") should return same agent")
	}
}

func TestOllamaAgent_WithReasoning(t *testing.T) {
	agent := NewOllamaAgent("")
	agent2 := agent.WithReasoning(ReasoningThorough)

	ollama2 := agent2.(*OllamaAgent)
	if ollama2.Reasoning != ReasoningThorough {
		t.Errorf("WithReasoning() reasoning = %v, want %v", ollama2.Reasoning, ReasoningThorough)
	}

	if agent.Reasoning != ReasoningStandard {
		t.Errorf("Original agent was modified, reasoning = %v", agent.Reasoning)
	}
}

func TestOllamaAgent_WithAgentic(t *testing.T) {
	agent := NewOllamaAgent("")
	agent2 := agent.WithAgentic(true)

	ollama2 := agent2.(*OllamaAgent)
	if !ollama2.Agentic {
		t.Error("WithAgentic(true) did not set agentic flag")
	}

	if agent.Agentic {
		t.Error("Original agent was modified")
	}
}

func TestOllamaAgent_Review_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("Expected path /api/chat, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Send streaming chat response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		responses := []string{
			`{"message":{"role":"assistant","content":"This "},"done":false}`,
			`{"message":{"role":"assistant","content":"is "},"done":false}`,
			`{"message":{"role":"assistant","content":"a test"},"done":false}`,
			`{"message":{"role":"assistant","content":""},"done":true}`,
		}
		for _, resp := range responses {
			w.Write([]byte(resp + "\n"))
		}
	}))
	defer server.Close()

	agent := NewOllamaAgent(server.URL)
	result, err := agent.Review(context.Background(), "/tmp", "abc123", "Review this code", nil)
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	expected := "This is a test"
	if result != expected {
		t.Errorf("Review() result = %q, want %q", result, expected)
	}
}

func TestOllamaAgent_Review_ModelNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"error":"model not found"}`))
	}))
	defer server.Close()

	agent := NewOllamaAgent(server.URL)
	agent = agent.WithModel("nonexistent:model").(*OllamaAgent)

	_, err := agent.Review(context.Background(), "/tmp", "abc123", "test", nil)
	if err == nil {
		t.Fatal("Expected error for 404 response")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "not found") {
		t.Errorf("Error should mention 'not found', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "ollama pull") {
		t.Errorf("Error should suggest 'ollama pull', got: %s", errMsg)
	}
}

func TestOllamaAgent_Review_ServerUnreachable(t *testing.T) {
	// Use invalid URL that will cause connection refused
	agent := NewOllamaAgent("http://localhost:1")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := agent.Review(ctx, "/tmp", "abc123", "test", nil)
	if err == nil {
		t.Fatal("Expected error for unreachable server")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "not reachable") && !strings.Contains(errMsg, "connection refused") {
		t.Errorf("Error should mention connection issue, got: %s", errMsg)
	}
}

func TestOllamaAgent_Review_Timeout(t *testing.T) {
	// Create server that never responds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	agent := NewOllamaAgent(server.URL)
	// Use custom HTTP client with short timeout
	agent.HTTPClient = &http.Client{Timeout: 100 * time.Millisecond}

	_, err := agent.Review(context.Background(), "/tmp", "abc123", "test", nil)
	if err == nil {
		t.Fatal("Expected timeout error")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "timed out") && !strings.Contains(errMsg, "timeout") {
		t.Errorf("Error should mention timeout, got: %s", errMsg)
	}
}

func TestOllamaAgent_Review_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		// Send done=true with no response text
		w.Write([]byte(`{"message":{"role":"assistant","content":""},"done":true}` + "\n"))
	}))
	defer server.Close()

	agent := NewOllamaAgent(server.URL)
	result, err := agent.Review(context.Background(), "/tmp", "abc123", "test", nil)
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	if result != "No review output generated" {
		t.Errorf("Review() result = %q, want %q", result, "No review output generated")
	}
}

func TestOllamaAgent_Review_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		// Send mix of valid and invalid JSON
		w.Write([]byte(`{"message":{"role":"assistant","content":"Good "},"done":false}` + "\n"))
		w.Write([]byte(`{invalid json}` + "\n")) // Should be skipped
		w.Write([]byte(`{"message":{"role":"assistant","content":"line"},"done":false}` + "\n"))
		w.Write([]byte(`{"message":{"role":"assistant","content":""},"done":true}` + "\n"))
	}))
	defer server.Close()

	agent := NewOllamaAgent(server.URL)
	result, err := agent.Review(context.Background(), "/tmp", "abc123", "test", nil)
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	// Should skip invalid JSON and concatenate valid responses
	expected := "Good line"
	if result != expected {
		t.Errorf("Review() result = %q, want %q", result, expected)
	}
}

func TestOllamaAgent_Review_StreamingOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"message":{"role":"assistant","content":"chunk1 "},"done":false}` + "\n"))
		w.Write([]byte(`{"message":{"role":"assistant","content":"chunk2"},"done":false}` + "\n"))
		w.Write([]byte(`{"message":{"role":"assistant","content":""},"done":true}` + "\n"))
	}))
	defer server.Close()

	agent := NewOllamaAgent(server.URL)
	var outputBuf strings.Builder
	result, err := agent.Review(context.Background(), "/tmp", "abc123", "test", &outputBuf)
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	// Verify result
	expected := "chunk1 chunk2"
	if result != expected {
		t.Errorf("Review() result = %q, want %q", result, expected)
	}

	// Verify output was streamed
	streamed := outputBuf.String()
	if streamed != expected {
		t.Errorf("Streamed output = %q, want %q", streamed, expected)
	}
}

func TestOllamaAgent_augmentPromptForAgentic(t *testing.T) {
	tests := []struct {
		name     string
		agentic  bool
		prompt   string
		contains []string
	}{
		{
			name:     "agentic disabled returns prompt unchanged",
			agentic:  false,
			prompt:   "Review this code",
			contains: []string{"Review this code"},
		},
		{
			name:    "agentic enabled adds tool descriptions",
			agentic: true,
			prompt:  "Review this code",
			contains: []string{
				"Review this code",
				"read_file",
				"write_file",
				"run_command",
				"You have access to the following tools",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &OllamaAgent{
				Agentic: tt.agentic,
			}
			result := agent.augmentPromptForAgentic(tt.prompt)

			// Check that result contains all expected strings
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("augmentPromptForAgentic() result missing %q", expected)
				}
			}

			// If agentic disabled, result should be exactly the prompt
			if !tt.agentic && result != tt.prompt {
				t.Errorf("augmentPromptForAgentic() with agentic=false should return prompt unchanged, got %q", result)
			}

			// If agentic enabled, result should be longer than original prompt
			if tt.agentic && len(result) <= len(tt.prompt) {
				t.Errorf("augmentPromptForAgentic() with agentic=true should augment prompt, but result is not longer")
			}
		})
	}
}

func TestOllamaAgent_buildRequest(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		reasoning ReasoningLevel
		wantModel string
		wantTemp  float64
		wantTopP  float64
	}{
		{
			name:      "default model and reasoning",
			model:     "",
			reasoning: ReasoningStandard,
			wantModel: "qwen2.5-coder:latest",
			wantTemp:  0.7,
			wantTopP:  0.95,
		},
		{
			name:      "custom model",
			model:     "llama3:70b",
			reasoning: ReasoningStandard,
			wantModel: "llama3:70b",
			wantTemp:  0.7,
			wantTopP:  0.95,
		},
		{
			name:      "thorough reasoning",
			model:     "",
			reasoning: ReasoningThorough,
			wantModel: "qwen2.5-coder:latest",
			wantTemp:  0.3,
			wantTopP:  0.9,
		},
		{
			name:      "fast reasoning",
			model:     "",
			reasoning: ReasoningFast,
			wantModel: "qwen2.5-coder:latest",
			wantTemp:  1.0,
			wantTopP:  1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &OllamaAgent{
				Model:     tt.model,
				Reasoning: tt.reasoning,
			}
			req := agent.buildRequest("test prompt")

			if req.Model != tt.wantModel {
				t.Errorf("buildRequest().Model = %q, want %q", req.Model, tt.wantModel)
			}
			if !req.Stream {
				t.Error("buildRequest().Stream = false, want true")
			}

			// Check messages structure
			if len(req.Messages) != 2 {
				t.Errorf("buildRequest().Messages length = %d, want 2", len(req.Messages))
			}
			if req.Messages[0].Role != "system" {
				t.Errorf("buildRequest().Messages[0].Role = %q, want \"system\"", req.Messages[0].Role)
			}
			if req.Messages[1].Role != "user" {
				t.Errorf("buildRequest().Messages[1].Role = %q, want \"user\"", req.Messages[1].Role)
			}
			if req.Messages[1].Content != "test prompt" {
				t.Errorf("buildRequest().Messages[1].Content = %q, want \"test prompt\"", req.Messages[1].Content)
			}

			if temp, ok := req.Options["temperature"].(float64); !ok || temp != tt.wantTemp {
				t.Errorf("buildRequest().Options[temperature] = %v, want %v", req.Options["temperature"], tt.wantTemp)
			}
			if topP, ok := req.Options["top_p"].(float64); !ok || topP != tt.wantTopP {
				t.Errorf("buildRequest().Options[top_p] = %v, want %v", req.Options["top_p"], tt.wantTopP)
			}
		})
	}
}
