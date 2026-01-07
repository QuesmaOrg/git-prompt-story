package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/cloud"
	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/note"
	"github.com/QuesmaOrg/git-prompt-story/internal/repair"
	"github.com/QuesmaOrg/git-prompt-story/internal/scrubber"
	"github.com/spf13/cobra"
)

var (
	addSourceFlag    string
	addSessionIDFlag string
	addAutoFlag      bool
	addNoScrubFlag   bool
	addDryRunFlag    bool
	addForceFlag     bool
	addScanFlag      bool
)

var addCmd = &cobra.Command{
	Use:   "add [commit|range]",
	Short: "Add prompt-story notes to commits",
	Long: `Add LLM session notes to commits from cloud or local sessions.

By default, auto-detects the best source:
  1. If cloud credentials exist, tries to find matching cloud session
  2. Falls back to scanning local Claude Code sessions

Use --source to explicitly specify cloud or local.

Examples:
  # Auto-detect source for HEAD
  git-prompt-story add

  # Add to specific commit (auto-detect)
  git-prompt-story add abc123

  # Add to range of commits (local sessions only)
  git-prompt-story add HEAD~5..HEAD

  # Scan for all commits needing notes
  git-prompt-story add --scan

  # Explicitly use cloud session
  git-prompt-story add --source=cloud --auto
  git-prompt-story add --source=cloud --session-id=session_01XXX

  # Explicitly use local sessions
  git-prompt-story add --source=local`,
	Run: func(cmd *cobra.Command, args []string) {
		// Determine commit(s) to process
		var commits []string
		var err error

		if addScanFlag {
			// Scan mode: find commits needing repair
			rangeArg := ""
			if len(args) > 0 {
				rangeArg = args[0]
			}
			commits, err = repair.ScanCommitsNeedingRepair(rangeArg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
				os.Exit(1)
			}
			if len(commits) == 0 {
				fmt.Println("No commits found needing notes.")
				return
			}
			fmt.Printf("Found %d commit(s) needing notes:\n", len(commits))
		} else if len(args) == 0 {
			// Default to HEAD
			commits = []string{"HEAD"}
		} else {
			// Single commit or range
			commits, err = git.ResolveCommitSpec(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
				os.Exit(1)
			}
		}

		// Determine source
		source := addSourceFlag
		if source == "" {
			source = "auto"
		}

		// Process each commit
		var added, skipped, failed int
		for _, commitRef := range commits {
			result, err := addToCommit(commitRef, source)
			if err != nil {
				fmt.Printf("  %s: ERROR - %v\n", shortSHA(commitRef), err)
				failed++
				continue
			}

			if result.Skipped {
				fmt.Printf("  %s: skipped (%s)\n", result.ShortSHA, result.Reason)
				skipped++
				continue
			}

			if addDryRunFlag {
				fmt.Printf("  %s: would add %s session (%s)\n", result.ShortSHA, result.Source, result.SessionID)
			} else {
				fmt.Printf("  %s: added %s session (%s)\n", result.ShortSHA, result.Source, result.SessionID)
			}
			added++
		}

		// Summary
		if len(commits) > 1 {
			fmt.Println()
			if addDryRunFlag {
				fmt.Printf("Dry run: %d would be added, %d skipped, %d failed\n", added, skipped, failed)
			} else {
				fmt.Printf("Done: %d added, %d skipped, %d failed\n", added, skipped, failed)
			}
		}

		if added > 0 && !addDryRunFlag {
			fmt.Println("\nRemember to push your notes:")
			fmt.Println("  git push origin refs/notes/prompt-story +refs/notes/prompt-story-transcripts")
		}
	},
}

type addResult struct {
	ShortSHA  string
	Source    string
	SessionID string
	Skipped   bool
	Reason    string
}

func shortSHA(ref string) string {
	if len(ref) > 7 {
		return ref[:7]
	}
	return ref
}

func addToCommit(commitRef, source string) (*addResult, error) {
	sha, err := git.ResolveCommit(commitRef)
	if err != nil {
		return nil, fmt.Errorf("invalid commit: %w", err)
	}

	result := &addResult{ShortSHA: sha[:7]}

	// Check if already has note (unless --force)
	if !addForceFlag {
		existingNote, _ := git.GetNote(note.NotesRef, sha)
		if existingNote != "" {
			result.Skipped = true
			result.Reason = "already has note, use --force to overwrite"
			return result, nil
		}
	}

	// Handle source
	switch source {
	case "cloud":
		return addCloudToCommit(sha, result)
	case "local":
		return addLocalToCommit(sha, result)
	case "auto":
		return addAutoToCommit(sha, result)
	default:
		return nil, fmt.Errorf("unknown source: %s", source)
	}
}

func addAutoToCommit(sha string, result *addResult) (*addResult, error) {
	// Try cloud first if credentials available
	client, err := cloud.NewClient()
	if err == nil {
		// Cloud available, try to find session
		branchName, _ := git.GetCurrentBranch()
		if branchName != "" {
			sess, err := client.FindSessionByBranch(branchName)
			if err == nil && sess != nil {
				fmt.Printf("Found cloud session for branch %s: %s\n", branchName, sess.Title)
				return addCloudSessionToCommit(sha, sess, result)
			}
		}
	}

	// Fall back to local
	return addLocalToCommit(sha, result)
}

func addCloudToCommit(sha string, result *addResult) (*addResult, error) {
	client, err := cloud.NewClient()
	if err != nil {
		return nil, fmt.Errorf("cloud not available: %w", err)
	}

	var sess *cloud.Session
	if addSessionIDFlag != "" {
		sess, err = client.GetSession(addSessionIDFlag)
		if err != nil {
			return nil, fmt.Errorf("failed to get session: %w", err)
		}
	} else if addAutoFlag {
		branchName, err := git.GetCurrentBranch()
		if err != nil {
			return nil, fmt.Errorf("failed to get branch: %w", err)
		}
		sess, err = client.FindSessionByBranch(branchName)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("cloud source requires --session-id or --auto")
	}

	return addCloudSessionToCommit(sha, sess, result)
}

func addCloudSessionToCommit(sha string, sess *cloud.Session, result *addResult) (*addResult, error) {
	result.Source = "cloud"
	result.SessionID = sess.ID

	if addDryRunFlag {
		return result, nil
	}

	client, err := cloud.NewClient()
	if err != nil {
		return nil, err
	}

	// Fetch events
	events, err := client.GetAllSessionEvents(sess.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	// Convert to JSONL
	jsonl, err := cloud.EventsToJSONL(events, sess)
	if err != nil {
		return nil, fmt.Errorf("failed to convert events: %w", err)
	}

	// Scrub PII
	if !addNoScrubFlag {
		piiScrubber, err := scrubber.NewDefault()
		if err != nil {
			return nil, fmt.Errorf("failed to create scrubber: %w", err)
		}
		jsonl, err = piiScrubber.Scrub(jsonl)
		if err != nil {
			return nil, fmt.Errorf("failed to scrub: %w", err)
		}
	}

	// Store transcript
	blobSHA, err := git.HashObject(jsonl)
	if err != nil {
		return nil, fmt.Errorf("failed to store transcript: %w", err)
	}

	if err := updateCloudTranscriptTreeForAdd(sess.ID, blobSHA); err != nil {
		return nil, fmt.Errorf("failed to update tree: %w", err)
	}

	// Create note
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
	noteJSON, _ := json.MarshalIndent(psNote, "", "  ")

	if err := git.AddNote(note.NotesRef, string(noteJSON), sha); err != nil {
		return nil, fmt.Errorf("failed to attach note: %w", err)
	}

	return result, nil
}

func addLocalToCommit(sha string, result *addResult) (*addResult, error) {
	opts := repair.Options{
		DryRun:  addDryRunFlag,
		Force:   addForceFlag,
		NoScrub: addNoScrubFlag,
	}

	repairResult, err := repair.RepairCommit(sha, opts)
	if err != nil {
		return nil, err
	}

	result.Source = "local"
	result.SessionID = fmt.Sprintf("%d sessions", repairResult.SessionsFound)

	if repairResult.AlreadyHasNote && !opts.Force {
		result.Skipped = true
		result.Reason = "already has note"
		return result, nil
	}

	if repairResult.SessionsFound == 0 {
		result.Skipped = true
		result.Reason = "no local sessions found"
		return result, nil
	}

	return result, nil
}

func init() {
	addCmd.Flags().StringVar(&addSourceFlag, "source", "", "Session source: cloud, local (default: auto-detect)")
	addCmd.Flags().StringVar(&addSessionIDFlag, "session-id", "", "Cloud session ID")
	addCmd.Flags().BoolVar(&addAutoFlag, "auto", false, "Auto-detect cloud session from branch")
	addCmd.Flags().BoolVar(&addNoScrubFlag, "no-scrub", false, "Disable PII scrubbing")
	addCmd.Flags().BoolVar(&addDryRunFlag, "dry-run", false, "Preview without making changes")
	addCmd.Flags().BoolVar(&addForceFlag, "force", false, "Overwrite existing notes")
	addCmd.Flags().BoolVar(&addScanFlag, "scan", false, "Scan for commits needing notes")
	rootCmd.AddCommand(addCmd)
}

// updateCloudTranscriptTreeForAdd adds a cloud session transcript to the tree
func updateCloudTranscriptTreeForAdd(sessionID, blobSHA string) error {
	newEntry := git.TreeEntry{
		Mode: "100644",
		Type: "blob",
		SHA:  blobSHA,
		Name: sessionID + ".jsonl",
	}

	var cloudEntries []git.TreeEntry
	existingTreeSHA, _ := git.GetRef(note.TranscriptsRef)
	if existingTreeSHA != "" {
		rootEntries, err := git.ReadTree(existingTreeSHA)
		if err == nil {
			for _, entry := range rootEntries {
				if entry.Name == "claude-cloud" && entry.Type == "tree" {
					existingCloudEntries, err := git.ReadTree(entry.SHA)
					if err == nil {
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

	cloudTreeSHA, err := git.CreateTree(cloudEntries)
	if err != nil {
		return err
	}

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
