package agent

import (
	"context"
	"fmt"
)

// Agent defines the interface for code review agents
type Agent interface {
	// Name returns the agent identifier (e.g., "codex", "claude-code")
	Name() string

	// Review runs a code review and returns the output
	Review(ctx context.Context, repoPath, commitSHA, prompt string) (output string, err error)
}

// Registry holds available agents
var registry = make(map[string]Agent)

// Register adds an agent to the registry
func Register(a Agent) {
	registry[a.Name()] = a
}

// Get returns an agent by name
func Get(name string) (Agent, error) {
	a, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown agent: %s", name)
	}
	return a, nil
}

// Available returns the names of all registered agents
func Available() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
