package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/hooks"
	"github.com/spf13/cobra"
)

// hookCmd is the parent command for all git hook subcommands
var hookCmd = &cobra.Command{
	Use:    "hook",
	Short:  "Git hook commands (internal)",
	Hidden: true, // Hide from main help since these are internal
}

// Hook subcommands
var hookPrepareCommitMsgCmd = &cobra.Command{
	Use:   "prepare-commit-msg <commit-msg-file> [source] [sha]",
	Short: "Hook: Prepare commit message with prompt info",
	Args:  cobra.MinimumNArgs(1),
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

var hookPostCommitCmd = &cobra.Command{
	Use:   "post-commit",
	Short: "Hook: Attach prompt note to commit",
	Run: func(cmd *cobra.Command, args []string) {
		if err := hooks.PostCommit(); err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
			// Don't exit with error to not block git
		}
	},
}

var hookPrePushCmd = &cobra.Command{
	Use:   "pre-push <remote-name> <remote-url>",
	Short: "Hook: Push notes to remote",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		remoteName := args[0]
		remoteURL := args[1]
		if err := hooks.PrePush(remoteName, remoteURL, os.Stdin); err != nil {
			// Print warning but don't exit with error - notes push failure
			// should not block the main push
			fmt.Fprintf(os.Stderr, "git-prompt-story: warning: %v\n", err)
		}
	},
}

var hookPostRewriteCmd = &cobra.Command{
	Use:   "post-rewrite [rebase|amend]",
	Short: "Hook: Transfer notes after rebase/amend",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rewriteType := args[0]
		if err := hooks.PostRewrite(rewriteType, os.Stdin); err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
			// Don't exit with error to not block git
		}
	},
}

func init() {
	// Add hook parent command
	rootCmd.AddCommand(hookCmd)

	// Add subcommands to hook parent
	hookCmd.AddCommand(hookPrepareCommitMsgCmd)
	hookCmd.AddCommand(hookPostCommitCmd)
	hookCmd.AddCommand(hookPrePushCmd)
	hookCmd.AddCommand(hookPostRewriteCmd)
}
