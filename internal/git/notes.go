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
