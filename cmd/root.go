package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/QuesmaOrg/git-prompt-story/internal/show"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var version = "dev"

func SetVersionInfo(v, commit, date string) {
	version = v

	// Build version string with optional commit and date
	var parts []string
	parts = append(parts, v)
	if commit != "" {
		parts = append(parts, commit)
	}
	if date != "" {
		// Shorten ISO date to just the date part if it's a full timestamp
		if len(date) > 10 {
			date = date[:10]
		}
		parts = append(parts, date)
	}

	rootCmd.Version = strings.Join(parts, " ")
}

func GetVersion() string {
	return version
}

var rootCmd = &cobra.Command{
	Use:   "git-prompt-story [commit]",
	Short: "Capture LLM sessions alongside git commits",
	Long: `git-prompt-story captures LLM sessions (Claude Code, Cursor, etc.)
and stores them as git notes attached to your commits.

When called without a subcommand, shows prompts for the specified commit (or HEAD).

Examples:
  git-prompt-story                    # Show prompts for HEAD
  git-prompt-story HEAD~5             # Show prompts for specific commit
  git-prompt-story show HEAD~5..HEAD  # Show prompts for commit range`,
	Version: version,
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Default action: show prompts
		commit := "HEAD"
		if len(args) > 0 {
			commit = args[0]
		}

		// Determine if we should use interactive mode
		isTTY := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
		useInteractive := isTTY

		if useInteractive {
			if err := show.RunTUI(commit, false); err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
				os.Exit(1)
			}
		} else {
			if err := show.ShowPrompts(commit, false); err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
