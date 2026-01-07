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

// Backward-compatible alias for "pages"
var ciHTMLCmd = &cobra.Command{
	Use:        "ci-html <commit-range>",
	Short:      "Generate HTML transcript pages",
	Hidden:     true, // Hidden, use "pages" instead
	Deprecated: "use 'git-prompt-story pages' instead",
	Args:       cobra.ExactArgs(1),
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
