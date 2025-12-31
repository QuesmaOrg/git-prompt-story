package cmd

import (
	"fmt"
	"os"

	"github.com/QuesmaOrg/git-prompt-story/internal/ci"
	"github.com/spf13/cobra"
)

var (
	ciHTMLOutputDir string
	ciHTMLPRNumber  int
)

var ciHTMLCmd = &cobra.Command{
	Use:   "ci-html <commit-range>",
	Short: "Generate HTML transcript pages",
	Long: `Generate static HTML pages showing full transcripts for commits in a range.

This command creates an index.html and individual commit pages suitable for
deployment to GitHub Pages.

Examples:
  git-prompt-story ci-html HEAD~5..HEAD --output-dir=./pages
  git-prompt-story ci-html main..feature --output-dir=./pr-42 --pr=42`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		commitRange := args[0]

		if ciHTMLOutputDir == "" {
			fmt.Fprintf(os.Stderr, "git-prompt-story: --output-dir is required\n")
			os.Exit(1)
		}

		// Generate with full prompts for HTML
		summary, err := ci.GenerateSummary(commitRange, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
			os.Exit(1)
		}

		if err := ci.GenerateHTML(summary, ciHTMLOutputDir, ciHTMLPRNumber); err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Generated HTML pages in %s\n", ciHTMLOutputDir)
		fmt.Printf("  - index.html\n")
		for _, commit := range summary.Commits {
			fmt.Printf("  - %s.html\n", commit.ShortSHA)
		}
	},
}

func init() {
	ciHTMLCmd.Flags().StringVar(&ciHTMLOutputDir, "output-dir", "", "Directory to write HTML files (required)")
	ciHTMLCmd.Flags().IntVar(&ciHTMLPRNumber, "pr", 0, "PR number for page title")
	rootCmd.AddCommand(ciHTMLCmd)
}
