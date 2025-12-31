package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/hooks"
	"github.com/spf13/cobra"
)

var postRewriteCmd = &cobra.Command{
	Use:    "post-rewrite [rebase|amend]",
	Short:  "Hook: Transfer notes after rebase/amend",
	Hidden: true, // Internal command for git hook
	Args:   cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rewriteType := args[0]
		if err := hooks.PostRewrite(rewriteType, os.Stdin); err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
			// Don't exit with error to not block git
		}
	},
}

func init() {
	rootCmd.AddCommand(postRewriteCmd)
}
