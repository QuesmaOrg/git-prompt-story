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

var sessionIDFlag string
var autoFlag bool
var noScrubFlag bool

var annotateCloudCmd = &cobra.Command{
	Use:   "annotate-cloud [commit]",
	Short: "Annotate a commit with a Claude Code Cloud session",
	Long: `Fetch a Claude Code Cloud session and attach it as a prompt-story note
to the specified commit.

Examples:
  # Annotate HEAD with a specific session
  git-prompt-story annotate-cloud HEAD --session-id=session_01XXX

  # Auto-detect session from current branch name
  git-prompt-story annotate-cloud HEAD --auto`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		commit := "HEAD"
		if len(args) > 0 {
			commit = args[0]
		}

		if sessionIDFlag == "" && !autoFlag {
			fmt.Fprintln(os.Stderr, "error: must specify --session-id or --auto")
			os.Exit(1)
		}

		if err := annotateCloudCommit(commit, sessionIDFlag, autoFlag, noScrubFlag); err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	annotateCloudCmd.Flags().StringVar(&sessionIDFlag, "session-id", "", "Cloud session ID to attach")
	annotateCloudCmd.Flags().BoolVar(&autoFlag, "auto", false, "Auto-detect session from branch name")
	annotateCloudCmd.Flags().BoolVar(&noScrubFlag, "no-scrub", false, "Disable PII scrubbing")
	rootCmd.AddCommand(annotateCloudCmd)
}

func annotateCloudCommit(commitRef, sessionID string, autoDetect, noScrub bool) error {
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

	// Update transcript tree (similar to note.UpdateTranscriptTree but for claude-cloud)
	if err := updateCloudTranscriptTree(sess.ID, blobSHA); err != nil {
		return fmt.Errorf("failed to update transcript tree: %w", err)
	}

	// Create PromptStoryNote using main's format
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
	if err := git.AddNote("refs/notes/prompt-story", string(noteJSON), sha); err != nil {
		return fmt.Errorf("failed to attach note: %w", err)
	}

	fmt.Printf("Successfully annotated commit %s with cloud session %s\n", sha[:7], sess.ID)
	return nil
}

// updateCloudTranscriptTree adds a cloud session transcript to the tree
func updateCloudTranscriptTree(sessionID, blobSHA string) error {
	transcriptRef := "refs/notes/prompt-story-transcripts"

	// Build entry for this session
	newEntry := git.TreeEntry{
		Mode: "100644",
		Type: "blob",
		SHA:  blobSHA,
		Name: sessionID + ".jsonl",
	}

	// Get existing cloud entries
	var cloudEntries []git.TreeEntry
	existingTreeSHA, _ := git.GetRef(transcriptRef)
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

	return git.UpdateRef(transcriptRef, rootTreeSHA)
}

// listCloudSessionsCmd lists available cloud sessions
var listCloudSessionsCmd = &cobra.Command{
	Use:   "list-cloud",
	Short: "List available Claude Code Cloud sessions",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := cloud.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		resp, err := client.ListSessions(20)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Recent Claude Code Cloud sessions:\n\n")
		for _, sess := range resp.Data {
			branch := ""
			for _, outcome := range sess.SessionContext.Outcomes {
				if len(outcome.GitInfo.Branches) > 0 {
					branch = outcome.GitInfo.Branches[0]
					break
				}
			}
			fmt.Printf("  %s\n", sess.ID)
			fmt.Printf("    Title:   %s\n", sess.Title)
			fmt.Printf("    Created: %s\n", sess.CreatedAt.Local().Format("2006-01-02 15:04"))
			if branch != "" {
				fmt.Printf("    Branch:  %s\n", branch)
			}
			fmt.Println()
		}
	},
}

func init() {
	rootCmd.AddCommand(listCloudSessionsCmd)
}
