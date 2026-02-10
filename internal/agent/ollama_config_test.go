package agent

import (
	"os"
	"testing"
)

// mockConfig implements the interface needed by ResolveOllamaBaseURL
type mockConfig struct {
	ollamaBaseURL string
}

func (m *mockConfig) GetOllamaBaseURL() string {
	return m.ollamaBaseURL
}

func TestResolveOllamaBaseURL_ConfigTOML(t *testing.T) {
	cfg := &mockConfig{ollamaBaseURL: "http://config:8080"}
	os.Setenv("OLLAMA_HOST", "http://env:9999")
	defer os.Unsetenv("OLLAMA_HOST")

	url := ResolveOllamaBaseURL(cfg)
	expected := "http://config:8080"
	if url != expected {
		t.Errorf("Config TOML should have highest priority, got %q, want %q", url, expected)
	}
}

func TestResolveOllamaBaseURL_EnvVar(t *testing.T) {
	cfg := &mockConfig{ollamaBaseURL: ""}
	os.Setenv("OLLAMA_HOST", "http://env:9999")
	defer os.Unsetenv("OLLAMA_HOST")

	url := ResolveOllamaBaseURL(cfg)
	expected := "http://env:9999"
	if url != expected {
		t.Errorf("ResolveOllamaBaseURL() = %q, want %q", url, expected)
	}
}

func TestResolveOllamaBaseURL_Default(t *testing.T) {
	cfg := &mockConfig{ollamaBaseURL: ""}
	os.Unsetenv("OLLAMA_HOST")

	url := ResolveOllamaBaseURL(cfg)
	expected := "http://localhost:11434"
	if url != expected {
		t.Errorf("ResolveOllamaBaseURL() = %q, want %q", url, expected)
	}
}

func TestResolveOllamaBaseURL_NilConfig(t *testing.T) {
	os.Setenv("OLLAMA_HOST", "http://env:9999")
	defer os.Unsetenv("OLLAMA_HOST")

	url := ResolveOllamaBaseURL(nil)
	expected := "http://env:9999"
	if url != expected {
		t.Errorf("Nil config should fall through to env var, got %q, want %q", url, expected)
	}
}

func TestResolveOllamaBaseURL_Priority(t *testing.T) {
	// All sources set - config should win
	cfg := &mockConfig{ollamaBaseURL: "http://config:7777"}
	os.Setenv("OLLAMA_HOST", "http://env:8888")
	defer os.Unsetenv("OLLAMA_HOST")

	url := ResolveOllamaBaseURL(cfg)
	expected := "http://config:7777"
	if url != expected {
		t.Errorf("Config TOML should override env var, got %q, want %q", url, expected)
	}
}

func TestValidateOllamaModel_Valid(t *testing.T) {
	validModels := []string{
		"llama3",
		"qwen2.5-coder:latest",
		"mistral:7b-instruct",
		"deepseek-coder:6.7b",
		"codellama:13b-python",
		"phi3:mini",
	}

	for _, model := range validModels {
		if err := validateOllamaModel(model); err != nil {
			t.Errorf("validateOllamaModel(%q) returned error: %v", model, err)
		}
	}
}

func TestValidateOllamaModel_Invalid(t *testing.T) {
	tests := []struct {
		model       string
		shouldError bool
	}{
		{"", true},                     // empty
		{"model with spaces", true},    // spaces
		{"model/with/slash", true},     // slash
		{"model@version", true},        // @ symbol
		{"llama3:latest", false},       // valid
		{"model_name-v1.0:tag", false}, // valid with multiple special chars
	}

	for _, tt := range tests {
		err := validateOllamaModel(tt.model)
		if tt.shouldError && err == nil {
			t.Errorf("validateOllamaModel(%q) should return error, got nil", tt.model)
		}
		if !tt.shouldError && err != nil {
			t.Errorf("validateOllamaModel(%q) should not return error, got: %v", tt.model, err)
		}
	}
}
