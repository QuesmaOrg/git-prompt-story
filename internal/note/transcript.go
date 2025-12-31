package note

import (
	"fmt"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/scrubber"
	"github.com/QuesmaOrg/git-prompt-story/internal/session"
)

const transcriptRef = "refs/notes/prompt-story-transcripts"

// StoreTranscripts stores session transcripts in the transcript tree
// If scrub is not nil, PII is scrubbed from content before storing
// Returns map of session ID -> blob SHA
func StoreTranscripts(sessions []session.ClaudeSession, scrub scrubber.Scrubber) (map[string]string, error) {
	blobs := make(map[string]string)

	for _, s := range sessions {
		content, err := session.ReadSessionContent(s.Path)
		if err != nil {
			continue // Skip files we can't read
		}

		// Scrub PII before storing
		if scrub != nil {
			content, err = scrub.Scrub(content)
			if err != nil {
				return nil, fmt.Errorf("scrubbing session %s: %w", s.ID, err)
			}
		}

		sha, err := git.HashObject(content)
		if err != nil {
			return nil, err
		}
		blobs[s.ID] = sha
	}

	return blobs, nil
}

// UpdateTranscriptTree updates the transcript tree ref with transcripts
func UpdateTranscriptTree(blobs map[string]string) error {
	// Build tree entries for claude-code/
	var claudeEntries []git.TreeEntry
	for id, sha := range blobs {
		claudeEntries = append(claudeEntries, git.TreeEntry{
			Mode: "100644",
			Type: "blob",
			SHA:  sha,
			Name: id + ".jsonl",
		})
	}

	// Check if we already have a transcript tree to merge with
	existingTreeSHA, _ := git.GetRef(transcriptRef)
	if existingTreeSHA != "" {
		// Read existing tree
		rootEntries, err := git.ReadTree(existingTreeSHA)
		if err == nil {
			// Find existing claude-code subtree
			for _, entry := range rootEntries {
				if entry.Name == "claude-code" && entry.Type == "tree" {
					// Read existing claude-code entries
					existingClaudeEntries, err := git.ReadTree(entry.SHA)
					if err == nil {
						// Merge: add existing entries that aren't being replaced
						existingIDs := make(map[string]bool)
						for _, e := range claudeEntries {
							existingIDs[e.Name] = true
						}
						for _, e := range existingClaudeEntries {
							if !existingIDs[e.Name] {
								claudeEntries = append(claudeEntries, e)
							}
						}
					}
					break
				}
			}
		}
	}

	// Create claude-code subtree
	claudeCodeTreeSHA, err := git.CreateTree(claudeEntries)
	if err != nil {
		return err
	}

	// Build root tree with claude-code subtree
	rootEntries := []git.TreeEntry{{
		Mode: "040000",
		Type: "tree",
		SHA:  claudeCodeTreeSHA,
		Name: "claude-code",
	}}
	rootTreeSHA, err := git.CreateTree(rootEntries)
	if err != nil {
		return err
	}

	// Update the ref
	return git.UpdateRef(transcriptRef, rootTreeSHA)
}
