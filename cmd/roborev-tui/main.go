package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/roborev/internal/storage"
)

var (
	serverAddr = "http://127.0.0.1:7373"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212"))

	queuedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))  // Yellow
	runningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))   // Blue
	doneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))   // Green
	failedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))  // Red

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)
)

type view int

const (
	viewQueue view = iota
	viewReview
)

type model struct {
	jobs         []storage.ReviewJob
	status       storage.DaemonStatus
	selectedIdx  int
	currentView  view
	currentReview *storage.Review
	reviewScroll int
	width        int
	height       int
	err          error
}

type tickMsg time.Time
type jobsMsg []storage.ReviewJob
type statusMsg storage.DaemonStatus
type reviewMsg *storage.Review
type errMsg error

func initialModel() model {
	return model{
		jobs:        []storage.ReviewJob{},
		currentView: viewQueue,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tick(), fetchJobs(), fetchStatus())
}

func tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchJobs() tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Get(serverAddr + "/api/jobs?limit=50")
		if err != nil {
			return errMsg(err)
		}
		defer resp.Body.Close()

		var result struct {
			Jobs []storage.ReviewJob `json:"jobs"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return errMsg(err)
		}
		return jobsMsg(result.Jobs)
	}
}

func fetchStatus() tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Get(serverAddr + "/api/status")
		if err != nil {
			return errMsg(err)
		}
		defer resp.Body.Close()

		var status storage.DaemonStatus
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			return errMsg(err)
		}
		return statusMsg(status)
	}
}

func fetchReview(sha string) tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Get(serverAddr + "/api/review?sha=" + sha)
		if err != nil {
			return errMsg(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			return errMsg(fmt.Errorf("no review found"))
		}

		var review storage.Review
		if err := json.NewDecoder(resp.Body).Decode(&review); err != nil {
			return errMsg(err)
		}
		return reviewMsg(&review)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.currentView == viewReview {
				m.currentView = viewQueue
				m.currentReview = nil
				m.reviewScroll = 0
				return m, nil
			}
			return m, tea.Quit

		case "up", "k":
			if m.currentView == viewQueue {
				if m.selectedIdx > 0 {
					m.selectedIdx--
				}
			} else {
				if m.reviewScroll > 0 {
					m.reviewScroll--
				}
			}

		case "down", "j":
			if m.currentView == viewQueue {
				if m.selectedIdx < len(m.jobs)-1 {
					m.selectedIdx++
				}
			} else {
				m.reviewScroll++
			}

		case "enter":
			if m.currentView == viewQueue && len(m.jobs) > 0 {
				job := m.jobs[m.selectedIdx]
				if job.Status == storage.JobStatusDone {
					return m, fetchReview(job.CommitSHA)
				}
			}

		case "esc":
			if m.currentView == viewReview {
				m.currentView = viewQueue
				m.currentReview = nil
				m.reviewScroll = 0
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		return m, tea.Batch(tick(), fetchJobs(), fetchStatus())

	case jobsMsg:
		m.jobs = msg
		if m.selectedIdx >= len(m.jobs) {
			m.selectedIdx = max(0, len(m.jobs)-1)
		}

	case statusMsg:
		m.status = storage.DaemonStatus(msg)

	case reviewMsg:
		m.currentReview = msg
		m.currentView = viewReview
		m.reviewScroll = 0

	case errMsg:
		m.err = msg
	}

	return m, nil
}

func (m model) View() string {
	if m.currentView == viewReview && m.currentReview != nil {
		return m.renderReviewView()
	}
	return m.renderQueueView()
}

func (m model) renderQueueView() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("RoboRev Queue"))
	b.WriteString("\n")

	// Status line
	statusLine := fmt.Sprintf("Workers: %d/%d | Queued: %d | Running: %d | Done: %d | Failed: %d",
		m.status.ActiveWorkers, m.status.MaxWorkers,
		m.status.QueuedJobs, m.status.RunningJobs,
		m.status.CompletedJobs, m.status.FailedJobs)
	b.WriteString(statusStyle.Render(statusLine))
	b.WriteString("\n\n")

	if len(m.jobs) == 0 {
		b.WriteString("No jobs in queue\n")
	} else {
		// Header
		header := fmt.Sprintf("%-4s %-7s %-15s %-12s %-8s %s",
			"ID", "SHA", "Repo", "Agent", "Status", "Time")
		b.WriteString(statusStyle.Render(header))
		b.WriteString("\n")
		b.WriteString(strings.Repeat("─", min(m.width-2, 70)))
		b.WriteString("\n")

		// Jobs
		for i, job := range m.jobs {
			line := m.renderJobLine(job)
			if i == m.selectedIdx {
				line = selectedStyle.Render("▶ " + line)
			} else {
				line = "  " + line
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Help
	b.WriteString(helpStyle.Render("↑/↓: navigate | enter: view review | q: quit"))

	return b.String()
}

func (m model) renderJobLine(job storage.ReviewJob) string {
	sha := job.CommitSHA
	if len(sha) > 7 {
		sha = sha[:7]
	}

	repo := job.RepoName
	if len(repo) > 15 {
		repo = repo[:12] + "..."
	}

	agent := job.Agent
	if len(agent) > 12 {
		agent = agent[:12]
	}

	elapsed := ""
	if job.StartedAt != nil {
		if job.FinishedAt != nil {
			elapsed = job.FinishedAt.Sub(*job.StartedAt).Round(time.Second).String()
		} else {
			elapsed = time.Since(*job.StartedAt).Round(time.Second).String()
		}
	}

	status := string(job.Status)
	var styledStatus string
	switch job.Status {
	case storage.JobStatusQueued:
		styledStatus = queuedStyle.Render(status)
	case storage.JobStatusRunning:
		styledStatus = runningStyle.Render(status)
	case storage.JobStatusDone:
		styledStatus = doneStyle.Render(status)
	case storage.JobStatusFailed:
		styledStatus = failedStyle.Render(status)
	default:
		styledStatus = status
	}

	return fmt.Sprintf("%-4d %-7s %-15s %-12s %-8s %s",
		job.ID, sha, repo, agent, styledStatus, elapsed)
}

func (m model) renderReviewView() string {
	var b strings.Builder

	review := m.currentReview
	if review.Job != nil {
		sha := review.Job.CommitSHA
		if len(sha) > 7 {
			sha = sha[:7]
		}
		title := fmt.Sprintf("Review: %s (%s)", sha, review.Agent)
		b.WriteString(titleStyle.Render(title))
	} else {
		b.WriteString(titleStyle.Render("Review"))
	}
	b.WriteString("\n")

	// Split output into lines and handle scrolling
	lines := strings.Split(review.Output, "\n")
	visibleLines := m.height - 5 // Leave room for title and help

	start := m.reviewScroll
	if start >= len(lines) {
		start = max(0, len(lines)-1)
	}
	end := min(start+visibleLines, len(lines))

	for i := start; i < end; i++ {
		b.WriteString(lines[i])
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(lines) > visibleLines {
		scrollInfo := fmt.Sprintf("[%d-%d of %d lines]", start+1, end, len(lines))
		b.WriteString(statusStyle.Render(scrollInfo))
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("↑/↓: scroll | esc/q: back"))

	return b.String()
}

func main() {
	if len(os.Args) > 1 {
		serverAddr = os.Args[1]
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
