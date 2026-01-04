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

