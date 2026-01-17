package cmd

import (
	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "PR integration commands",
}

func init() {
	rootCmd.AddCommand(prCmd)
}
