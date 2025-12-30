package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/note"
	"github.com/QuesmaOrg/git-prompt-story/internal/session"
)

// PrepareCommitMsg implements the prepare-commit-msg hook logic
func PrepareCommitMsg(msgFile, source, sha string) error {
	// Get repo root
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Read current commit message to detect if this is an amend
	msgContent, err := os.ReadFile(msgFile)
	if err != nil {
		return fmt.Errorf("failed to read commit message: %w", err)
	}

	// Detect amend: if message already has our marker, or source is "commit" with SHA
	// Note: git passes source="message" when using -m flag, even with --amend
	// So we also check for existing marker as a reliable amend indicator
	isAmend := (source == "commit" && sha != "") ||
		strings.Contains(string(msgContent), "Prompt-Story:")

	// Find Claude Code sessions for this repo
	sessions, err := session.FindSessions(repoRoot)
	if err != nil {
		// Don't fail the commit, just log
		fmt.Fprintf(os.Stderr, "git-prompt-story: warning: %v\n", err)
		sessions = nil
	}

	// Filter sessions to only those overlapping with the work period
	if len(sessions) > 0 {
		startWork, _ := git.CalculateWorkStartTime(isAmend)
		endWork := git.GetCommitTime()
		sessions = session.FilterSessionsByTime(sessions, startWork, endWork)
	}

	// Get git directory for pending file
	gitDir, err := git.GetGitDir()
	if err != nil {
		return err
	}
	pendingFile := filepath.Join(gitDir, "PENDING-PROMPT-STORY")

	var summary string

	if len(sessions) == 0 {
		summary = "Prompt-Story: none"
		// Clean up any stale pending file
		os.Remove(pendingFile)
	} else {
		// Store transcripts as blobs
		blobs, err := note.StoreTranscripts(sessions)
		if err != nil {
			return fmt.Errorf("failed to store transcripts: %w", err)
		}

		// Update transcript tree ref
		if err := note.UpdateTranscriptTree(blobs); err != nil {
			return fmt.Errorf("failed to update transcript tree: %w", err)
		}

		// Create PromptStoryNote
		psNote := note.NewPromptStoryNote(sessions, isAmend)
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

		summary = psNote.GenerateSummary(noteSHA)
	}

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
