package agent

import (
	"context"
	"fmt"
	"time"
)

// TestAgent is a mock agent for testing that returns predictable output
type TestAgent struct {
	Delay  time.Duration // Simulated processing delay
	Output string        // Fixed output to return
	Fail   bool          // If true, returns an error
}

// NewTestAgent creates a new test agent
func NewTestAgent() *TestAgent {
	return &TestAgent{
		Delay:  100 * time.Millisecond,
		Output: "Test review output: This commit looks good. No issues found.",
	}
}

func (a *TestAgent) Name() string {
	return "test"
}

func (a *TestAgent) Review(ctx context.Context, repoPath, commitSHA, prompt string) (string, error) {
	// Respect context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(a.Delay):
	}

	if a.Fail {
		return "", fmt.Errorf("test agent configured to fail")
	}

	return fmt.Sprintf("%s\n\nCommit: %s\nRepo: %s", a.Output, commitSHA[:7], repoPath), nil
}

func init() {
	Register(NewTestAgent())
}
