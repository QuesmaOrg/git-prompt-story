package hooks

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/note"
)

// PostRewrite handles the post-rewrite hook.
// It's called after rebase/amend with mappings of old-sha -> new-sha.
// For squashed commits (multiple old -> same new), it merges the notes.
func PostRewrite(rewriteType string, mappings io.Reader) error {
	// Parse mappings from stdin
	// Format: "old-sha new-sha" per line
	oldToNew := make(map[string]string)
	newToOlds := make(map[string][]string)

	scanner := bufio.NewScanner(mappings)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		oldSHA := parts[0]
		newSHA := parts[1]

		oldToNew[oldSHA] = newSHA
		newToOlds[newSHA] = append(newToOlds[newSHA], oldSHA)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading mappings: %w", err)
	}

	// Process each new commit
	for newSHA, oldSHAs := range newToOlds {
		if err := processRewrittenCommit(newSHA, oldSHAs); err != nil {
			// Log but don't fail - notes are optional
			fmt.Printf("Warning: could not transfer notes for %s: %v\n", newSHA[:7], err)
		}
	}

	return nil
}

// processRewrittenCommit transfers/merges notes from old commits to new commit
func processRewrittenCommit(newSHA string, oldSHAs []string) error {
	// Collect notes from all old commits
	var notes []*note.PromptStoryNote

	for _, oldSHA := range oldSHAs {
		noteData, err := note.GetNote(oldSHA)
		if err != nil {
			// No note on this commit, skip
			continue
		}

		parsed, err := note.ParseNote([]byte(noteData))
		if err != nil {
			// Invalid note format, skip
			continue
		}

		notes = append(notes, parsed)
	}

	if len(notes) == 0 {
		// No notes to transfer
		return nil
	}

	// Merge notes if multiple
	merged := note.MergeNotes(notes)
	if merged == nil {
		return nil
	}

	// Serialize and add to new commit
	jsonData, err := merged.ToJSON()
	if err != nil {
		return fmt.Errorf("serializing merged note: %w", err)
	}

	// Add note to new commit
	if err := git.AddNote(note.NotesRef, string(jsonData), newSHA); err != nil {
		return fmt.Errorf("adding note to %s: %w", newSHA, err)
	}

	fmt.Printf("Transferred prompt-story note to %s (%d sessions)\n", newSHA[:7], len(merged.Sessions))
	return nil
}
