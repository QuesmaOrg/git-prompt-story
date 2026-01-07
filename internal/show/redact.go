package show

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/note"
)

const redactedPlaceholder = "<REDACTED BY USER>"

// RedactMessage redacts a specific message in a session transcript.
// It updates both the git ref and local file (if found).
func RedactMessage(tool, sessionID string, timestamp time.Time) error {
	sessionPath := fmt.Sprintf("%s/%s.jsonl", tool, sessionID)

	// Read current transcript from git
	content, err := git.GetBlobContent(note.TranscriptsRef, sessionPath)
	if err != nil {
		return fmt.Errorf("failed to read transcript: %w", err)
	}

	// Redact the message
	newContent, err := redactJSONLEntry(content, timestamp)
	if err != nil {
		return fmt.Errorf("failed to redact message: %w", err)
	}

	// Update git ref
	if err := updateTranscriptInGit(sessionPath, newContent); err != nil {
		return fmt.Errorf("failed to update git ref: %w", err)
	}

	// Update local file (best effort - don't fail if not found)
	if err := updateLocalSessionFile(sessionID, newContent); err != nil {
		// Log but don't fail - local file might not exist
		fmt.Fprintf(os.Stderr, "Warning: could not update local file: %v\n", err)
	}

	return nil
}

// DeleteSession clears all content from a session transcript.
// It updates both the git ref and empties the local file.
func DeleteSession(tool, sessionID string) error {
	sessionPath := fmt.Sprintf("%s/%s.jsonl", tool, sessionID)

	// Empty content for the session
	emptyContent := []byte{}

	// Update git ref with empty content
	if err := updateTranscriptInGit(sessionPath, emptyContent); err != nil {
		return fmt.Errorf("failed to update git ref: %w", err)
	}

	// Empty local file (best effort - keep file but clear content)
	if err := updateLocalSessionFile(sessionID, emptyContent); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not empty local file: %v\n", err)
	}

	return nil
}

// WasNotesPushed checks if the transcript notes ref was pushed to origin.
// Returns true if local and remote refs match (meaning changes need force push).
func WasNotesPushed() bool {
	local, err := git.GetRef(note.TranscriptsRef)
	if err != nil || local == "" {
		return false
	}

	remote, err := git.GetRemoteRef("origin", note.TranscriptsRef)
	if err != nil || remote == "" {
		return false
	}

	return local == remote
}

// redactJSONLEntry finds and redacts a message by timestamp in JSONL content.
// It replaces the message content with [REDACTED].
func redactJSONLEntry(content []byte, timestamp time.Time) ([]byte, error) {
	var result bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(content))
	found := false

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Parse the entry to check timestamp
		var entry map[string]interface{}
		if err := json.Unmarshal(line, &entry); err != nil {
			// Keep invalid lines as-is
			result.Write(line)
			result.WriteByte('\n')
			continue
		}

		// Check if this is the entry to redact
		if shouldRedact(entry, timestamp) {
			// Redact the message content
			redactEntry(entry)
			found = true

			// Re-serialize
			newLine, err := json.Marshal(entry)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal redacted entry: %w", err)
			}
			result.Write(newLine)
		} else {
			result.Write(line)
		}
		result.WriteByte('\n')
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if !found {
		return nil, fmt.Errorf("message not found at timestamp %v", timestamp)
	}

	return result.Bytes(), nil
}

// shouldRedact checks if an entry matches the timestamp to redact
func shouldRedact(entry map[string]interface{}, timestamp time.Time) bool {
	tsStr, ok := entry["timestamp"].(string)
	if !ok {
		return false
	}

	// Parse timestamp (ISO 8601 format)
	entryTime, err := time.Parse(time.RFC3339, tsStr)
	if err != nil {
		// Try alternative formats
		entryTime, err = time.Parse("2006-01-02T15:04:05Z", tsStr)
		if err != nil {
			return false
		}
	}

	// Compare timestamps (within 1 second tolerance for rounding)
	diff := entryTime.Sub(timestamp)
	if diff < 0 {
		diff = -diff
	}
	return diff < time.Second
}

// redactEntry replaces the message content with [REDACTED]
func redactEntry(entry map[string]interface{}) {
	if msg, ok := entry["message"].(map[string]interface{}); ok {
		// Replace content field
		if _, hasContent := msg["content"]; hasContent {
			msg["content"] = redactedPlaceholder
		}
	}

	// Also redact direct content field if present
	if _, hasContent := entry["content"]; hasContent {
		entry["content"] = redactedPlaceholder
	}
}

// updateTranscriptInGit updates a transcript blob in the git refs tree
func updateTranscriptInGit(sessionPath string, content []byte) error {
	// Create new blob
	blobSHA, err := git.HashObject(content)
	if err != nil {
		return fmt.Errorf("failed to create blob: %w", err)
	}

	// Parse path to get tool and filename
	parts := strings.SplitN(sessionPath, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid session path: %s", sessionPath)
	}
	tool := parts[0]
	filename := parts[1]

	// Get existing transcript tree
	existingTreeSHA, _ := git.GetRef(note.TranscriptsRef)
	if existingTreeSHA == "" {
		return fmt.Errorf("transcript tree not found")
	}

	// Read root tree
	rootEntries, err := git.ReadTree(existingTreeSHA)
	if err != nil {
		return fmt.Errorf("failed to read root tree: %w", err)
	}

	// Find tool subtree
	var toolTreeSHA string
	for _, entry := range rootEntries {
		if entry.Name == tool && entry.Type == "tree" {
			toolTreeSHA = entry.SHA
			break
		}
	}
	if toolTreeSHA == "" {
		return fmt.Errorf("tool tree not found: %s", tool)
	}

	// Read tool subtree entries
	toolEntries, err := git.ReadTree(toolTreeSHA)
	if err != nil {
		return fmt.Errorf("failed to read tool tree: %w", err)
	}

	// Update the entry with new blob SHA
	found := false
	for i, entry := range toolEntries {
		if entry.Name == filename {
			toolEntries[i].SHA = blobSHA
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("transcript not found: %s", filename)
	}

	// Create new tool subtree
	newToolTreeSHA, err := git.CreateTree(toolEntries)
	if err != nil {
		return fmt.Errorf("failed to create tool tree: %w", err)
	}

	// Update root tree with new tool subtree
	for i, entry := range rootEntries {
		if entry.Name == tool {
			rootEntries[i].SHA = newToolTreeSHA
			break
		}
	}

	// Create new root tree
	newRootTreeSHA, err := git.CreateTree(rootEntries)
	if err != nil {
		return fmt.Errorf("failed to create root tree: %w", err)
	}

	// Update ref
	return git.UpdateRef(note.TranscriptsRef, newRootTreeSHA)
}

// updateLocalSessionFile updates a local session file with new content
func updateLocalSessionFile(sessionID string, content []byte) error {
	path, err := findLocalSessionFile(sessionID)
	if err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("local session file not found")
	}

	return os.WriteFile(path, content, 0644)
}

// findLocalSessionFile searches for a session file in ~/.claude/projects/
func findLocalSessionFile(sessionID string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	projectsDir := filepath.Join(homeDir, ".claude", "projects")

	// List all project directories
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	filename := sessionID + ".jsonl"

	// Search in each project directory
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(projectsDir, entry.Name(), filename)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", nil
}
