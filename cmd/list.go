package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/cloud"
	"github.com/spf13/cobra"
)

var listSourceFlag string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available sessions",
	Long: `List available LLM sessions from external sources.

Sources:
  cloud   - Claude Code Cloud sessions (requires ANTHROPIC_API_KEY)

Examples:
  git-prompt-story list --source=cloud`,
	Run: func(cmd *cobra.Command, args []string) {
		switch listSourceFlag {
		case "cloud":
			if err := listCloudSessions(); err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "error: unknown source: %s (valid: cloud)\n", listSourceFlag)
			os.Exit(1)
		}
	},
}

func init() {
	listCmd.Flags().StringVar(&listSourceFlag, "source", "", "Session source: cloud (required)")
	listCmd.MarkFlagRequired("source")
	rootCmd.AddCommand(listCmd)
}

func listCloudSessions() error {
	client, err := cloud.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize cloud client: %w", err)
	}

	resp, err := client.ListSessions(20)
	if err != nil {
		return err
	}

	fmt.Printf("Recent Claude Code Cloud sessions:\n\n")
	for _, sess := range resp.Data {
		branch := ""
		for _, outcome := range sess.SessionContext.Outcomes {
			if len(outcome.GitInfo.Branches) > 0 {
				branch = outcome.GitInfo.Branches[0]
				break
			}
		}
		fmt.Printf("  %s\n", sess.ID)
		fmt.Printf("    Title:   %s\n", sess.Title)
		fmt.Printf("    Created: %s\n", sess.CreatedAt.Local().Format("2006-01-02 15:04"))
		if branch != "" {
			fmt.Printf("    Branch:  %s\n", branch)
		}
		fmt.Println()
	}
	return nil
}
