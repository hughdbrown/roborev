package agent

import (
	"testing"
)

func TestAgentRegistry(t *testing.T) {
	// Check that default agents are registered
	codex, err := Get("codex")
	if err != nil {
		t.Fatalf("Failed to get codex agent: %v", err)
	}
	if codex.Name() != "codex" {
		t.Errorf("Expected name 'codex', got '%s'", codex.Name())
	}

	claude, err := Get("claude-code")
	if err != nil {
		t.Fatalf("Failed to get claude-code agent: %v", err)
	}
	if claude.Name() != "claude-code" {
		t.Errorf("Expected name 'claude-code', got '%s'", claude.Name())
	}

	// Check unknown agent
	_, err = Get("unknown-agent")
	if err == nil {
		t.Error("Expected error for unknown agent")
	}
}

func TestAvailableAgents(t *testing.T) {
	agents := Available()
	if len(agents) < 2 {
		t.Errorf("Expected at least 2 agents, got %d", len(agents))
	}

	hasCodex := false
	hasClaude := false
	for _, a := range agents {
		if a == "codex" {
			hasCodex = true
		}
		if a == "claude-code" {
			hasClaude = true
		}
	}

	if !hasCodex {
		t.Error("Expected codex in available agents")
	}
	if !hasClaude {
		t.Error("Expected claude-code in available agents")
	}
}
