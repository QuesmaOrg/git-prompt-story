package cmd

import (
	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Commands for PR integration and analysis",
	Long: `Commands for generating PR comments, summaries, and HTML transcripts.

These commands are designed for CI/CD pipelines to create PR comments,
generate summary reports, and deploy transcript pages.

Available subcommands:
  analyze   Analyze PR and determine if comment should be posted
  summary   Generate markdown or JSON summary of commits
  html      Generate static HTML transcript pages
  preview   Preview summary as rendered GitHub markdown`,
}

func init() {
	rootCmd.AddCommand(prCmd)
}
