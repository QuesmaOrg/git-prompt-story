package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// HashObject creates a blob object from content, returns SHA
func HashObject(content []byte) (string, error) {
	cmd := exec.Command("git", "hash-object", "-w", "--stdin")
	cmd.Stdin = bytes.NewReader(content)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git hash-object: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// TreeEntry represents an entry in a git tree
type TreeEntry struct {
	Mode string // "100644" for files, "040000" for directories
	Type string // "blob" or "tree"
	SHA  string
	Name string
}

// CreateTree creates a tree object from entries
func CreateTree(entries []TreeEntry) (string, error) {
	var buf bytes.Buffer
	for _, e := range entries {
		// Format: mode SP type SP sha TAB name NUL
		fmt.Fprintf(&buf, "%s %s %s\t%s\n", e.Mode, e.Type, e.SHA, e.Name)
	}

	cmd := exec.Command("git", "mktree")
	cmd.Stdin = &buf
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git mktree: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ReadTree reads an existing tree and returns its entries
func ReadTree(treeSHA string) ([]TreeEntry, error) {
	cmd := exec.Command("git", "ls-tree", treeSHA)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-tree: %w", err)
	}

	var entries []TreeEntry
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Format: mode SP type SP sha TAB name
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		fields := strings.Fields(parts[0])
		if len(fields) != 3 {
			continue
		}
		entries = append(entries, TreeEntry{
			Mode: fields[0],
			Type: fields[1],
			SHA:  fields[2],
			Name: parts[1],
		})
	}
	return entries, nil
}

// GetBlobContent retrieves content from a ref:path specification
// Example: GetBlobContent("refs/notes/prompt-story-transcripts", "claude-code/session-id.jsonl")
func GetBlobContent(ref, path string) ([]byte, error) {
	spec := ref + ":" + path
	cmd := exec.Command("git", "cat-file", "-p", spec)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git cat-file %s: %w", spec, err)
	}
	return out, nil
}

// ResolveCommit resolves a commit reference (HEAD, hash, etc.) to full SHA
func ResolveCommit(ref string) (string, error) {
	cmd := exec.Command("git", "rev-parse", ref)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse %s: %w", ref, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// RevList returns commits in a range (e.g., "HEAD~3..HEAD")
// Returns commits in reverse chronological order (newest first)
func RevList(rangeSpec string) ([]string, error) {
	cmd := exec.Command("git", "rev-list", rangeSpec)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git rev-list %s: %w", rangeSpec, err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var commits []string
	for _, line := range lines {
		if line != "" {
			commits = append(commits, line)
		}
	}
	return commits, nil
}

