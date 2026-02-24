package daemon

import (
	"fmt"
	"os"
	"testing"

	"github.com/roborev-dev/roborev/internal/testenv"
)

// TestMain isolates the entire daemon test package from the production
// ~/.roborev directory. Without this, NewServer creates activity/error
// logs at DefaultActivityLogPath() → ~/.roborev/activity.log, polluting
// the production log with test events and confusing running TUIs.
func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	// Snapshot prod log state BEFORE overriding ROBOREV_DATA_DIR.
	barrier := testenv.NewProdLogBarrier(
		testenv.DefaultProdDataDir(),
	)

	tmpDir, err := os.MkdirTemp("", "roborev-daemon-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"failed to create temp dir: %v\n", err)
		return 1
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("ROBOREV_DATA_DIR", tmpDir)
	code := m.Run()

	// Hard barrier: fail if tests polluted production logs.
	if msg := barrier.Check(); msg != "" {
		fmt.Fprintln(os.Stderr, msg)
		return 1
	}
	return code
}
