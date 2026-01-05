package explain

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/session"
)

// Explain runs the discovery and filtering pipeline with full tracing
// and outputs a human-readable explanation
func Explain(commitRef string, w io.Writer) error {
	// Get repo root
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Create trace context
	trace := &session.TraceContext{}

	// For explain, we always simulate a normal (non-amend) commit
	isAmend := false

	// Calculate work period with tracing
	startWork, workTrace, err := git.CalculateWorkStartTimeWithTrace(isAmend)
	if err != nil {
		return fmt.Errorf("failed to calculate work period: %w", err)
	}

	// If examining a specific commit (not HEAD), adjust the end time
	endWork := time.Now().UTC()
	if commitRef != "HEAD" {
		commitTime, err := git.GetCommitTimestamp(commitRef)
		if err != nil {
			return fmt.Errorf("failed to get commit timestamp: %w", err)
		}
		endWork = commitTime
	}

	// Store work period trace
	trace.WorkPeriod = session.WorkPeriodTrace{
		IsAmend:             workTrace.IsAmend,
		Ref:                 workTrace.Ref,
		PrevCommitTimestamp: workTrace.PrevCommitTimestamp,
		BranchSwitchTime:    workTrace.BranchSwitchTime,
		CalculatedStart:     workTrace.CalculatedStart,
		EndWork:             endWork,
		Explanation:         workTrace.Explanation,
	}

	// Discover sessions with tracing
	sessions, err := session.FindSessions(repoRoot, trace)
	if err != nil {
		fmt.Fprintf(w, "Warning: %v\n", err)
		sessions = nil
	}

	// Filter by time with tracing
	sessions = session.FilterSessionsByTime(sessions, startWork, endWork, trace)

	// Filter by user messages with tracing
	_ = session.FilterSessionsByUserMessages(sessions, startWork, endWork, trace)

	// Output the explanation
	return renderExplanation(trace, w)
}

func renderExplanation(trace *session.TraceContext, w io.Writer) error {
	// Header
	fmt.Fprintln(w, "=== Session Discovery ===")
	fmt.Fprintln(w)

	// Session directory info
	fmt.Fprintf(w, "Repository: %s\n", trace.RepoPath)
	fmt.Fprintf(w, "Session directory: %s\n", trace.SessionDir)

	// Show shortened path for readability
	shortPath := trace.SessionDir
	if strings.HasPrefix(shortPath, trace.RepoPath) {
		shortPath = "~/.claude/projects/" + trace.EncodedPath
	}
	if shortPath != trace.SessionDir {
		fmt.Fprintf(w, "  (%s)\n", shortPath)
	}

	if trace.SessionDirExists {
		fmt.Fprintf(w, "  Status: exists, found %d session file(s)\n", len(trace.FoundFiles))
	} else {
		fmt.Fprintln(w, "  Status: directory does not exist")
	}
	fmt.Fprintln(w)

	// Work period explanation
	fmt.Fprintln(w, "=== Work Period ===")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Reference: %s\n", trace.WorkPeriod.Ref)
	if !trace.WorkPeriod.PrevCommitTimestamp.IsZero() {
		fmt.Fprintf(w, "Previous commit: %s\n",
			trace.WorkPeriod.PrevCommitTimestamp.Local().Format("2006-01-02 15:04:05"))
	} else {
		fmt.Fprintln(w, "Previous commit: (none - initial commit)")
	}
	if !trace.WorkPeriod.BranchSwitchTime.IsZero() {
		fmt.Fprintf(w, "Branch switch: %s\n",
			trace.WorkPeriod.BranchSwitchTime.Local().Format("2006-01-02 15:04:05"))
	} else {
		fmt.Fprintln(w, "Branch switch: (none found)")
	}
	fmt.Fprintf(w, "Result: %s\n", trace.WorkPeriod.Explanation)

	startStr := "(none)"
	if !trace.WorkPeriod.CalculatedStart.IsZero() {
		startStr = trace.WorkPeriod.CalculatedStart.Local().Format("2006-01-02 15:04:05")
	}
	fmt.Fprintf(w, "Work period: %s → %s\n",
		startStr,
		trace.WorkPeriod.EndWork.Local().Format("2006-01-02 15:04:05"))
	fmt.Fprintln(w)

	// Per-session decisions
	fmt.Fprintln(w, "=== Sessions ===")
	fmt.Fprintln(w)

	if len(trace.Sessions) == 0 {
		fmt.Fprintln(w, "No sessions found.")
		fmt.Fprintln(w)
		printSummary(trace, w)
		return nil
	}

	included := 0
	for _, s := range trace.Sessions {
		// Truncate ID for readability (show first 8 chars or full if shorter)
		displayID := s.ID
		if len(displayID) > 36 {
			displayID = displayID[:8] + "..."
		}
		fmt.Fprintf(w, "%s\n", displayID)
		fmt.Fprintf(w, "  Created:  %s\n", s.Created.Local().Format("2006-01-02 15:04:05"))
		fmt.Fprintf(w, "  Modified: %s\n", s.Modified.Local().Format("2006-01-02 15:04:05"))

		// Time filter result
		fmt.Fprintf(w, "  Time filter: %s\n", s.TimeFilterReason)

		// User message filter result (only if passed time filter)
		if s.TimeFilterPassed {
			msgInfo := s.UserMsgReason
			if s.UserMsgCount > 0 {
				msgInfo = fmt.Sprintf("%s (%d messages in range)", s.UserMsgReason, s.UserMsgCount)
			}
			fmt.Fprintf(w, "  User messages: %s\n", msgInfo)
		}

		// Final decision with arrow indicator
		if s.Included {
			fmt.Fprintln(w, "  → INCLUDED")
			included++
		} else {
			fmt.Fprintln(w, "  → EXCLUDED")
		}
		fmt.Fprintln(w)
	}

	printSummary(trace, w)
	return nil
}

func printSummary(trace *session.TraceContext, w io.Writer) {
	included := 0
	for _, s := range trace.Sessions {
		if s.Included {
			included++
		}
	}

	fmt.Fprintln(w, "=== Summary ===")
	fmt.Fprintf(w, "Found: %d session(s)\n", len(trace.Sessions))
	fmt.Fprintf(w, "Included: %d session(s)\n", included)
}
