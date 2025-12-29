package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/hooks"
	"github.com/spf13/cobra"
)

var globalFlag bool

var installHooksCmd = &cobra.Command{
	Use:   "install-hooks",
	Short: "Install git hooks for prompt capture",
	Long: `Install git hooks to automatically capture LLM sessions on commit.

By default, installs hooks in the current repository.
Use --global to install hooks globally for all repositories.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := hooks.InstallHooks(globalFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	installHooksCmd.Flags().BoolVar(&globalFlag, "global", false, "Install hooks globally")
	rootCmd.AddCommand(installHooksCmd)
}
