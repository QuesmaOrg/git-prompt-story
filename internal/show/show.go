package show

import (
	"fmt"
	"strings"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/note"
	"github.com/QuesmaOrg/git-prompt-story/internal/session"
	"gopkg.in/yaml.v3"
)

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

	// Parse note YAML
	var psNote note.PromptStoryNote
	if err := yaml.Unmarshal([]byte(noteContent), &psNote); err != nil {
		return fmt.Errorf("failed to parse note: %w", err)
	}

	// Get end work time from commit timestamp
	endWork, err := git.GetCommitTimestamp(sha)
	if err != nil {
		endWork = time.Now() // fallback
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
	for _, sessPath := range psNote.Sessions {
		shown, err := showSession(sessPath, psNote.StartWork, endWork, full)
		if err != nil {
			fmt.Printf("Warning: could not load session %s: %v\n", sessPath, err)
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

func showSession(sessPath string, startWork, endWork time.Time, full bool) (bool, error) {
	// Fetch transcript content using session path (e.g. "claude-code/uuid.jsonl")
	content, err := git.GetBlobContent(note.TranscriptsRefPrefix, sessPath)
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
					entryType = "USER"
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

	// Print session header (extract tool/id from path)
	tool, sessionID := note.ParseSessionPath(sessPath)
	fmt.Printf("Session: %s/%s\n\n", tool, sessionID)

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
		// Summary mode: truncate long content
		summary := truncate(de.text, 100)
		charCount := len(de.text)
		if charCount > 100 {
			fmt.Printf("[%s] %s: %s (%d chars)\n", timeStr, de.entryType, summary, charCount)
		} else {
			fmt.Printf("[%s] %s: %s\n", timeStr, de.entryType, summary)
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
