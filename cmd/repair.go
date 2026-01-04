package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/repair"
	"github.com/spf13/cobra"
)

var (
	repairDryRun  bool
	repairForce   bool
	repairNoScrub bool
	repairScan    bool
)

var repairCmd = &cobra.Command{
	Use:   "repair [commit|range]",
	Short: "Repair missing prompt-story notes from local sessions",
	Long: `Recreates notes for commits that have local Claude Code sessions
but missing git notes. Useful when notes were lost or never pushed.

Examples:
  # Repair a single commit
  git-prompt-story repair HEAD
  git-prompt-story repair abc1234

  # Repair a range of commits
  git-prompt-story repair HEAD~5..HEAD

  # Scan and repair all commits with Prompt-Story markers but no notes
  git-prompt-story repair --scan

  # Preview what would be repaired
  git-prompt-story repair --dry-run HEAD`,
	Run: func(cmd *cobra.Command, args []string) {
		opts := repair.Options{
			DryRun:  repairDryRun,
			Force:   repairForce,
			NoScrub: repairNoScrub,
		}

		var commits []string
		var err error

		if repairScan {
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
				fmt.Println("No commits found needing repair.")
				return
			}
			fmt.Printf("Found %d commit(s) needing repair:\n", len(commits))
		} else if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "error: specify a commit, range, or use --scan")
			os.Exit(1)
		} else if strings.Contains(args[0], "..") {
			// Range mode
			commits, err = parseCommitRange(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Single commit
			commits = []string{args[0]}
		}

		// Process each commit
		var repaired, skipped, failed int
		for _, sha := range commits {
			result, err := repair.RepairCommit(sha, opts)
			if err != nil {
				fmt.Printf("  %s: ERROR - %v\n", sha[:7], err)
				failed++
				continue
			}

			if result.AlreadyHasNote && !opts.Force {
				fmt.Printf("  %s: skipped (already has note)\n", result.ShortSHA)
				skipped++
				continue
			}

			if result.SessionsFound == 0 {
				fmt.Printf("  %s: skipped (no sessions found for work period)\n", result.ShortSHA)
				skipped++
				continue
			}

			if opts.DryRun {
				fmt.Printf("  %s: would create note (%d sessions)\n", result.ShortSHA, result.SessionsFound)
			} else {
				fmt.Printf("  %s: created note (%d sessions, sha: %s)\n",
					result.ShortSHA, result.SessionsFound, result.NoteSHA[:7])
			}

			repaired++
		}

		// Summary
		fmt.Println()
		if opts.DryRun {
			fmt.Printf("Dry run: %d would be repaired, %d skipped, %d failed\n", repaired, skipped, failed)
		} else {
			fmt.Printf("Done: %d repaired, %d skipped, %d failed\n", repaired, skipped, failed)
		}

		if repaired > 0 && !opts.DryRun {
			fmt.Println("\nRemember to push your notes:")
			fmt.Println("  git push origin refs/notes/prompt-story +refs/notes/prompt-story-transcripts")
		}
	},
}

func parseCommitRange(rangeSpec string) ([]string, error) {
	output, err := git.RunGit("rev-list", rangeSpec)
	if err != nil {
		return nil, fmt.Errorf("invalid range %s: %w", rangeSpec, err)
	}
	if output == "" {
		return nil, fmt.Errorf("no commits in range %s", rangeSpec)
	}
	return strings.Split(strings.TrimSpace(output), "\n"), nil
}

func init() {
	repairCmd.Flags().BoolVar(&repairDryRun, "dry-run", false, "Preview without making changes")
	repairCmd.Flags().BoolVar(&repairForce, "force", false, "Overwrite existing notes")
	repairCmd.Flags().BoolVar(&repairNoScrub, "no-scrub", false, "Disable PII scrubbing")
	repairCmd.Flags().BoolVar(&repairScan, "scan", false, "Scan for commits needing repair")
	rootCmd.AddCommand(repairCmd)
}
