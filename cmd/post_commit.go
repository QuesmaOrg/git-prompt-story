package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/hooks"
	"github.com/spf13/cobra"
)

var postCommitCmd = &cobra.Command{
	Use:    "post-commit",
	Short:  "Hook: Attach prompt note to commit",
	Hidden: true, // Internal command for git hook
	Run: func(cmd *cobra.Command, args []string) {
		if err := hooks.PostCommit(); err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
			// Don't exit with error to not block git
		}
	},
}

func init() {
	rootCmd.AddCommand(postCommitCmd)
}
