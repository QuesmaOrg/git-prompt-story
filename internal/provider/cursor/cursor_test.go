package cursor

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiscoverSessions(t *testing.T) {
	// Skip if Cursor DB doesn't exist
	dbPath := GetDBPath()
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skip("Cursor database not found, skipping test")
	}

	p := &Provider{}
	// Use current directory as repo path (this test file is in a git repo)
	repoPath, _ := os.Getwd()
	// Go up to the repo root (git-prompt-story)
	for i := 0; i < 4; i++ {
		repoPath = filepath.Dir(repoPath)
	}

	startWork := time.Now().Add(-365 * 24 * time.Hour) // 1 year ago
	endWork := time.Now()

	sessions, err := p.DiscoverSessions(repoPath, startWork, endWork)
	if err != nil {
		t.Errorf("DiscoverSessions failed: %v", err)
	}

	// This test just verifies the function runs without error
	// It may or may not find sessions depending on the machine
	t.Logf("Found %d Cursor sessions for %s", len(sessions), repoPath)
}

func TestExtractWorkspacePath(t *testing.T) {
	data := &ComposerData{
		Conversation: []Bubble{
			{
				CodeBlocks: []CodeBlock{
					{URI: struct{ FSPath string `json:"fsPath"` }{FSPath: "/Users/test/project/src/main.go"}},
					{URI: struct{ FSPath string `json:"fsPath"` }{FSPath: "/Users/test/project/src/util.go"}},
				},
			},
		},
	}

	workspace := extractWorkspacePath(data)
	t.Logf("Extracted workspace: %s", workspace)
	// Note: This will return empty if /Users/test/project doesn't have a .git directory
}
