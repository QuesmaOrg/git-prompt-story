package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/cloud"
	"github.com/spf13/cobra"
)

var (
	annotateCloudSessionIDFlag string
	annotateCloudAutoFlag      bool
	annotateCloudNoScrubFlag   bool
)

// Backward-compatible alias for "add --source=cloud"
var annotateCloudCmd = &cobra.Command{
	Use:        "annotate-cloud [commit]",
	Short:      "Annotate a commit with a Claude Code Cloud session",
	Hidden:     true, // Hidden, use "add --source=cloud" instead
	Deprecated: "use 'git-prompt-story add --source=cloud' instead",
	Args:       cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		commit := "HEAD"
		if len(args) > 0 {
			commit = args[0]
		}

		if annotateCloudSessionIDFlag == "" && !annotateCloudAutoFlag {
			fmt.Fprintln(os.Stderr, "error: must specify --session-id or --auto")
			os.Exit(1)
		}

		// Forward to add command with cloud source
		addSessionIDFlag = annotateCloudSessionIDFlag
		addAutoFlag = annotateCloudAutoFlag
		addNoScrubFlag = annotateCloudNoScrubFlag

		result, err := addToCommit(commit, "cloud")
		if err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
			os.Exit(1)
		}

		if result.Skipped {
			fmt.Printf("  %s: skipped (%s)\n", result.ShortSHA, result.Reason)
		} else {
			fmt.Printf("  %s: added %s session (%s)\n", result.ShortSHA, result.Source, result.SessionID)
		}
	},
}

func init() {
	annotateCloudCmd.Flags().StringVar(&annotateCloudSessionIDFlag, "session-id", "", "Cloud session ID to attach")
	annotateCloudCmd.Flags().BoolVar(&annotateCloudAutoFlag, "auto", false, "Auto-detect session from branch name")
	annotateCloudCmd.Flags().BoolVar(&annotateCloudNoScrubFlag, "no-scrub", false, "Disable PII scrubbing")
	rootCmd.AddCommand(annotateCloudCmd)
}

// Backward-compatible alias for "list --source=cloud"
var listCloudSessionsCmd = &cobra.Command{
	Use:        "list-cloud",
	Short:      "List available Claude Code Cloud sessions",
	Hidden:     true, // Hidden, use "list --source=cloud" instead
	Deprecated: "use 'git-prompt-story list --source=cloud' instead",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := cloud.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		resp, err := client.ListSessions(20)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
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
	},
}

func init() {
	rootCmd.AddCommand(listCloudSessionsCmd)
}
