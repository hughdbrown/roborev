package prompt

import (
	"fmt"
	"strings"

	"github.com/user/roborev/internal/git"
	"github.com/user/roborev/internal/storage"
)

// Builder constructs review prompts
type Builder struct {
	db *storage.DB
}

// NewBuilder creates a new prompt builder
func NewBuilder(db *storage.DB) *Builder {
	return &Builder{db: db}
}

// Build constructs a review prompt for a commit
func (b *Builder) Build(repoPath, sha string, repoID int64, contextCount int) (string, error) {
	var sb strings.Builder

	// Header
	sb.WriteString("# Code Review Request\n\n")

	// Get commit info
	info, err := git.GetCommitInfo(repoPath, sha)
	if err != nil {
		return "", fmt.Errorf("get commit info: %w", err)
	}

	sb.WriteString("## Commit Details\n\n")
	sb.WriteString(fmt.Sprintf("- **SHA**: %s\n", info.SHA))
	sb.WriteString(fmt.Sprintf("- **Author**: %s\n", info.Author))
	sb.WriteString(fmt.Sprintf("- **Subject**: %s\n", info.Subject))
	sb.WriteString(fmt.Sprintf("- **Date**: %s\n", info.Timestamp.Format("2006-01-02 15:04:05")))
	sb.WriteString("\n")

	// Get files changed
	files, err := git.GetFilesChanged(repoPath, sha)
	if err != nil {
		return "", fmt.Errorf("get files changed: %w", err)
	}

	sb.WriteString("## Files Changed\n\n")
	for _, f := range files {
		sb.WriteString(fmt.Sprintf("- %s\n", f))
	}
	sb.WriteString("\n")

	// Get the diff
	diff, err := git.GetDiff(repoPath, sha)
	if err != nil {
		return "", fmt.Errorf("get diff: %w", err)
	}

	sb.WriteString("## Diff\n\n```diff\n")
	sb.WriteString(diff)
	sb.WriteString("```\n\n")

	// Get recent reviews for context
	if contextCount > 0 && b.db != nil {
		reviews, err := b.db.GetRecentReviewsForRepo(repoID, contextCount)
		if err == nil && len(reviews) > 0 {
			sb.WriteString("## Recent Reviews (for context)\n\n")
			for i, r := range reviews {
				sb.WriteString(fmt.Sprintf("### Review %d (by %s)\n\n", i+1, r.Agent))
				// Truncate long outputs for context
				output := r.Output
				if len(output) > 500 {
					output = output[:500] + "...(truncated)"
				}
				sb.WriteString(output)
				sb.WriteString("\n\n")
			}
		}
	}

	// Review instructions
	sb.WriteString("## Review Instructions\n\n")
	sb.WriteString("Please review this commit for:\n\n")
	sb.WriteString("1. **Correctness**: Logic errors, bugs, edge cases not handled\n")
	sb.WriteString("2. **Behavior Regressions**: Changes that might break existing functionality\n")
	sb.WriteString("3. **Testing Gaps**: Missing tests, especially end-to-end tests for frontend changes\n")
	sb.WriteString("4. **Security Issues**: Potential vulnerabilities (injection, XSS, etc.)\n")
	sb.WriteString("5. **Performance**: Obvious performance problems or improvements\n\n")
	sb.WriteString("Focus on substantive issues. Don't comment on style unless it impacts readability significantly.\n\n")
	sb.WriteString("If the commit looks good, say so briefly. If there are issues, be specific about what and where.\n")

	return sb.String(), nil
}

// BuildSimple constructs a simpler prompt without database context
func BuildSimple(repoPath, sha string) (string, error) {
	b := &Builder{}
	return b.Build(repoPath, sha, 0, 0)
}
