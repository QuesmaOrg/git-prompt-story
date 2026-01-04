package note

import (
	"github.com/QuesmaOrg/git-prompt-story/internal/git"
)

// Note refs for prompt-story data
const (
	// NotesRef is the primary ref for commit metadata notes
	NotesRef = "refs/notes/prompt-story"

	// LegacyNotesRef is the old ref location for backward compatibility
	LegacyNotesRef = "refs/notes/commits"

	// TranscriptsRef is the ref for transcript tree storage
	TranscriptsRef = "refs/notes/prompt-story-transcripts"
)

// GetNoteWithFallback tries to get a note from the primary ref,
// falling back to the legacy ref for backward compatibility
func GetNoteWithFallback(sha string) (string, error) {
	// Try primary ref first
	note, err := git.GetNote(NotesRef, sha)
	if err == nil {
		return note, nil
	}

	// Fall back to legacy ref
	return git.GetNote(LegacyNotesRef, sha)
}

