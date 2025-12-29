package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/show"
	"github.com/spf13/cobra"
)

var fullFlag bool

var showCmd = &cobra.Command{
	Use:   "show [commit]",
	Short: "Show prompts for a commit",
	Long: `Display LLM prompts and sessions attached to a commit.

By default, shows prompts for HEAD. Optionally specify a commit hash or reference.
Use --full to display complete message content instead of summaries.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		commit := "HEAD"
		if len(args) > 0 {
			commit = args[0]
		}
		if err := show.ShowPrompts(commit, fullFlag); err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	showCmd.Flags().BoolVar(&fullFlag, "full", false, "Show full message content")
	rootCmd.AddCommand(showCmd)
}
