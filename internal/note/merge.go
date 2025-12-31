package note

import (
	"encoding/json"
	"sort"
)

// MergeNotes combines multiple PromptStoryNotes into one.
// Used when commits are squashed to preserve all session references.
// - Sessions are combined and deduplicated by ID
// - StartWork is set to the earliest timestamp
// - Version is set to the latest version
func MergeNotes(notes []*PromptStoryNote) *PromptStoryNote {
	if len(notes) == 0 {
		return nil
	}

	if len(notes) == 1 {
		return notes[0]
	}

	merged := &PromptStoryNote{
		Version:   1,
		Sessions:  make([]SessionEntry, 0),
		StartWork: notes[0].StartWork,
	}

	// Track seen session IDs to deduplicate
	seenSessions := make(map[string]bool)

	for _, note := range notes {
		// Use the earliest StartWork
		if note.StartWork.Before(merged.StartWork) {
			merged.StartWork = note.StartWork
		}

		// Use the latest version
		if note.Version > merged.Version {
			merged.Version = note.Version
		}

		// Add sessions, deduplicating by ID
		for _, session := range note.Sessions {
			if !seenSessions[session.ID] {
				seenSessions[session.ID] = true
				merged.Sessions = append(merged.Sessions, session)
			}
		}
	}

	// Sort sessions by created time for consistent output
	sort.Slice(merged.Sessions, func(i, j int) bool {
		return merged.Sessions[i].Created.Before(merged.Sessions[j].Created)
	})

	return merged
}

// ParseNote parses a JSON note into a PromptStoryNote
func ParseNote(data []byte) (*PromptStoryNote, error) {
	var note PromptStoryNote
	if err := json.Unmarshal(data, &note); err != nil {
		return nil, err
	}
	return &note, nil
}
