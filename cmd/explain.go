package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/explain"
	"github.com/spf13/cobra"
)

var explainAllFlag bool

var explainCmd = &cobra.Command{
	Use:   "explain [commit]",
	Short: "Explain session discovery and filtering decisions",
	Long: `Explain how git-prompt-story discovers and filters Claude Code sessions.

Shows the decision process for what sessions would be included for a commit,
including:
- Where sessions are searched for
- How the work period is calculated
- Why each session was included or excluded

By default explains decisions for HEAD. Optionally specify a commit to
explain decisions relative to that commit's timestamp.

Use --all to show details for every session (including excluded ones).`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		commit := "HEAD"
		if len(args) > 0 {
			commit = args[0]
		}
		opts := explain.ExplainOptions{
			ShowAll: explainAllFlag,
		}
		if err := explain.Explain(commit, opts, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	explainCmd.Flags().BoolVar(&explainAllFlag, "all", false, "Show all sessions including excluded ones")
	rootCmd.AddCommand(explainCmd)
}
