package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/QuesmaOrg/git-prompt-story/internal/banned"
	"github.com/spf13/cobra"
)

var unbanAllFlag bool

var unbanCmd = &cobra.Command{
	Use:   "unban",
	Short: "Unban sessions to allow future captures",
	Long: `Interactively unban sessions that were previously banned.

Without arguments, shows a list of banned sessions and allows you to select
which ones to unban.

Use --all to unban all sessions at once.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runUnban(); err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	unbanCmd.Flags().BoolVar(&unbanAllFlag, "all", false, "Unban all sessions")
	rootCmd.AddCommand(unbanCmd)
}

func runUnban() error {
	reader := bufio.NewReader(os.Stdin)

	// Load banned list
	list, err := banned.Load()
	if err != nil {
		return fmt.Errorf("loading banned list: %w", err)
	}

	if len(list.Banned) == 0 {
		fmt.Println("No banned sessions.")
		return nil
	}

	// Display banned sessions
	fmt.Println("Banned sessions:")
	fmt.Println()
	for i, s := range list.Banned {
		fmt.Printf("  [%d] %s/%s\n", i+1, s.Tool, s.ID)
		fmt.Printf("      Banned: %s\n", s.BannedAt.Format("2006-01-02 15:04"))
		if s.Reason != "" {
			fmt.Printf("      Reason: %q\n", s.Reason)
		}
		fmt.Println()
	}

	// Select sessions to unban
	var selectedIndices []int
	if unbanAllFlag {
		for i := range list.Banned {
			selectedIndices = append(selectedIndices, i)
		}
	} else {
		fmt.Print("? Which sessions to unban? [1")
		if len(list.Banned) > 1 {
			fmt.Printf("-%d,all,none", len(list.Banned))
		}
		fmt.Print("]: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "none" || input == "" {
			fmt.Println("No sessions selected.")
			return nil
		}

		if input == "all" {
			for i := range list.Banned {
				selectedIndices = append(selectedIndices, i)
			}
		} else {
			// Parse comma-separated numbers
			for _, part := range strings.Split(input, ",") {
				part = strings.TrimSpace(part)
				num, err := strconv.Atoi(part)
				if err != nil || num < 1 || num > len(list.Banned) {
					fmt.Printf("Invalid selection: %s\n", part)
					continue
				}
				selectedIndices = append(selectedIndices, num-1)
			}
		}
	}

	if len(selectedIndices) == 0 {
		fmt.Println("No sessions selected.")
		return nil
	}

	// Unban selected sessions (in reverse order to maintain indices)
	// Sort indices in reverse order
	for i := len(selectedIndices) - 1; i >= 0; i-- {
		idx := selectedIndices[i]
		s := list.Banned[idx]
		if err := banned.Unban(s.ID); err != nil {
			fmt.Printf("  ⚠ Failed to unban %s: %v\n", s.ID[:8], err)
		} else {
			fmt.Printf("✓ Session %s unbanned. It may be captured in future commits.\n", s.ID[:8])
		}
	}

	return nil
}
