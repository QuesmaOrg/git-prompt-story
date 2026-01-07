package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/hooks"
	"github.com/spf13/cobra"
)

// Backward-compatible alias for "hook pre-push"
var prePushCmd = &cobra.Command{
	Use:        "pre-push <remote-name> <remote-url>",
	Short:      "Hook: Push notes to remote",
	Hidden:     true, // Internal command for git hook (use "hook pre-push" instead)
	Deprecated: "use 'git-prompt-story hook pre-push' instead",
	Args:       cobra.ExactArgs(2),
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

func init() {
	rootCmd.AddCommand(prePushCmd)
}
