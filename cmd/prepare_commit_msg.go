package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/hooks"
	"github.com/spf13/cobra"
)

// Backward-compatible alias for "hook prepare-commit-msg"
var prepareCommitMsgCmd = &cobra.Command{
	Use:        "prepare-commit-msg <commit-msg-file> [source] [sha]",
	Short:      "Hook: Prepare commit message with prompt info",
	Hidden:     true, // Internal command for git hook (use "hook prepare-commit-msg" instead)
	Deprecated: "use 'git-prompt-story hook prepare-commit-msg' instead",
	Args:       cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		msgFile := args[0]
		source := ""
		sha := ""
		if len(args) > 1 {
			source = args[1]
		}
		if len(args) > 2 {
			sha = args[2]
		}

		if err := hooks.PrepareCommitMsg(msgFile, source, sha); err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
			// Don't exit with error to not block the commit
		}
	},
}

func init() {
	rootCmd.AddCommand(prepareCommitMsgCmd)
}
