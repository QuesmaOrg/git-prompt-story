package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/hooks"
	"github.com/spf13/cobra"
)

var (
	installHooksGlobalFlag   bool
	installHooksAutoPushFlag bool
)

// Backward-compatible alias for "install"
var installHooksCmd = &cobra.Command{
	Use:        "install-hooks",
	Short:      "Install git hooks for prompt capture",
	Hidden:     true, // Hidden, use "install" instead
	Deprecated: "use 'git-prompt-story install' instead",
	Run: func(cmd *cobra.Command, args []string) {
		opts := hooks.InstallOptions{
			Global:   installHooksGlobalFlag,
			AutoPush: installHooksAutoPushFlag,
		}
		if err := hooks.InstallHooks(opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	installHooksCmd.Flags().BoolVar(&installHooksGlobalFlag, "global", false, "Install hooks globally")
	installHooksCmd.Flags().BoolVar(&installHooksAutoPushFlag, "auto-push", false, "Install pre-push hook to auto-sync notes")
	rootCmd.AddCommand(installHooksCmd)
}
