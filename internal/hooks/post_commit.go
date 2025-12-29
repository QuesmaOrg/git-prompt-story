package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
)

const notesRef = "refs/notes/commits"

// PostCommit implements the post-commit hook logic
func PostCommit() error {
	// Get git directory
	gitDir, err := git.GetGitDir()
	if err != nil {
		return err
	}

	pendingFile := filepath.Join(gitDir, "PENDING-PROMPT-STORY")

	// Read pending note SHA
	content, err := os.ReadFile(pendingFile)
	if os.IsNotExist(err) {
		// No pending note, nothing to do
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read pending file: %w", err)
	}

	noteSHA := strings.TrimSpace(string(content))
	if noteSHA == "" {
		os.Remove(pendingFile)
		return nil
	}

	// Get HEAD commit SHA
	headSHA, err := git.GetHead()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Read the note blob content
	noteContent, err := readBlobContent(noteSHA)
	if err != nil {
		return fmt.Errorf("failed to read note blob: %w", err)
	}

	// Attach note to HEAD
	if err := git.AddNote(notesRef, noteContent, headSHA); err != nil {
		return fmt.Errorf("failed to attach note: %w", err)
	}

	// Cleanup
	os.Remove(pendingFile)

	return nil
}

// readBlobContent reads the content of a blob object
func readBlobContent(sha string) (string, error) {
	cmd := exec.Command("git", "cat-file", "-p", sha)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
