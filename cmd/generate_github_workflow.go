package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/workflow"
	"github.com/spf13/cobra"
)

// Backward-compatible alias for "workflow"
var generateGitHubWorkflowCmd = &cobra.Command{
	Use:        "generate-github-workflow",
	Short:      "Generate GitHub Action workflow for prompt-story",
	Hidden:     true, // Hidden, use "workflow" instead
	Deprecated: "use 'git-prompt-story workflow' instead",
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
