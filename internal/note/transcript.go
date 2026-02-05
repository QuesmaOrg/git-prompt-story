package note

import (
	"fmt"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/provider"
	"github.com/QuesmaOrg/git-prompt-story/internal/scrubber"
)

// TranscriptBlob represents a stored transcript blob
type TranscriptBlob struct {
	Tool string // Tool name (e.g., "claude-code", "cursor")
	ID   string // Session/composer ID
	SHA  string // Git blob SHA
	Ext  string // File extension (e.g., ".jsonl", ".json")
}

// StoreTranscriptsMulti stores session transcripts from multiple providers
// If scrub is not nil, PII is scrubbed from content before storing
// Returns list of stored transcript blobs
func StoreTranscriptsMulti(sessions []provider.RawSession, scrub scrubber.Scrubber) ([]TranscriptBlob, error) {
	var blobs []TranscriptBlob

	for _, s := range sessions {
		p := provider.Get(s.Tool)
		if p == nil {
			continue // Skip unknown providers
		}

		content, err := p.ReadTranscript(s)
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

		blobs = append(blobs, TranscriptBlob{
			Tool: s.Tool,
			ID:   s.ID,
			SHA:  sha,
			Ext:  p.FileExtension(),
		})
	}

	return blobs, nil
}

// UpdateTranscriptTreeMulti updates the transcript tree ref with transcripts from multiple tools
func UpdateTranscriptTreeMulti(blobs []TranscriptBlob) error {
	if len(blobs) == 0 {
		return nil
	}

	// Group blobs by tool
	toolBlobs := make(map[string][]TranscriptBlob)
	for _, b := range blobs {
		toolBlobs[b.Tool] = append(toolBlobs[b.Tool], b)
	}

	// Read existing tree to merge with
	existingSubtrees := make(map[string]string) // tool -> tree SHA
	existingTreeSHA, _ := git.GetRef(TranscriptsRef)
	if existingTreeSHA != "" {
		rootEntries, err := git.ReadTree(existingTreeSHA)
		if err == nil {
			for _, entry := range rootEntries {
				if entry.Type == "tree" {
					existingSubtrees[entry.Name] = entry.SHA
				}
			}
		}
	}

	// Build subtrees for each tool
	var rootEntries []git.TreeEntry
	for tool, tblobs := range toolBlobs {
		// Build entries for this tool
		var entries []git.TreeEntry
		newIDs := make(map[string]bool)
		for _, b := range tblobs {
			entries = append(entries, git.TreeEntry{
				Mode: "100644",
				Type: "blob",
				SHA:  b.SHA,
				Name: b.ID + b.Ext,
			})
			newIDs[b.ID+b.Ext] = true
		}

		// Merge with existing entries for this tool
		if existingSHA, ok := existingSubtrees[tool]; ok {
			existingEntries, err := git.ReadTree(existingSHA)
			if err == nil {
				for _, e := range existingEntries {
					if !newIDs[e.Name] {
						entries = append(entries, e)
					}
				}
			}
			delete(existingSubtrees, tool) // Mark as handled
		}

		// Create subtree
		subtreeSHA, err := git.CreateTree(entries)
		if err != nil {
			return err
		}

		rootEntries = append(rootEntries, git.TreeEntry{
			Mode: "040000",
			Type: "tree",
			SHA:  subtreeSHA,
			Name: tool,
		})
	}

	// Include existing subtrees that weren't updated
	for tool, sha := range existingSubtrees {
		rootEntries = append(rootEntries, git.TreeEntry{
			Mode: "040000",
			Type: "tree",
			SHA:  sha,
			Name: tool,
		})
	}

	// Create root tree
	rootTreeSHA, err := git.CreateTree(rootEntries)
	if err != nil {
		return err
	}

	// Update the ref
	return git.UpdateRef(TranscriptsRef, rootTreeSHA)
}
