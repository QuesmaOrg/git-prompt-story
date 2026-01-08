package note

import (
	"fmt"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/scrubber"
	"github.com/QuesmaOrg/git-prompt-story/internal/session"
)

// TranscriptBlob represents a stored transcript blob
type TranscriptBlob struct {
	PromptTool string // e.g., "claude-code"
	ID         string // Session ID
	SHA        string // Git blob SHA
}

// StoreTranscripts stores session transcripts in the transcript tree.
// Accepts sessions implementing the session.Session interface.
// If scrub is not nil, PII is scrubbed from content before storing.
// Returns slice of TranscriptBlob with prompt tool, session ID, and blob SHA.
func StoreTranscripts(sessions []session.Session, scrub scrubber.Scrubber) ([]TranscriptBlob, error) {
	var blobs []TranscriptBlob

	for _, s := range sessions {
		content, err := s.ReadContent()
		if err != nil {
			continue // Skip files we can't read
		}

		// Scrub PII before storing
		if scrub != nil {
			content, err = scrub.Scrub(content)
			if err != nil {
				return nil, fmt.Errorf("scrubbing session %s: %w", s.GetID(), err)
			}
		}

		sha, err := git.HashObject(content)
		if err != nil {
			return nil, err
		}
		blobs = append(blobs, TranscriptBlob{
			PromptTool: s.GetPromptTool(),
			ID:         s.GetID(),
			SHA:        sha,
		})
	}

	return blobs, nil
}

// UpdateTranscriptTree updates the transcript tree ref with transcripts.
// Groups blobs by prompt tool and stores in versioned paths (e.g., claude-code/v1/).
func UpdateTranscriptTree(blobs []TranscriptBlob) error {
	// Group blobs by prompt tool
	byTool := make(map[string][]TranscriptBlob)
	for _, b := range blobs {
		byTool[b.PromptTool] = append(byTool[b.PromptTool], b)
	}

	// Get existing tree to merge with
	existingTreeSHA, _ := git.GetRef(TranscriptsRef)
	existingToolTrees := make(map[string]string) // tool name -> tree SHA

	if existingTreeSHA != "" {
		rootEntries, err := git.ReadTree(existingTreeSHA)
		if err == nil {
			for _, entry := range rootEntries {
				if entry.Type == "tree" {
					existingToolTrees[entry.Name] = entry.SHA
				}
			}
		}
	}

	// Build new tool subtrees
	toolSubtrees := make(map[string]string) // tool name -> new tree SHA

	for tool, toolBlobs := range byTool {
		// Build blob entries for this tool
		var blobEntries []git.TreeEntry
		for _, b := range toolBlobs {
			blobEntries = append(blobEntries, git.TreeEntry{
				Mode: "100644",
				Type: "blob",
				SHA:  b.SHA,
				Name: b.ID + ".jsonl",
			})
		}

		// Merge with existing entries for this tool
		if existingSHA, ok := existingToolTrees[tool]; ok {
			existingEntries, err := git.ReadTree(existingSHA)
			if err == nil {
				existingIDs := make(map[string]bool)
				for _, e := range blobEntries {
					existingIDs[e.Name] = true
				}
				for _, e := range existingEntries {
					if !existingIDs[e.Name] {
						blobEntries = append(blobEntries, e)
					}
				}
			}
		}

		// Create tool subtree
		toolTreeSHA, err := git.CreateTree(blobEntries)
		if err != nil {
			return err
		}
		toolSubtrees[tool] = toolTreeSHA
	}

	// Preserve existing tool trees that weren't updated
	for tool, sha := range existingToolTrees {
		if _, updated := toolSubtrees[tool]; !updated {
			toolSubtrees[tool] = sha
		}
	}

	// Build root tree with all tool subtrees
	var rootEntries []git.TreeEntry
	for tool, sha := range toolSubtrees {
		rootEntries = append(rootEntries, git.TreeEntry{
			Mode: "040000",
			Type: "tree",
			SHA:  sha,
			Name: tool,
		})
	}

	rootTreeSHA, err := git.CreateTree(rootEntries)
	if err != nil {
		return err
	}

	// Update the ref
	return git.UpdateRef(TranscriptsRef, rootTreeSHA)
}
