package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/show"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	fullFlag          bool
	interactiveFlag   bool
	noInteractiveFlag bool
	clearSessionFlag  string
	redactMessageFlag string
)

var showCmd = &cobra.Command{
	Use:   "show [commit]",
	Short: "Show prompts for a commit",
	Long: `Display LLM prompts and sessions attached to a commit or commit range.

By default, opens an interactive TUI viewer when running in a terminal.
Use --no-interactive for plain text output (useful for piping).
Use --full to display complete message content.

Examples:
  git-prompt-story show                # Show prompts for HEAD
  git-prompt-story show abc123         # Show prompts for specific commit
  git-prompt-story show HEAD~5..HEAD   # Show prompts for commit range`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Handle redaction flags (non-interactive operations)
		if clearSessionFlag != "" {
			if err := handleClearSession(clearSessionFlag); err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
				os.Exit(1)
			}
			return
		}
		if redactMessageFlag != "" {
			if err := handleRedactMessage(redactMessageFlag); err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
				os.Exit(1)
			}
			return
		}

		commit := "HEAD"
		if len(args) > 0 {
			commit = args[0]
		}

		// Determine if we should use interactive mode
		isTTY := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
		useInteractive := (interactiveFlag || isTTY) && !noInteractiveFlag

		if useInteractive {
			if err := show.RunTUI(commit, fullFlag); err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
				os.Exit(1)
			}
		} else {
			if err := show.ShowPrompts(commit, fullFlag); err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

// handleClearSession parses "tool/session-id" and clears the session
func handleClearSession(spec string) error {
	parts := strings.SplitN(spec, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid session spec: %s (expected tool/session-id)", spec)
	}
	tool, sessionID := parts[0], parts[1]

	if err := show.DeleteSession(tool, sessionID); err != nil {
		return err
	}

	if show.WasNotesPushed() {
		fmt.Println("Session cleared. Force push needed: git push -f origin refs/notes/*")
	} else {
		fmt.Println("Session cleared")
	}
	return nil
}

// handleRedactMessage parses "tool/session-id@timestamp" and redacts the message
func handleRedactMessage(spec string) error {
	// Split by @ to get session spec and timestamp
	atIdx := strings.LastIndex(spec, "@")
	if atIdx == -1 {
		return fmt.Errorf("invalid redact spec: %s (expected tool/session-id@timestamp)", spec)
	}
	sessionSpec := spec[:atIdx]
	timestampStr := spec[atIdx+1:]

	// Parse session spec
	parts := strings.SplitN(sessionSpec, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid session spec: %s (expected tool/session-id)", sessionSpec)
	}
	tool, sessionID := parts[0], parts[1]

	// Parse timestamp
	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %s (expected RFC3339 format)", timestampStr)
	}

	if err := show.RedactMessage(tool, sessionID, timestamp); err != nil {
		return err
	}

	if show.WasNotesPushed() {
		fmt.Println("Message redacted. Force push needed: git push -f origin refs/notes/*")
	} else {
		fmt.Println("Message redacted")
	}
	return nil
}

func init() {
	showCmd.Flags().BoolVar(&fullFlag, "full", false, "Show full message content")
	showCmd.Flags().BoolVarP(&interactiveFlag, "interactive", "i", false, "Force interactive TUI mode")
	showCmd.Flags().BoolVar(&noInteractiveFlag, "no-interactive", false, "Disable interactive TUI, use plain text output")
	showCmd.Flags().StringVar(&clearSessionFlag, "clear-session", "", "Clear session content (format: tool/session-id)")
	showCmd.Flags().StringVar(&redactMessageFlag, "redact-message", "", "Redact message (format: tool/session-id@timestamp)")
	rootCmd.AddCommand(showCmd)
}
