package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// UpdateRef creates or updates a ref to point to an object
func UpdateRef(ref, sha string) error {
	cmd := exec.Command("git", "update-ref", ref, sha)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git update-ref %s %s: %w", ref, sha, err)
	}
	return nil
}

// GetRef returns the SHA a ref points to, or empty if not exists
func GetRef(ref string) (string, error) {
	cmd := exec.Command("git", "show-ref", "--hash", ref)
	out, err := cmd.Output()
	if err != nil {
		// Ref doesn't exist
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}

// GetRemoteRef returns the SHA of a ref on the remote, or empty if not exists
func GetRemoteRef(remote, ref string) (string, error) {
	cmd := exec.Command("git", "ls-remote", remote, ref)
	out, err := cmd.Output()
	if err != nil {
		return "", nil
	}
	output := strings.TrimSpace(string(out))
	if output == "" {
		return "", nil
	}
	// Format: sha TAB ref
	parts := strings.Fields(output)
	if len(parts) >= 1 {
		return parts[0], nil
	}
	return "", nil
}

// RefExistsOnRemote checks if a ref exists on the remote
func RefExistsOnRemote(remote, ref string) bool {
	sha, _ := GetRemoteRef(remote, ref)
	return sha != ""
}

// ForcePushRef force-pushes a ref to a remote
func ForcePushRef(remote, ref string) error {
	cmd := exec.Command("git", "push", "-f", remote, ref)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git push -f %s %s: %w", remote, ref, err)
	}
	return nil
}

// PushRefs pushes multiple refs to a remote
// refspecs can include force markers (e.g., "+refs/notes/foo")
func PushRefs(remote string, refspecs ...string) error {
	args := append([]string{"push", remote}, refspecs...)
	cmd := exec.Command("git", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git push %s: %w", remote, err)
	}
	return nil
}

