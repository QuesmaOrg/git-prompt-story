package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

var rootCmd = &cobra.Command{
	Use:   "git-prompt-story",
	Short: "Capture LLM sessions alongside git commits",
	Long: `git-prompt-story captures LLM sessions (Claude Code, Cursor, etc.)
and stores them as git notes attached to your commits.`,
	Version: version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
