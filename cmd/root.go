package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var version = "dev"

func SetVersionInfo(v, commit, date string) {
	version = v

	// Build version string with optional commit and date
	var parts []string
	parts = append(parts, v)
	if commit != "" {
		parts = append(parts, commit)
	}
	if date != "" {
		// Shorten ISO date to just the date part if it's a full timestamp
		if len(date) > 10 {
			date = date[:10]
		}
		parts = append(parts, date)
	}

	rootCmd.Version = strings.Join(parts, " ")
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
