package explain

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/session"
)

// ExplainOptions configures the explain command
type ExplainOptions struct {
	ShowAll bool // Show all sessions including excluded ones
}

// Explain runs the discovery and filtering pipeline with full tracing
// and outputs a human-readable explanation.
// If showAll is true, every session is listed with full details.
// If showAll is false (default), excluded sessions are grouped by reason.
func Explain(commitRef string, opts ExplainOptions, w io.Writer) error {
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

	// Discover sessions with tracing (includes time filtering)
	sessions, err := session.FindSessions(repoRoot, startWork, endWork, trace)
	if err != nil {
		fmt.Fprintf(w, "Warning: %v\n", err)
		sessions = nil
	}

	// Filter by user messages with tracing
	_ = session.FilterSessionsByUserMessages(sessions, startWork, endWork, trace)

	// Output the explanation
	return renderExplanation(trace, opts.ShowAll, w)
}

func renderExplanation(trace *session.TraceContext, showAll bool, w io.Writer) error {
	// Header
	fmt.Fprintln(w, "=== Session Discovery ===")
	fmt.Fprintln(w)

	// Session directory info
	fmt.Fprintf(w, "Repository: %s\n", trace.RepoPath)

	// Show candidate directories
	if len(trace.CandidateDirs) > 0 {
		fmt.Fprintf(w, "Candidate directories: %d\n", len(trace.CandidateDirs))
		for _, dir := range trace.CandidateDirs {
			fmt.Fprintf(w, "  - %s\n", dir)
		}
	} else {
		fmt.Fprintf(w, "Session directory: %s\n", trace.SessionDir)

		// Show shortened path for readability
		shortPath := trace.SessionDir
		if strings.HasPrefix(shortPath, trace.RepoPath) {
			shortPath = "~/.claude/projects/" + trace.EncodedPath
		}
		if shortPath != trace.SessionDir {
			fmt.Fprintf(w, "  (%s)\n", shortPath)
		}
	}

	if trace.SessionDirExists {
		fmt.Fprintf(w, "Status: found %d session file(s)\n", len(trace.FoundFiles))
		if trace.SkippedByMtime > 0 {
			fmt.Fprintf(w, "  Skipped by mtime: %d (not modified in work period)\n", trace.SkippedByMtime)
		}
	} else {
		fmt.Fprintln(w, "Status: no session directories found")
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

	if len(trace.Sessions) == 0 {
		fmt.Fprintln(w, "=== Sessions ===")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "No sessions found.")
		fmt.Fprintln(w)
		printSummary(trace, w)
		return nil
	}

	// Separate included and excluded sessions
	var included, excluded []session.SessionTrace
	for _, s := range trace.Sessions {
		if s.Included {
			included = append(included, s)
		} else {
			excluded = append(excluded, s)
		}
	}

	// Show included sessions with full details
	fmt.Fprintf(w, "=== Included Sessions (%d) ===\n", len(included))
	fmt.Fprintln(w)

	if len(included) == 0 {
		fmt.Fprintln(w, "(none)")
		fmt.Fprintln(w)
	} else {
		for _, s := range included {
			printSessionDetails(s, w)
		}
	}

	// Show excluded sessions
	fmt.Fprintf(w, "=== Excluded Sessions (%d) ===\n", len(excluded))
	fmt.Fprintln(w)

	if len(excluded) == 0 {
		fmt.Fprintln(w, "(none)")
		fmt.Fprintln(w)
	} else if showAll {
		// Show all excluded sessions with full details
		for _, s := range excluded {
			printSessionDetails(s, w)
		}
	} else {
		// Group by reason
		reasonCounts := make(map[string]int)
		for _, s := range excluded {
			reason := s.FinalReason
			reasonCounts[reason]++
		}
		for reason, count := range reasonCounts {
			fmt.Fprintf(w, "%s: %d session(s)\n", reason, count)
		}
		fmt.Fprintln(w)
	}

	printSummary(trace, w)
	return nil
}

func printSessionDetails(s session.SessionTrace, w io.Writer) {
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
		fmt.Fprintln(w, "  -> INCLUDED")
	} else {
		fmt.Fprintln(w, "  -> EXCLUDED")
	}
	fmt.Fprintln(w)
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
