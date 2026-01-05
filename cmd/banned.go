package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/banned"
	"github.com/spf13/cobra"
)

var bannedCmd = &cobra.Command{
	Use:   "banned",
	Short: "List banned sessions",
	Long:  `Show all sessions that have been banned from future captures.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runBanned(); err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(bannedCmd)
}

func runBanned() error {
	list, err := banned.Load()
	if err != nil {
		return fmt.Errorf("loading banned list: %w", err)
	}

	if len(list.Banned) == 0 {
		fmt.Println("No banned sessions.")
		return nil
	}

	fmt.Printf("Banned sessions (%d):\n\n", len(list.Banned))
	for _, s := range list.Banned {
		fmt.Printf("  %s/%s\n", s.Tool, s.ID)
		fmt.Printf("    Banned: %s\n", s.BannedAt.Format("2006-01-02 15:04"))
		if s.Reason != "" {
			fmt.Printf("    Reason: %s\n", s.Reason)
		}
	}

	return nil
}
