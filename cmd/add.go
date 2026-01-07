package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/cloud"
	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/note"
	"github.com/QuesmaOrg/git-prompt-story/internal/scrubber"
	"github.com/spf13/cobra"
)

var (
	addSourceFlag    string
	addSessionIDFlag string
	addAutoFlag      bool
	addNoScrubFlag   bool
)

var addCmd = &cobra.Command{
	Use:   "add [commit]",
	Short: "Add a session to a commit",
	Long: `Add an LLM session from an external source to a commit.

Sources:
  cloud   - Claude Code Cloud session (requires ANTHROPIC_API_KEY)
  local   - Local Claude Code session (coming soon)

Examples:
  # Add a cloud session to HEAD
  git-prompt-story add --source=cloud --session-id=session_01XXX

  # Auto-detect cloud session from branch name
  git-prompt-story add --source=cloud --auto

  # Add to specific commit
  git-prompt-story add abc123 --source=cloud --session-id=session_01XXX`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		commit := "HEAD"
		if len(args) > 0 {
			commit = args[0]
		}

		switch addSourceFlag {
		case "cloud":
			if addSessionIDFlag == "" && !addAutoFlag {
				fmt.Fprintln(os.Stderr, "error: must specify --session-id or --auto for cloud source")
				os.Exit(1)
			}
			if err := addCloudSession(commit, addSessionIDFlag, addAutoFlag, addNoScrubFlag); err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
				os.Exit(1)
			}
		case "local":
			fmt.Fprintln(os.Stderr, "error: local source not yet implemented, use 'repair' command instead")
			os.Exit(1)
		default:
			fmt.Fprintf(os.Stderr, "error: unknown source: %s (valid: cloud, local)\n", addSourceFlag)
			os.Exit(1)
		}
	},
}

func init() {
	addCmd.Flags().StringVar(&addSourceFlag, "source", "", "Session source: cloud, local (required)")
	addCmd.Flags().StringVar(&addSessionIDFlag, "session-id", "", "Session ID to add")
	addCmd.Flags().BoolVar(&addAutoFlag, "auto", false, "Auto-detect session from branch name")
	addCmd.Flags().BoolVar(&addNoScrubFlag, "no-scrub", false, "Disable PII scrubbing")
	addCmd.MarkFlagRequired("source")
	rootCmd.AddCommand(addCmd)
}

func addCloudSession(commitRef, sessionID string, autoDetect, noScrub bool) error {
	// Resolve commit
	sha, err := git.ResolveCommit(commitRef)
	if err != nil {
		return fmt.Errorf("invalid commit reference: %w", err)
	}

	// Create cloud client
	client, err := cloud.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize cloud client: %w", err)
	}

	// Get session (either by ID or auto-detect)
	var sess *cloud.Session
	if autoDetect {
		// Get current branch name
		branchName, err := git.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
		fmt.Printf("Looking for cloud session matching branch: %s\n", branchName)

		sess, err = client.FindSessionByBranch(branchName)
		if err != nil {
			return err
		}
		fmt.Printf("Found session: %s (%s)\n", sess.Title, sess.ID)
	} else {
		sess, err = client.GetSession(sessionID)
		if err != nil {
			return fmt.Errorf("failed to get session: %w", err)
		}
	}

	// Fetch all events
	fmt.Printf("Fetching events from session...\n")
	events, err := client.GetAllSessionEvents(sess.ID)
	if err != nil {
		return fmt.Errorf("failed to get session events: %w", err)
	}

	// Count user/assistant messages
	userCount, assistantCount := 0, 0
	for _, e := range events {
		if e.Type == "user" {
			userCount++
		} else if e.Type == "assistant" {
			assistantCount++
		}
	}
	fmt.Printf("Found %d user messages, %d assistant responses\n", userCount, assistantCount)

	// Convert events to JSONL
	jsonl, err := cloud.EventsToJSONL(events, sess)
	if err != nil {
		return fmt.Errorf("failed to convert events: %w", err)
	}

	// Scrub PII from transcript (unless --no-scrub)
	if !noScrub {
		piiScrubber, err := scrubber.NewDefault()
		if err != nil {
			return fmt.Errorf("failed to create scrubber: %w", err)
		}
		jsonl, err = piiScrubber.Scrub(jsonl)
		if err != nil {
			return fmt.Errorf("failed to scrub PII: %w", err)
		}
	}

	// Store transcript as blob
	blobSHA, err := git.HashObject(jsonl)
	if err != nil {
		return fmt.Errorf("failed to store transcript: %w", err)
	}

	// Update transcript tree
	if err := updateCloudTranscriptTreeForAdd(sess.ID, blobSHA); err != nil {
		return fmt.Errorf("failed to update transcript tree: %w", err)
	}

	// Create PromptStoryNote
	psNote := &note.PromptStoryNote{
		Version:   1,
		StartWork: sess.CreatedAt,
		Sessions: []note.SessionEntry{{
			Tool:     "claude-cloud",
			ID:       sess.ID,
			Path:     note.GetTranscriptPath("claude-cloud", sess.ID),
			Created:  sess.CreatedAt,
			Modified: sess.UpdatedAt,
		}},
	}
	noteJSON, err := json.MarshalIndent(psNote, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize note: %w", err)
	}

	// Attach note to commit
	if err := git.AddNote(note.NotesRef, string(noteJSON), sha); err != nil {
		return fmt.Errorf("failed to attach note: %w", err)
	}

	fmt.Printf("Successfully added cloud session %s to commit %s\n", sess.ID, sha[:7])
	return nil
}

// updateCloudTranscriptTreeForAdd adds a cloud session transcript to the tree
func updateCloudTranscriptTreeForAdd(sessionID, blobSHA string) error {
	// Build entry for this session
	newEntry := git.TreeEntry{
		Mode: "100644",
		Type: "blob",
		SHA:  blobSHA,
		Name: sessionID + ".jsonl",
	}

	// Get existing cloud entries
	var cloudEntries []git.TreeEntry
	existingTreeSHA, _ := git.GetRef(note.TranscriptsRef)
	if existingTreeSHA != "" {
		rootEntries, err := git.ReadTree(existingTreeSHA)
		if err == nil {
			for _, entry := range rootEntries {
				if entry.Name == "claude-cloud" && entry.Type == "tree" {
					existingCloudEntries, err := git.ReadTree(entry.SHA)
					if err == nil {
						// Keep existing entries that aren't being replaced
						for _, e := range existingCloudEntries {
							if e.Name != newEntry.Name {
								cloudEntries = append(cloudEntries, e)
							}
						}
					}
					break
				}
			}
		}
	}
	cloudEntries = append(cloudEntries, newEntry)

	// Create claude-cloud subtree
	cloudTreeSHA, err := git.CreateTree(cloudEntries)
	if err != nil {
		return err
	}

	// Build root tree - need to preserve other subtrees (like claude-code)
	var rootEntries []git.TreeEntry
	if existingTreeSHA != "" {
		existingRootEntries, _ := git.ReadTree(existingTreeSHA)
		for _, entry := range existingRootEntries {
			if entry.Name != "claude-cloud" {
				rootEntries = append(rootEntries, entry)
			}
		}
	}
	rootEntries = append(rootEntries, git.TreeEntry{
		Mode: "040000",
		Type: "tree",
		SHA:  cloudTreeSHA,
		Name: "claude-cloud",
	})

	rootTreeSHA, err := git.CreateTree(rootEntries)
	if err != nil {
		return err
	}

	return git.UpdateRef(note.TranscriptsRef, rootTreeSHA)
}
