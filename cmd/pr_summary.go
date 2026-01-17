package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/ci"
	"github.com/spf13/cobra"
)

var (
	prSummaryFull     bool
	prSummaryPagesURL string
	prSummaryOutput   string
	prSummaryGHA      bool
)

var prSummaryCmd = &cobra.Command{
	Use:   "summary <commit-range>",
	Short: "Generate summary for commits",
	Long: `Generate a summary of LLM sessions for commits in a range.

This command is designed for CI/CD pipelines to create PR comments or reports.

Examples:
  git-prompt-story pr summary HEAD~5..HEAD
  git-prompt-story pr summary main..feature-branch --pages-url=https://example.github.io/repo/pr-42/
  git-prompt-story pr summary origin/main..HEAD --gha --output=summary.md`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		commitRange := args[0]

		summary, err := ci.GenerateSummary(commitRange, prSummaryFull)
		if err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
			os.Exit(1)
		}

		if prSummaryGHA {
			// GitHub Actions mode: output metadata to stdout
			shouldPost := summary.CommitsWithNotes > 0
			notesMissing := summary.CommitsMissingNotes > 0
			fmt.Printf("commits-analyzed=%d\n", summary.CommitsAnalyzed)
			fmt.Printf("commits-with-notes=%d\n", summary.CommitsWithNotes)
			fmt.Printf("commits-missing-notes=%d\n", summary.CommitsMissingNotes)
			fmt.Printf("notes-missing=%t\n", notesMissing)
			fmt.Printf("should-post-comment=%t\n", shouldPost || notesMissing)

			// Write markdown to file
			if prSummaryOutput != "" {
				var markdown string
				if shouldPost {
					markdown = ci.RenderMarkdown(summary, prSummaryPagesURL, GetVersion())
				} else if notesMissing {
					markdown = ci.RenderMissingNotesWarning(summary.CommitsMissingNotes, GetVersion())
				}
				if markdown != "" {
					if err := os.WriteFile(prSummaryOutput, []byte(markdown), 0644); err != nil {
						fmt.Fprintf(os.Stderr, "git-prompt-story: failed to write output: %v\n", err)
						os.Exit(1)
					}
				}
			}
			return
		}

		// Normal mode: output markdown
		output := ci.RenderMarkdown(summary, prSummaryPagesURL, GetVersion())

		if prSummaryOutput != "" {
			if err := os.WriteFile(prSummaryOutput, []byte(output), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: failed to write output: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Print(output)
		}
	},
}

func init() {
	prSummaryCmd.Flags().BoolVar(&prSummaryFull, "full", false, "Include full prompt text (not truncated)")
	prSummaryCmd.Flags().StringVar(&prSummaryPagesURL, "pages-url", "", "URL to GitHub Pages transcripts")
	prSummaryCmd.Flags().StringVar(&prSummaryOutput, "output", "", "Write markdown to file instead of stdout")
	prSummaryCmd.Flags().BoolVar(&prSummaryGHA, "gha", false, "GitHub Actions mode: output metadata to stdout")
	prCmd.AddCommand(prSummaryCmd)
}
