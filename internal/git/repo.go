package git

import (
	"os"
	"os/exec"
	"strings"
	"time"
)

// GetRepoRoot returns the root directory of the git repo
func GetRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GetGitDir returns the .git directory path
func GetGitDir() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// IsInsideWorkTree checks if we're in a git repository
func IsInsideWorkTree() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

// GetHead returns the SHA of HEAD
func GetHead() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GetCommitTime returns the effective commit timestamp
// Checks GIT_COMMITTER_DATE and GIT_AUTHOR_DATE env vars first (set by faketime or user)
// Falls back to current time if not set
func GetCommitTime() time.Time {
	// Try GIT_COMMITTER_DATE first (takes precedence)
	if dateStr := os.Getenv("GIT_COMMITTER_DATE"); dateStr != "" {
		if t, err := parseGitDate(dateStr); err == nil {
			return t
		}
	}

	// Try GIT_AUTHOR_DATE
	if dateStr := os.Getenv("GIT_AUTHOR_DATE"); dateStr != "" {
		if t, err := parseGitDate(dateStr); err == nil {
			return t
		}
	}

	// Fall back to current time
	return time.Now().UTC()
}

// parseGitDate parses common git date formats
func parseGitDate(dateStr string) (time.Time, error) {
	// Try RFC3339 format first
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return t.UTC(), nil
	}

	// Try git's default format: "Mon Jan 2 15:04:05 2006 -0700"
	if t, err := time.Parse("Mon Jan 2 15:04:05 2006 -0700", dateStr); err == nil {
		return t.UTC(), nil
	}

	// Try ISO format: "2006-01-02 15:04:05 -0700"
	if t, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr); err == nil {
		return t.UTC(), nil
	}

	// Try Unix timestamp
	if t, err := time.Parse("@1136239445", dateStr); err == nil {
		return t.UTC(), nil
	}

	return time.Time{}, nil
}
