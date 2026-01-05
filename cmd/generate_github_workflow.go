package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/workflow"
	"github.com/spf13/cobra"
)

var generateGitHubWorkflowCmd = &cobra.Command{
	Use:   "generate-github-workflow",
	Short: "Generate GitHub Action workflow for prompt-story",
	Long: `Generate a GitHub Action workflow file that analyzes LLM sessions
and posts summaries on pull requests.

The command will prompt you to enable GitHub Pages for full transcripts.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := workflow.Generate(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(generateGitHubWorkflowCmd)
}
