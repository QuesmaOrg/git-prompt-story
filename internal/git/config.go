package git

import (
	"os/exec"
	"regexp"
	"strings"
)

const defaultViewerURL = "https://prompt-story.quesma.com/{owner}/{repo}/prompt/{note}"

// GetViewerURL retrieves the configured viewer URL template
func GetViewerURL() string {
	cmd := exec.Command("git", "config", "--get", "prompt-story.viewer-url")
	out, err := cmd.Output()
	if err != nil {
		return defaultViewerURL
	}
	url := strings.TrimSpace(string(out))
	if url == "" {
		return defaultViewerURL
	}
	return url
}

// GetRemoteURL retrieves the origin remote URL
func GetRemoteURL() (string, error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ExtractOwnerRepo parses owner and repo from remote URL
// Supports: git@github.com:owner/repo.git and https://github.com/owner/repo.git
func ExtractOwnerRepo(remoteURL string) (owner, repo string, err error) {
	// SSH format: git@github.com:owner/repo.git
	sshRegex := regexp.MustCompile(`git@[^:]+:([^/]+)/([^/]+?)(?:\.git)?$`)
	if matches := sshRegex.FindStringSubmatch(remoteURL); matches != nil {
		return matches[1], matches[2], nil
	}

	// HTTPS format: https://github.com/owner/repo.git
	httpsRegex := regexp.MustCompile(`https?://[^/]+/([^/]+)/([^/]+?)(?:\.git)?$`)
	if matches := httpsRegex.FindStringSubmatch(remoteURL); matches != nil {
		return matches[1], matches[2], nil
	}

	return "", "", nil
}
