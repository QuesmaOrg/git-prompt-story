package hooks

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

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
		// We don't need the ref info, just consume it
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	// Build refspecs for existing note refs
	var refspecs []string
	if hasNotesRef(note.NotesRef) {
		refspecs = append(refspecs, note.NotesRef+":"+note.NotesRef)
	}
	if hasNotesRef(note.TranscriptsRef) {
		// Force push for transcripts (they can be amended)
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
	outputStr := string(output)

	// Check if nothing was pushed (already up to date)
	if strings.Contains(outputStr, "Everything up-to-date") ||
		strings.Contains(outputStr, "up to date") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("pushing notes: %s", strings.TrimSpace(outputStr))
	}

	// Only print if something was actually pushed
	fmt.Printf("git-prompt-story: pushed notes to %s\n", remoteName)
	return nil
}

// hasNotesRef checks if a notes ref exists locally
func hasNotesRef(ref string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", ref)
	return cmd.Run() == nil
}
