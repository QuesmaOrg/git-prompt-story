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

// ShowPrompts displays prompts for a given commit or range
func ShowPrompts(commitRef string, full bool) error {
	// Determine the type of reference and get commit list
	commits, err := resolveCommitSpec(commitRef)
	if err != nil {
		return err
	}

	// Show prompts for each commit
	for i, sha := range commits {
		if i > 0 {
			fmt.Println("---")
			fmt.Println()
		}
		if err := showCommitPrompts(sha, full); err != nil {
			return err
		}
	}
	return nil
}

// resolveCommitSpec resolves a commit specification to a list of commit SHAs
// Supports: single ref, ranges (A..B)
func resolveCommitSpec(spec string) ([]string, error) {
	// Check for range (contains ..)
	if strings.Contains(spec, "..") {
		commits, err := git.RevList(spec)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve range %s: %w", spec, err)
		}
		if len(commits) == 0 {
			return nil, fmt.Errorf("no commits in range %s", spec)
		}
		return commits, nil
	}

	// Single commit reference
	sha, err := git.ResolveCommit(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve commit %s: %w", spec, err)
	}
	return []string{sha}, nil
}

// showCommitPrompts displays prompts for a single commit
func showCommitPrompts(sha string, full bool) error {

	// Get note attached to commit
	noteContent, err := note.GetNoteWithFallback(sha)
	if err != nil {
		return fmt.Errorf("no prompt-story note found for commit %s", sha[:7])
	}

	// Parse note JSON
	var psNote note.PromptStoryNote
	if err := json.Unmarshal([]byte(noteContent), &psNote); err != nil {
		return fmt.Errorf("failed to parse note: %w", err)
	}

	// Get commit timestamp from git (end of work period)
	endWork, err := git.GetPreviousCommitTimestamp(sha)
	if err != nil || endWork.IsZero() {
		// Fallback to latest session modified time if commit time unavailable
		for _, sess := range psNote.Sessions {
			if sess.Modified.After(endWork) {
				endWork = sess.Modified
			}
		}
	}

	// Print header
	fmt.Printf("Commit: %s\n", sha[:7])
	fmt.Printf("Work period: %s - %s\n\n",
		psNote.StartWork.Local().Format("2006-01-02 15:04"),
		endWork.Local().Format("2006-01-02 15:04"))

	if len(psNote.Sessions) == 0 {
		fmt.Println("No sessions recorded")
		return nil
	}

	// Process each session, filtering out empty ones
	shownSessions := 0
	for _, sess := range psNote.Sessions {
		shown, err := showSession(sess, psNote.StartWork, endWork, full)
		if err != nil {
			fmt.Printf("Warning: could not load session %s: %v\n", sess.ID, err)
			continue
		}
		if shown {
			shownSessions++
		}
	}

	if shownSessions == 0 {
		fmt.Println("No messages to display")
	}

	return nil
}

// displayEntry holds a parsed entry ready for display
type displayEntry struct {
	ts       time.Time
	entryType string
	text     string
}

func showSession(sess note.SessionEntry, startWork, endWork time.Time, full bool) (bool, error) {
	// Extract relative path from full ref path
	relPath := strings.TrimPrefix(sess.Path, note.TranscriptsRef+"/")

	// Fetch transcript content
	content, err := git.GetBlobContent(note.TranscriptsRef, relPath)
	if err != nil {
		return false, fmt.Errorf("failed to fetch transcript: %w", err)
	}

	// Parse messages
	entries, err := session.ParseMessages(content)
	if err != nil {
		return false, fmt.Errorf("failed to parse messages: %w", err)
	}

	// Collect displayable entries within the work period
	var displayEntries []displayEntry
	for _, entry := range entries {
		// Get timestamp
		ts := entry.Timestamp
		if ts.IsZero() && entry.Snapshot != nil {
			ts = entry.Snapshot.Timestamp
		}

		// Skip entries outside work period or without timestamp
		if ts.IsZero() {
			continue
		}
		if ts.Before(startWork) || ts.After(endWork) {
			continue
		}

		// Determine entry type and text to display
		var entryType, text string

		switch entry.Type {
		case "user":
			if entry.Message != nil {
				text = entry.Message.GetTextContent()
				if text != "" {
					entryType = "PROMPT"
				}
			}
		case "tool_reject":
			entryType = "TOOL_REJECT"
			if entry.Message != nil {
				text = entry.Message.GetTextContent()
			}
			if text == "" {
				text = "User rejected tool call"
			}
		}

		// Check for user commands (messages starting with <command-name>)
		if entry.Type == "user" && entry.Message != nil {
			msgText := entry.Message.GetTextContent()
			if strings.HasPrefix(msgText, "<command-name>") {
				// Extract command name
				start := strings.Index(msgText, "<command-name>") + len("<command-name>")
				end := strings.Index(msgText, "</command-name>")
				if end > start {
					cmdName := msgText[start:end]
					// Remove leading slash if present (command names may include it)
					cmdName = strings.TrimPrefix(cmdName, "/")
					entryType = "COMMAND"
					text = "/" + cmdName
				}
			}
			// Skip local command output entries
			if strings.HasPrefix(msgText, "<local-command-stdout>") {
				entryType = ""
				text = ""
			}
		}

		if entryType != "" {
			displayEntries = append(displayEntries, displayEntry{
				ts:       ts,
				entryType: entryType,
				text:     text,
			})
		}
	}

	// Skip session if no entries to display
	if len(displayEntries) == 0 {
		return false, nil
	}

	// Print session header
	fmt.Printf("Session: %s/%s\n", sess.Tool, sess.ID)
	fmt.Printf("Duration: %s - %s\n\n",
		sess.Created.Local().Format("2006-01-02 15:04"),
		sess.Modified.Local().Format("2006-01-02 15:04"))

	// Display entries
	for _, de := range displayEntries {
		displayMessage(de, full)
	}

	fmt.Println()
	return true, nil
}

func displayMessage(de displayEntry, full bool) {
	timeStr := de.ts.Local().Format("15:04")

	if full {
		// Full mode: show complete content
		fmt.Printf("[%s] %s:\n%s\n\n", timeStr, de.entryType, de.text)
	} else {
		// Summary mode: show up to 3 lines of 80 chars each
		lines, truncated := wrapAndTruncate(de.text, 80, 3)
		charCount := len(de.text)
		fmt.Printf("[%s] %s:\n", timeStr, de.entryType)
		for _, line := range lines {
			fmt.Printf("  %s\n", line)
		}
		if truncated {
			fmt.Printf("  (%d chars total)\n", charCount)
		}
	}
}

// wrapAndTruncate wraps text to maxWidth chars per line and limits to maxLines.
// Returns the lines and whether truncation occurred.
func wrapAndTruncate(s string, maxWidth, maxLines int) ([]string, bool) {
	// Normalize whitespace but preserve logical line breaks
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	var result []string
	truncated := false

	// Split by newlines first
	paragraphs := strings.Split(s, "\n")

	for _, para := range paragraphs {
		if len(result) >= maxLines {
			truncated = true
			break
		}

		// Normalize whitespace within paragraph
		para = strings.Join(strings.Fields(para), " ")
		if para == "" {
			continue
		}

		// Wrap the paragraph
		for len(para) > 0 && len(result) < maxLines {
			if len(para) <= maxWidth {
				result = append(result, para)
				break
			}
			// Find break point (prefer space)
			breakAt := maxWidth
			for i := maxWidth; i > 0; i-- {
				if para[i] == ' ' {
					breakAt = i
					break
				}
			}
			result = append(result, para[:breakAt])
			para = strings.TrimLeft(para[breakAt:], " ")
		}
		if len(para) > 0 && len(result) >= maxLines {
			truncated = true
		}
	}

	// Mark last line as truncated if needed
	if truncated && len(result) > 0 {
		lastLine := result[len(result)-1]
		if len(lastLine) > maxWidth-3 {
			result[len(result)-1] = lastLine[:maxWidth-3] + "..."
		} else {
			result[len(result)-1] = lastLine + "..."
		}
	}

	return result, truncated
}
