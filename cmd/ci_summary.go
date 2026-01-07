package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/ci"
	"github.com/spf13/cobra"
)

var (
	ciSummaryFormat   string
	ciSummaryFull     bool
	ciSummaryPagesURL string
	ciSummaryOutput   string
)

// Backward-compatible alias for "summary"
var ciSummaryCmd = &cobra.Command{
	Use:        "ci-summary <commit-range>",
	Short:      "Generate CI summary for commits",
	Hidden:     true, // Hidden, use "summary" instead
	Deprecated: "use 'git-prompt-story summary' instead",
	Args:       cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		commitRange := args[0]

		summary, err := ci.GenerateSummary(commitRange, ciSummaryFull)
		if err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
			os.Exit(1)
		}

		var output string
		switch ciSummaryFormat {
		case "markdown", "md":
			output = ci.RenderMarkdown(summary, ciSummaryPagesURL, GetVersion())
		case "json":
			jsonBytes, err := ci.RenderJSON(summary)
			if err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: failed to render JSON: %v\n", err)
				os.Exit(1)
			}
			output = string(jsonBytes)
		default:
			fmt.Fprintf(os.Stderr, "git-prompt-story: unknown format: %s\n", ciSummaryFormat)
			os.Exit(1)
		}

		if ciSummaryOutput != "" {
			if err := os.WriteFile(ciSummaryOutput, []byte(output), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: failed to write output: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Print(output)
		}
	},
}

func init() {
	ciSummaryCmd.Flags().StringVar(&ciSummaryFormat, "format", "markdown", "Output format: markdown or json")
	ciSummaryCmd.Flags().BoolVar(&ciSummaryFull, "full", false, "Include full prompt text (not truncated)")
	ciSummaryCmd.Flags().StringVar(&ciSummaryPagesURL, "pages-url", "", "URL to GitHub Pages transcripts (included in markdown output)")
	ciSummaryCmd.Flags().StringVar(&ciSummaryOutput, "output", "", "Write output to file instead of stdout")
	rootCmd.AddCommand(ciSummaryCmd)
}
