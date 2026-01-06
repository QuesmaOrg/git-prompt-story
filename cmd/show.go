package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/show"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	fullFlag          bool
	interactiveFlag   bool
	noInteractiveFlag bool
)

var showCmd = &cobra.Command{
	Use:   "show [commit]",
	Short: "Show prompts for a commit",
	Long: `Display LLM prompts and sessions attached to a commit or commit range.

By default, opens an interactive TUI viewer when running in a terminal.
Use --no-interactive for plain text output (useful for piping).
Use --full to display complete message content.

Examples:
  git-prompt-story show                # Show prompts for HEAD
  git-prompt-story show abc123         # Show prompts for specific commit
  git-prompt-story show HEAD~5..HEAD   # Show prompts for commit range`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		commit := "HEAD"
		if len(args) > 0 {
			commit = args[0]
		}

		// Determine if we should use interactive mode
		isTTY := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
		useInteractive := (interactiveFlag || isTTY) && !noInteractiveFlag

		if useInteractive {
			if err := show.RunTUI(commit, fullFlag); err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
				os.Exit(1)
			}
		} else {
			if err := show.ShowPrompts(commit, fullFlag); err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	showCmd.Flags().BoolVar(&fullFlag, "full", false, "Show full message content")
	showCmd.Flags().BoolVarP(&interactiveFlag, "interactive", "i", false, "Force interactive TUI mode")
	showCmd.Flags().BoolVar(&noInteractiveFlag, "no-interactive", false, "Disable interactive TUI, use plain text output")
	rootCmd.AddCommand(showCmd)
}
