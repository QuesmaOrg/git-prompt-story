package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/ci"
	"github.com/spf13/cobra"
)

var (
	summaryFormat   string
	summaryFull     bool
	summaryPagesURL string
	summaryOutput   string
)

var summaryCmd = &cobra.Command{
	Use:   "summary <commit-range>",
	Short: "Generate summary for commits",
	Long: `Generate a summary of LLM sessions for commits in a range.

This command is designed for CI/CD pipelines to create PR comments or reports.
Output formats: markdown (default) or json.

Examples:
  git-prompt-story summary HEAD~5..HEAD
  git-prompt-story summary abc123..def456 --format=json
  git-prompt-story summary main..feature-branch --pages-url=https://example.github.io/repo/pr-42/`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		commitRange := args[0]

		summary, err := ci.GenerateSummary(commitRange, summaryFull)
		if err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
			os.Exit(1)
		}

		var output string
		switch summaryFormat {
		case "markdown", "md":
			output = ci.RenderMarkdown(summary, summaryPagesURL, GetVersion())
		case "json":
			jsonBytes, err := ci.RenderJSON(summary)
			if err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: failed to render JSON: %v\n", err)
				os.Exit(1)
			}
			output = string(jsonBytes)
		default:
			fmt.Fprintf(os.Stderr, "git-prompt-story: unknown format: %s\n", summaryFormat)
			os.Exit(1)
		}

		if summaryOutput != "" {
			if err := os.WriteFile(summaryOutput, []byte(output), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: failed to write output: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Print(output)
		}
	},
}

func init() {
	summaryCmd.Flags().StringVar(&summaryFormat, "format", "markdown", "Output format: markdown or json")
	summaryCmd.Flags().BoolVar(&summaryFull, "full", false, "Include full prompt text (not truncated)")
	summaryCmd.Flags().StringVar(&summaryPagesURL, "pages-url", "", "URL to GitHub Pages transcripts (included in markdown output)")
	summaryCmd.Flags().StringVar(&summaryOutput, "output", "", "Write output to file instead of stdout")
	rootCmd.AddCommand(summaryCmd)
}
