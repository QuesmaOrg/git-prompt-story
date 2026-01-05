package note

import (
	"github.com/QuesmaOrg/git-prompt-story/internal/git"
)

// Note refs for prompt-story data
const (
	// NotesRef is the primary ref for commit metadata notes
	NotesRef = "refs/notes/prompt-story"

	// TranscriptsRef is the ref for transcript tree storage
	TranscriptsRef = "refs/notes/prompt-story-transcripts"
)

// GetNote retrieves a prompt-story note for the given commit SHA
func GetNote(sha string) (string, error) {
	return git.GetNote(NotesRef, sha)
}

