package note

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
)

// MarkSessionRemoved marks a session as removed in a commit's note
// Returns true if the session was found and marked, false if not found
func MarkSessionRemoved(commitSHA, sessionID string) (bool, error) {
	// Get the existing note
	noteJSON, err := GetNoteWithFallback(commitSHA)
	if err != nil {
		return false, fmt.Errorf("getting note for %s: %w", commitSHA[:8], err)
	}

	// Parse it
	var note PromptStoryNote
	if err := json.Unmarshal([]byte(noteJSON), &note); err != nil {
		return false, fmt.Errorf("parsing note for %s: %w", commitSHA[:8], err)
	}

	// Find and mark the session
	found := false
	now := time.Now()
	for i := range note.Sessions {
		if note.Sessions[i].ID == sessionID {
			note.Sessions[i].Removed = true
			note.Sessions[i].RemovedAt = &now
			found = true
			break
		}
	}

	if !found {
		return false, nil
	}

	// Serialize and update
	newNoteJSON, err := json.MarshalIndent(note, "", "  ")
	if err != nil {
		return false, fmt.Errorf("serializing note: %w", err)
	}

	if err := git.AddNote(NotesRef, string(newNoteJSON), commitSHA); err != nil {
		return false, fmt.Errorf("updating note for %s: %w", commitSHA[:8], err)
	}

	return true, nil
}

// RemoveSessionFromTranscripts removes a session's transcript from the tree
func RemoveSessionFromTranscripts(sessionID, tool string) error {
	// Get existing tree
	existingTreeSHA, err := git.GetRef(TranscriptsRef)
	if err != nil || existingTreeSHA == "" {
		// No transcript tree exists, nothing to remove
		return nil
	}

	// Read root tree
	rootEntries, err := git.ReadTree(existingTreeSHA)
	if err != nil {
		return fmt.Errorf("reading transcript tree: %w", err)
	}

	// Find the tool subtree (e.g., claude-code)
	var toolTreeSHA string
	for _, entry := range rootEntries {
		if entry.Name == tool && entry.Type == "tree" {
			toolTreeSHA = entry.SHA
			break
		}
	}

	if toolTreeSHA == "" {
		// Tool subtree doesn't exist
		return nil
	}

	// Read tool subtree entries
	toolEntries, err := git.ReadTree(toolTreeSHA)
	if err != nil {
		return fmt.Errorf("reading %s tree: %w", tool, err)
	}

	// Filter out the session
	targetFile := sessionID + ".jsonl"
	var newEntries []git.TreeEntry
	found := false
	for _, entry := range toolEntries {
		if entry.Name == targetFile {
			found = true
			continue // Skip this entry
		}
		newEntries = append(newEntries, entry)
	}

	if !found {
		// Session wasn't in the tree
		return nil
	}

	// Create new tool subtree
	var newToolTreeSHA string
	if len(newEntries) > 0 {
		newToolTreeSHA, err = git.CreateTree(newEntries)
		if err != nil {
			return fmt.Errorf("creating new %s tree: %w", tool, err)
		}
	}

	// Rebuild root tree
	var newRootEntries []git.TreeEntry
	for _, entry := range rootEntries {
		if entry.Name == tool {
			if newToolTreeSHA != "" {
				newRootEntries = append(newRootEntries, git.TreeEntry{
					Mode: entry.Mode,
					Type: entry.Type,
					SHA:  newToolTreeSHA,
					Name: entry.Name,
				})
			}
			// If newToolTreeSHA is empty, we omit this entry (empty subtree)
		} else {
			newRootEntries = append(newRootEntries, entry)
		}
	}

	if len(newRootEntries) == 0 {
		// All transcripts removed - we could delete the ref, but for now leave an empty tree
		emptyTreeSHA, err := git.CreateTree(nil)
		if err != nil {
			return fmt.Errorf("creating empty tree: %w", err)
		}
		return git.UpdateRef(TranscriptsRef, emptyTreeSHA)
	}

	newRootTreeSHA, err := git.CreateTree(newRootEntries)
	if err != nil {
		return fmt.Errorf("creating new root tree: %w", err)
	}

	return git.UpdateRef(TranscriptsRef, newRootTreeSHA)
}

// SessionCommitInfo holds info about a session and which commits reference it
type SessionCommitInfo struct {
	SessionID string
	Tool      string
	Created   time.Time
	Modified  time.Time
	Removed   bool
	Commits   []string // commit SHAs that reference this session
}

// FindSessionsInCommits finds all sessions referenced by the given commits
// Returns a map of session ID -> SessionCommitInfo
func FindSessionsInCommits(commits []string) (map[string]*SessionCommitInfo, error) {
	sessions := make(map[string]*SessionCommitInfo)

	for _, sha := range commits {
		noteJSON, err := GetNoteWithFallback(sha)
		if err != nil {
			continue // No note for this commit
		}

		var note PromptStoryNote
		if err := json.Unmarshal([]byte(noteJSON), &note); err != nil {
			continue // Invalid note
		}

		for _, s := range note.Sessions {
			if existing, ok := sessions[s.ID]; ok {
				// Add this commit to existing session
				existing.Commits = append(existing.Commits, sha)
			} else {
				// New session
				sessions[s.ID] = &SessionCommitInfo{
					SessionID: s.ID,
					Tool:      s.Tool,
					Created:   s.Created,
					Modified:  s.Modified,
					Removed:   s.Removed,
					Commits:   []string{sha},
				}
			}
		}
	}

	return sessions, nil
}

// FindAllCommitsWithSession finds all commits that reference a specific session
func FindAllCommitsWithSession(sessionID string) ([]string, error) {
	// List all commits with notes
	allCommits, err := git.ListNotedCommits(NotesRef)
	if err != nil {
		return nil, err
	}

	// Also check legacy ref
	legacyCommits, _ := git.ListNotedCommits(LegacyNotesRef)
	allCommits = append(allCommits, legacyCommits...)

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, c := range allCommits {
		if !seen[c] {
			seen[c] = true
			unique = append(unique, c)
		}
	}

	// Find which ones reference this session
	var result []string
	for _, sha := range unique {
		noteJSON, err := GetNoteWithFallback(sha)
		if err != nil {
			continue
		}

		var note PromptStoryNote
		if err := json.Unmarshal([]byte(noteJSON), &note); err != nil {
			continue
		}

		for _, s := range note.Sessions {
			if s.ID == sessionID {
				result = append(result, sha)
				break
			}
		}
	}

	return result, nil
}
