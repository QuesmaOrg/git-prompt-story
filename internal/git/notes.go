package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// AddNote adds a note to an object using a specific notes ref
func AddNote(ref, message, object string) error {
	cmd := exec.Command("git", "notes", "--ref="+ref, "add", "-f", "-m", message, object)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git notes add: %w", err)
	}
	return nil
}

// AddNoteFromBlob adds a note to an object by reusing an existing blob
func AddNoteFromBlob(ref, blobSHA, object string) error {
	cmd := exec.Command("git", "notes", "--ref="+ref, "add", "-f", "-C", blobSHA, object)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git notes add -C: %w", err)
	}
	return nil
}

// GetNote retrieves a note for an object
func GetNote(ref, object string) (string, error) {
	cmd := exec.Command("git", "notes", "--ref="+ref, "show", object)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ListNotedCommits returns all commit SHAs that have notes in the given ref
func ListNotedCommits(ref string) ([]string, error) {
	cmd := exec.Command("git", "notes", "--ref="+ref, "list")
	out, err := cmd.Output()
	if err != nil {
		// No notes exist
		return nil, nil
	}

	var commits []string
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Format: note-sha SP commit-sha
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			commits = append(commits, parts[1])
		}
	}
	return commits, nil
}
