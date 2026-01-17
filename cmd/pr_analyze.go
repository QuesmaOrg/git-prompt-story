package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/ci"
	"github.com/spf13/cobra"
)

var (
	prAnalyzePagesURL   string
	prAnalyzeOutputJSON string
	prAnalyzeOutputMD   string
	prAnalyzeSetOutput  bool
)

var prAnalyzeCmd = &cobra.Command{
	Use:   "analyze <commit-range>",
	Short: "Analyze PR for GitHub Actions",
	Long: `Analyze commits in a PR and output results for GitHub Actions.

This command encapsulates all PR analysis logic, making the GitHub Action
a thin wrapper. It determines whether to post a comment and generates
the appropriate content.

The key insight: only post a comment if we have actual notes. This avoids
false "Notes not found" warnings that were triggered by fragile marker detection.

Examples:
  git-prompt-story pr analyze origin/main..HEAD
  git-prompt-story pr analyze origin/main..HEAD --output-json=results.json
  git-prompt-story pr analyze origin/main..HEAD --output-json=results.json --output-markdown=summary.md
  git-prompt-story pr analyze origin/main..HEAD --set-output`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		commitRange := args[0]

		result, err := ci.AnalyzePR(commitRange, prAnalyzePagesURL, GetVersion())
		if err != nil {
			// Still output the result with error info
			if prAnalyzeOutputJSON != "" {
				prOutputJSON(result, prAnalyzeOutputJSON)
			}
			fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
			os.Exit(1)
		}

		// Output JSON if requested
		if prAnalyzeOutputJSON != "" {
			prOutputJSON(result, prAnalyzeOutputJSON)
		}

		// Output markdown if requested and we have content
		if prAnalyzeOutputMD != "" && result.MarkdownBody != "" {
			if err := os.WriteFile(prAnalyzeOutputMD, []byte(result.MarkdownBody), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: failed to write markdown: %v\n", err)
				os.Exit(1)
			}
		}

		// Output GitHub Actions format if requested
		if prAnalyzeSetOutput {
			fmt.Printf("commits-analyzed=%d\n", result.CommitsAnalyzed)
			fmt.Printf("commits-with-notes=%d\n", result.CommitsWithNotes)
			fmt.Printf("should-post-comment=%t\n", result.ShouldPostComment)
			fmt.Printf("comment-type=%s\n", result.CommentType)
		}

		// If no output options specified, print JSON to stdout
		if prAnalyzeOutputJSON == "" && !prAnalyzeSetOutput {
			jsonBytes, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(jsonBytes))
		}
	},
}

func prOutputJSON(result *ci.AnalysisResult, path string) {
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "git-prompt-story: failed to marshal JSON: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(path, jsonBytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "git-prompt-story: failed to write JSON: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	prAnalyzeCmd.Flags().StringVar(&prAnalyzePagesURL, "pages-url", "", "URL to GitHub Pages transcripts")
	prAnalyzeCmd.Flags().StringVar(&prAnalyzeOutputJSON, "output-json", "", "Write JSON results to file")
	prAnalyzeCmd.Flags().StringVar(&prAnalyzeOutputMD, "output-markdown", "", "Write markdown summary to file")
	prAnalyzeCmd.Flags().BoolVar(&prAnalyzeSetOutput, "set-output", false, "Output in GitHub Actions format (key=value lines)")
	prCmd.AddCommand(prAnalyzeCmd)
}
