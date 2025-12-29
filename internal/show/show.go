package show

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/note"
	"github.com/QuesmaOrg/git-prompt-story/internal/session"
)

const transcriptsRef = "refs/notes/prompt-story-transcripts"
const notesRef = "refs/notes/commits"

// ShowPrompts displays prompts for a given commit
func ShowPrompts(commitRef string, full bool) error {
	// Resolve commit to SHA
	sha, err := git.ResolveCommit(commitRef)
	if err != nil {
		return fmt.Errorf("failed to resolve commit %s: %w", commitRef, err)
	}

	// Get note attached to commit
	noteContent, err := git.GetNote(notesRef, sha)
	if err != nil {
		return fmt.Errorf("no prompt-story note found for commit %s", sha[:7])
	}

	// Parse note JSON
	var psNote note.PromptStoryNote
	if err := json.Unmarshal([]byte(noteContent), &psNote); err != nil {
		return fmt.Errorf("failed to parse note: %w", err)
	}

	// Print header
	fmt.Printf("Commit: %s\n", sha[:7])
	fmt.Printf("Work period: %s - %s\n\n",
		psNote.StartWork.Local().Format("2006-01-02 15:04"),
		psNote.EndWork.Local().Format("2006-01-02 15:04"))

	if len(psNote.Sessions) == 0 {
		fmt.Println("No sessions recorded")
		return nil
	}

	// Process each session
	for _, sess := range psNote.Sessions {
		if err := showSession(sess, psNote.StartWork, psNote.EndWork, full); err != nil {
			fmt.Printf("Warning: could not load session %s: %v\n", sess.ID, err)
			continue
		}
	}

	return nil
}

func showSession(sess note.SessionEntry, startWork, endWork time.Time, full bool) error {
	fmt.Printf("Session: %s/%s\n", sess.Tool, sess.ID)
	fmt.Printf("Duration: %s - %s\n\n",
		sess.Created.Local().Format("15:04"),
		sess.Modified.Local().Format("15:04"))

	// Extract relative path from full ref path
	// Path is like: refs/notes/prompt-story-transcripts/claude-code/session-id.jsonl
	relPath := strings.TrimPrefix(sess.Path, transcriptsRef+"/")

	// Fetch transcript content
	content, err := git.GetBlobContent(transcriptsRef, relPath)
	if err != nil {
		return fmt.Errorf("failed to fetch transcript: %w", err)
	}

	// Parse messages
	entries, err := session.ParseMessages(content)
	if err != nil {
		return fmt.Errorf("failed to parse messages: %w", err)
	}

	// Filter and display messages within the work period
	for _, entry := range entries {
		// Get timestamp
		ts := entry.Timestamp
		if ts.IsZero() && entry.Snapshot != nil {
			ts = entry.Snapshot.Timestamp
		}

		// Skip entries outside work period or without message
		if ts.IsZero() || entry.Message == nil {
			continue
		}

		// Filter by time range
		if ts.Before(startWork) || ts.After(endWork) {
			continue
		}

		// Only show user messages
		if entry.Type != "user" {
			continue
		}

		displayMessage(entry, ts, full)
	}

	fmt.Println()
	return nil
}

func displayMessage(entry session.MessageEntry, ts time.Time, full bool) {
	role := strings.ToUpper(entry.Type)
	timeStr := ts.Local().Format("15:04")

	// Extract text content from message
	text := entry.Message.GetTextContent()
	if text == "" {
		return
	}

	if full {
		// Full mode: show complete content
		fmt.Printf("[%s] %s:\n%s\n\n", timeStr, role, text)
	} else {
		// Summary mode: truncate long content
		summary := truncate(text, 100)
		charCount := len(text)
		if charCount > 100 {
			fmt.Printf("[%s] %s: %s (%d chars)\n", timeStr, role, summary, charCount)
		} else {
			fmt.Printf("[%s] %s: %s\n", timeStr, role, summary)
		}
	}
}

func truncate(s string, maxLen int) string {
	// Replace newlines with spaces for summary
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ") // Normalize whitespace

	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
