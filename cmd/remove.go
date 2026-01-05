package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/QuesmaOrg/git-prompt-story/internal/banned"
	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/note"
	"github.com/spf13/cobra"
)

var (
	removeYesFlag  bool
	removeBanFlag  bool
	removePushFlag bool
)

var removeCmd = &cobra.Command{
	Use:   "remove <commit-or-range>",
	Short: "Remove sessions from commits",
	Long: `Remove LLM sessions from commits and optionally ban them from future captures.

Examples:
  git-prompt-story remove HEAD          # Remove sessions from current commit
  git-prompt-story remove abc123        # Remove from specific commit
  git-prompt-story remove HEAD~5..HEAD  # Remove from range of commits
  git-prompt-story remove main..feature # Remove from branch commits

Interactive prompts guide you through:
  1. Which sessions to remove
  2. Whether to push changes to remote
  3. Whether to ban sessions from future captures`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runRemove(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	removeCmd.Flags().BoolVar(&removeYesFlag, "yes", false, "Skip confirmation prompts")
	removeCmd.Flags().BoolVar(&removeBanFlag, "ban", false, "Ban sessions from future captures")
	removeCmd.Flags().BoolVar(&removePushFlag, "push", false, "Push changes to remote")
	rootCmd.AddCommand(removeCmd)
}

func runRemove(commitSpec string) error {
	reader := bufio.NewReader(os.Stdin)

	// Resolve commits
	commits, err := git.ResolveCommitSpec(commitSpec)
	if err != nil {
		return err
	}

	fmt.Printf("Scanning %d commit(s) for sessions...\n\n", len(commits))

	// Find all sessions in these commits
	sessions, err := note.FindSessionsInCommits(commits)
	if err != nil {
		return fmt.Errorf("scanning commits: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found in the specified commits.")
		return nil
	}

	// Filter out already-removed sessions for display, but track them
	var activeSessions []*note.SessionCommitInfo
	var removedCount int
	for _, s := range sessions {
		if s.Removed {
			removedCount++
		} else {
			activeSessions = append(activeSessions, s)
		}
	}

	// Sort by created time
	sort.Slice(activeSessions, func(i, j int) bool {
		return activeSessions[i].Created.Before(activeSessions[j].Created)
	})

	if len(activeSessions) == 0 {
		fmt.Printf("All %d session(s) are already removed.\n", removedCount)
		return nil
	}

	// Display sessions
	fmt.Printf("Found %d session(s) across %d commit(s)", len(activeSessions), len(commits))
	if removedCount > 0 {
		fmt.Printf(" (%d already removed)", removedCount)
	}
	fmt.Println(":")
	fmt.Println()

	for i, s := range activeSessions {
		fmt.Printf("  [%d] %s/%s\n", i+1, s.Tool, s.SessionID)
		fmt.Printf("      Created:  %s\n", s.Created.Format("2006-01-02 15:04"))
		fmt.Printf("      Modified: %s\n", s.Modified.Format("2006-01-02 15:04"))
		fmt.Printf("      Commits:  %s\n", formatCommitList(s.Commits))
		fmt.Println()
	}

	// Ask which sessions to remove
	var selectedSessions []*note.SessionCommitInfo
	if removeYesFlag {
		selectedSessions = activeSessions
	} else {
		selected, err := promptSessionSelection(reader, activeSessions)
		if err != nil {
			return err
		}
		selectedSessions = selected
	}

	if len(selectedSessions) == 0 {
		fmt.Println("No sessions selected.")
		return nil
	}

	// Process each selected session
	for _, s := range selectedSessions {
		if err := removeSession(reader, s); err != nil {
			return err
		}
	}

	fmt.Println("\nDone.")
	return nil
}

func promptSessionSelection(reader *bufio.Reader, sessions []*note.SessionCommitInfo) ([]*note.SessionCommitInfo, error) {
	fmt.Print("? Which sessions to remove? [1")
	if len(sessions) > 1 {
		fmt.Printf("-%d,all,none", len(sessions))
	}
	fmt.Print("]: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "none" || input == "" {
		return nil, nil
	}

	if input == "all" {
		return sessions, nil
	}

	// Parse comma-separated numbers
	var selected []*note.SessionCommitInfo
	for _, part := range strings.Split(input, ",") {
		part = strings.TrimSpace(part)
		num, err := strconv.Atoi(part)
		if err != nil || num < 1 || num > len(sessions) {
			fmt.Printf("Invalid selection: %s\n", part)
			continue
		}
		selected = append(selected, sessions[num-1])
	}

	return selected, nil
}

func removeSession(reader *bufio.Reader, s *note.SessionCommitInfo) error {
	fmt.Printf("\n--- Removing session %s ---\n", s.SessionID[:8])

	// Confirm removal
	if !removeYesFlag {
		fmt.Printf("? Remove session %s from %d commit(s)? [Y/n]: ", s.SessionID[:8], len(s.Commits))
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "n" || input == "no" {
			fmt.Println("Skipped.")
			return nil
		}
	}

	// Mark as removed in all commits
	fmt.Println("Removing from local notes...")
	for _, sha := range s.Commits {
		marked, err := note.MarkSessionRemoved(sha, s.SessionID)
		if err != nil {
			return fmt.Errorf("marking session removed in %s: %w", sha[:8], err)
		}
		if marked {
			fmt.Printf("  ✓ Removed from commit %s\n", sha[:8])
		}
	}

	// Remove from transcript tree
	if err := note.RemoveSessionFromTranscripts(s.SessionID, s.Tool); err != nil {
		return fmt.Errorf("removing transcript: %w", err)
	}
	fmt.Println("  ✓ Removed from transcript tree")

	// Check remote
	remoteName := "origin" // TODO: make configurable
	notesOnRemote := git.RefExistsOnRemote(remoteName, note.NotesRef)
	transcriptsOnRemote := git.RefExistsOnRemote(remoteName, note.TranscriptsRef)

	if notesOnRemote || transcriptsOnRemote {
		shouldPush := removePushFlag
		if !shouldPush && !removeYesFlag {
			fmt.Printf("\nSession exists on remote %s.\n", remoteName)
			fmt.Print("? Also remove from remote? (requires force push) [y/N]: ")
			input, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			input = strings.TrimSpace(strings.ToLower(input))
			shouldPush = input == "y" || input == "yes"
		}

		if shouldPush {
			fmt.Println("\nPushing to remote...")
			if notesOnRemote {
				if err := git.ForcePushRef(remoteName, note.NotesRef); err != nil {
					fmt.Printf("  ⚠ Warning: failed to push %s: %v\n", note.NotesRef, err)
				} else {
					fmt.Printf("  ✓ Updated %s\n", note.NotesRef)
				}
			}
			if transcriptsOnRemote {
				if err := git.ForcePushRef(remoteName, note.TranscriptsRef); err != nil {
					fmt.Printf("  ⚠ Warning: failed to push %s: %v\n", note.TranscriptsRef, err)
				} else {
					fmt.Printf("  ✓ Updated %s\n", note.TranscriptsRef)
				}
			}
		}
	}

	// Ask about banning
	shouldBan := removeBanFlag
	if !shouldBan && !removeYesFlag {
		fmt.Print("\n? Ban this session from future captures? [Y/n]: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		input = strings.TrimSpace(strings.ToLower(input))
		shouldBan = input != "n" && input != "no"
	}

	if shouldBan {
		if err := banned.Ban(s.SessionID, s.Tool, ""); err != nil {
			return fmt.Errorf("banning session: %w", err)
		}
		fmt.Printf("  ✓ Session banned from future captures\n")
	}

	return nil
}

func formatCommitList(commits []string) string {
	if len(commits) == 0 {
		return "(none)"
	}

	var short []string
	for _, c := range commits {
		if len(c) >= 7 {
			short = append(short, c[:7])
		} else {
			short = append(short, c)
		}
	}

	if len(short) <= 3 {
		return strings.Join(short, ", ")
	}
	return fmt.Sprintf("%s, ... (%d total)", strings.Join(short[:2], ", "), len(short))
}
