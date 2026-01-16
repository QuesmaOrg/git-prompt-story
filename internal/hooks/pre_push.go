package hooks

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/note"
)

// Environment variable to prevent recursive hook invocation
const prePushEnvVar = "GIT_PROMPT_STORY_PUSHING_NOTES"

// PrePush implements the pre-push hook logic.
// It pushes prompt-story notes to the same remote being pushed to.
//
// Git pre-push hook receives:
//   - args: remote name (e.g., "origin") and URL
//   - stdin: lines of "<local-ref> <local-sha> <remote-ref> <remote-sha>"
//
// We read stdin (to consume it) but don't need its content - we just
// push all notes refs whenever a push happens.
func PrePush(remoteName, remoteURL string, stdin io.Reader) error {
	// Prevent recursive invocation when we push notes from within this hook
	if os.Getenv(prePushEnvVar) == "1" {
		// Consume stdin and return early - this is our own notes push
		io.Copy(io.Discard, stdin)
		return nil
	}

	// Consume stdin (required by git)
	scanner := bufio.NewScanner(stdin)
	for scanner.Scan() {
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	// Build refspecs for existing note refs
	// Force push both refs (+prefix) because notes can diverge when:
	// - Commits are amended/rebased (old SHA keeps orphaned note)
	// - Multiple machines work on same repo without syncing notes
	// Force push is safe since notes are metadata - losing an orphaned
	// note for a non-existent commit has no impact.
	var refspecs []string
	if hasNotesRef(note.NotesRef) {
		refspecs = append(refspecs, "+"+note.NotesRef+":"+note.NotesRef)
	}
	if hasNotesRef(note.TranscriptsRef) {
		refspecs = append(refspecs, "+"+note.TranscriptsRef+":"+note.TranscriptsRef)
	}

	if len(refspecs) == 0 {
		// No notes refs exist, nothing to push
		return nil
	}

	// Single push with all refspecs
	args := append([]string{"push", remoteName}, refspecs...)
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), prePushEnvVar+"=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		if strings.Contains(outputStr, "Everything up-to-date") ||
			strings.Contains(outputStr, "up to date") {
			return nil
		}
		return fmt.Errorf("pushing notes: %s", strings.TrimSpace(outputStr))
	}

	fmt.Printf("git-prompt-story: pushed notes to %s\n", remoteName)

	// Show GitHub workflow nudge if applicable
	maybeShowGitHubWorkflowNudge(remoteURL)

	return nil
}

// maybeShowGitHubWorkflowNudge shows a tip to install GitHub workflow if pushing to GitHub
// without the workflow file present
func maybeShowGitHubWorkflowNudge(remoteURL string) {
	if !isGitHubRemote(remoteURL) || nudgesDisabled() || hasPromptStoryWorkflow() {
		return
	}

	fmt.Println("💡 Show in PRs → git-prompt-story install-github-workflow")
	fmt.Println("   Snooze: touch .git/prompt-story-no-nudge")
}

// isGitHubRemote checks if the remote URL points to GitHub
func isGitHubRemote(remoteURL string) bool {
	return strings.Contains(remoteURL, "github.com")
}

// hasPromptStoryWorkflow checks if the GitHub workflow file exists
func hasPromptStoryWorkflow() bool {
	_, err := os.Stat(".github/workflows/prompt-story.yml")
	return err == nil
}

// nudgesDisabled checks if nudges have been disabled via marker file
func nudgesDisabled() bool {
	gitDir, err := git.GetGitDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(gitDir, "prompt-story-no-nudge"))
	return err == nil
}

// hasNotesRef checks if a notes ref exists locally
func hasNotesRef(ref string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", ref)
	return cmd.Run() == nil
}
