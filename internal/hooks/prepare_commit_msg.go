package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/note"
	"github.com/QuesmaOrg/git-prompt-story/internal/parser"
	"github.com/QuesmaOrg/git-prompt-story/internal/provider"
	"github.com/QuesmaOrg/git-prompt-story/internal/scrubber"

	// Import providers and parsers to register them
	_ "github.com/QuesmaOrg/git-prompt-story/internal/parser/claude"
	_ "github.com/QuesmaOrg/git-prompt-story/internal/parser/cursor"
	_ "github.com/QuesmaOrg/git-prompt-story/internal/provider/claude"
	_ "github.com/QuesmaOrg/git-prompt-story/internal/provider/cursor"
)

// PrepareCommitMsg implements the prepare-commit-msg hook logic
func PrepareCommitMsg(msgFile, source, sha, version string) error {
	// Get repo root
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Get git directory for debug log
	gitDir, err := git.GetGitDir()
	if err != nil {
		return err
	}
	debugLog := newDebugLogger(filepath.Join(gitDir, "prompt-story-debug.log"))
	debugLog.log("=== prepare-commit-msg started at %s ===", time.Now().UTC().Format(time.RFC3339))
	debugLog.log("repoRoot: %s", repoRoot)
	debugLog.log("msgFile: %s, source: %q, sha: %q", msgFile, source, sha)

	// Read current commit message to detect if this is an amend
	msgContent, err := os.ReadFile(msgFile)
	if err != nil {
		return fmt.Errorf("failed to read commit message: %w", err)
	}

	// Detect amend: if message already has our marker, or source is "commit" with SHA
	// Note: git passes source="message" when using -m flag, even with --amend
	// So we also check for existing marker as a reliable amend indicator
	hasMarker := strings.Contains(string(msgContent), "Prompt-Story:")
	isAmend := (source == "commit" && sha != "") || hasMarker
	debugLog.log("isAmend: %v (source=commit&&sha: %v, hasMarker: %v)", isAmend, source == "commit" && sha != "", hasMarker)

	// Calculate work period
	startWork, _ := git.CalculateWorkStartTime(isAmend)
	endWork := time.Now().UTC()
	debugLog.log("Work period: %s - %s (now)", startWork.UTC().Format(time.RFC3339), endWork.Format(time.RFC3339))

	// Discover sessions from all registered providers
	var allSessions []provider.RawSession
	for _, p := range provider.All() {
		debugLog.log("Discovering sessions from %s...", p.Name())
		sessions, err := p.DiscoverSessions(repoRoot, startWork, endWork)
		if err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: warning: %s discovery failed: %v\n", p.Name(), err)
			debugLog.log("%s discovery error: %v", p.Name(), err)
			continue
		}
		debugLog.log("%s found %d sessions", p.Name(), len(sessions))
		for _, s := range sessions {
			debugLog.log("  - %s: created=%s, modified=%s", s.ID, s.Created.UTC().Format(time.RFC3339), s.Modified.UTC().Format(time.RFC3339))
		}
		allSessions = append(allSessions, sessions...)
	}

	debugLog.log("Total sessions from all providers: %d", len(allSessions))

	pendingFile := filepath.Join(gitDir, "PENDING-PROMPT-STORY")

	var summary string

	if len(allSessions) == 0 {
		summary = fmt.Sprintf("Prompt-Story: none [%s]", version)
		// Clean up any stale pending file
		os.Remove(pendingFile)
	} else {
		// Create PII scrubber (disabled via GIT_PROMPT_STORY_NO_SCRUB=1)
		var piiScrubber scrubber.Scrubber
		if os.Getenv("GIT_PROMPT_STORY_NO_SCRUB") != "1" {
			piiScrubber, err = scrubber.NewDefault()
			if err != nil {
				return fmt.Errorf("failed to create scrubber: %w", err)
			}
		}

		// Store transcripts as blobs (with optional PII scrubbing)
		blobs, err := note.StoreTranscriptsMulti(allSessions, piiScrubber)
		if err != nil {
			return fmt.Errorf("failed to store transcripts: %w", err)
		}

		// Update transcript tree ref
		if err := note.UpdateTranscriptTreeMulti(blobs); err != nil {
			return fmt.Errorf("failed to update transcript tree: %w", err)
		}

		// Create PromptStoryNote
		psNote := note.NewPromptStoryNoteMulti(allSessions, isAmend)
		noteJSON, err := psNote.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to serialize note: %w", err)
		}

		// Store note as blob
		noteSHA, err := git.HashObject(noteJSON)
		if err != nil {
			return fmt.Errorf("failed to store note blob: %w", err)
		}

		// Write pending note SHA
		if err := os.WriteFile(pendingFile, []byte(noteSHA), 0644); err != nil {
			return fmt.Errorf("failed to write pending file: %w", err)
		}

		// Count user actions using parsers
		promptCount := countUserActions(allSessions, startWork, endWork)

		summary = psNote.GenerateSummary(promptCount, version)
	}

	debugLog.log("Final summary: %s", summary)
	debugLog.log("=== prepare-commit-msg finished ===\n")

	// Append summary to commit message
	return appendToCommitMessage(msgFile, summary)
}

// appendToCommitMessage appends the summary line to the commit message file
// If a Prompt-Story marker already exists (e.g., during amend), it replaces it
func appendToCommitMessage(msgFile, summary string) error {
	content, err := os.ReadFile(msgFile)
	if err != nil {
		return err
	}

	newContent := string(content)

	// Remove existing Prompt-Story marker if present (for amend case)
	lines := strings.Split(newContent, "\n")
	var filtered []string
	for _, line := range lines {
		if !strings.HasPrefix(line, "Prompt-Story:") {
			filtered = append(filtered, line)
		}
	}
	newContent = strings.Join(filtered, "\n")

	// Trim trailing newlines and add consistent formatting
	newContent = strings.TrimRight(newContent, "\n")
	if len(newContent) > 0 {
		newContent += "\n"
	}
	newContent += "\n" + summary + "\n"

	return os.WriteFile(msgFile, []byte(newContent), 0644)
}

// debugLogger writes debug info to a file
type debugLogger struct {
	path string
}

func newDebugLogger(path string) *debugLogger {
	return &debugLogger{path: path}
}

func (d *debugLogger) log(format string, args ...interface{}) {
	f, err := os.OpenFile(d.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return // Silently fail - debug logging shouldn't break commits
	}
	defer f.Close()
	fmt.Fprintf(f, format+"\n", args...)
}

// countUserActions counts user actions across all sessions using the parser registry
func countUserActions(sessions []provider.RawSession, startWork, endWork time.Time) int {
	count := 0
	for _, sess := range sessions {
		p := provider.Get(sess.Tool)
		if p == nil {
			continue
		}

		content, err := p.ReadTranscript(sess)
		if err != nil {
			continue
		}

		pr := parser.Get(sess.Tool)
		if pr == nil {
			continue
		}

		count += pr.CountUserActions(content, startWork, endWork)
	}
	return count
}
