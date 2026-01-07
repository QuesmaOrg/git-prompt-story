package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/hooks"
	"github.com/spf13/cobra"
)

var (
	installGlobalFlag   bool
	installAutoPushFlag bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install git hooks for prompt capture",
	Long: `Install git hooks to automatically capture LLM sessions on commit.

By default, installs hooks in the current repository.
Use --global to install hooks globally for all repositories.
Use --auto-push to also install a pre-push hook that syncs notes.`,
	Run: func(cmd *cobra.Command, args []string) {
		opts := hooks.InstallOptions{
			Global:   installGlobalFlag,
			AutoPush: installAutoPushFlag,
		}
		if err := hooks.InstallHooks(opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	installCmd.Flags().BoolVar(&installGlobalFlag, "global", false, "Install hooks globally")
	installCmd.Flags().BoolVar(&installAutoPushFlag, "auto-push", false, "Install pre-push hook to auto-sync notes")
	rootCmd.AddCommand(installCmd)
}
