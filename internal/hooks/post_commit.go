package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/note"
)

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

	// Attach note to HEAD by reusing the existing blob SHA
	// This ensures the note hash matches what's in the commit message trailer
	if err := git.AddNoteFromBlob(note.NotesRef, noteSHA, headSHA); err != nil {
		return fmt.Errorf("failed to attach note: %w", err)
	}

	// Cleanup
	os.Remove(pendingFile)

	return nil
}
