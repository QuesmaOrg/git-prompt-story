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

	// Check if we have any notes to push
	hasNotes := hasNotesRef(note.NotesRef)
	hasTranscripts := hasNotesRef(note.TranscriptsRef)

	if !hasNotes && !hasTranscripts {
		// No notes refs exist, nothing to push
		return nil
	}

	var errors []string

	// Push main notes ref (non-force, fast-forward only)
	if hasNotes {
		if err := pushNotesRef(remoteName, note.NotesRef, false); err != nil {
			errors = append(errors, fmt.Sprintf("prompt-story notes: %v", err))
		} else {
			fmt.Printf("git-prompt-story: pushed %s to %s\n", note.NotesRef, remoteName)
		}
	}

	// Push transcripts ref (force push - transcripts can be amended)
	if hasTranscripts {
		if err := pushNotesRef(remoteName, note.TranscriptsRef, true); err != nil {
			errors = append(errors, fmt.Sprintf("transcripts: %v", err))
		} else {
			fmt.Printf("git-prompt-story: pushed %s to %s\n", note.TranscriptsRef, remoteName)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to push some notes: %s", strings.Join(errors, "; "))
	}

	return nil
}

// hasNotesRef checks if a notes ref exists locally
func hasNotesRef(ref string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", ref)
	return cmd.Run() == nil
}

// pushNotesRef pushes a notes ref to the remote
func pushNotesRef(remote, ref string, force bool) error {
	refspec := ref + ":" + ref
	if force {
		refspec = "+" + refspec
	}

	cmd := exec.Command("git", "push", remote, refspec)
	// Set environment variable to prevent recursive hook invocation
	cmd.Env = append(os.Environ(), prePushEnvVar+"=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if the error is just "already up to date" or "no updates"
		outputStr := string(output)
		if strings.Contains(outputStr, "Everything up-to-date") ||
			strings.Contains(outputStr, "up to date") {
			return nil
		}
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(outputStr))
	}
	return nil
}
