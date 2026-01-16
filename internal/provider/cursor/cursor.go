package cursor

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/QuesmaOrg/git-prompt-story/internal/provider"
)

func init() {
	provider.Register(&Provider{})
}

// Provider implements the provider.Provider interface for Cursor
type Provider struct{}

// Name returns the tool identifier
func (p *Provider) Name() string {
	return "cursor"
}

// FileExtension returns the file extension for Cursor transcripts
func (p *Provider) FileExtension() string {
	return ".json"
}

// ComposerData represents the structure of Cursor's composerData entries
type ComposerData struct {
	ComposerID         string            `json:"composerId"`
	CreatedAt          int64             `json:"createdAt"`          // epoch ms
	LastUpdatedAt      int64             `json:"lastUpdatedAt"`      // epoch ms
	Conversation       []Bubble          `json:"conversation"`       // array of bubbles (old format)
	OriginalFileStates map[string]interface{} `json:"originalFileStates"` // file:///path -> state (new format)
	RawJSON            []byte            `json:"-"`                  // original JSON for storage
}

// Bubble represents a single message in a Cursor conversation
type Bubble struct {
	Type       int        `json:"type"`       // 1=user, 2=AI
	BubbleID   string     `json:"bubbleId"`
	Text       string     `json:"text"`
	TimingInfo TimingInfo `json:"timingInfo,omitempty"`
	CodeBlocks []CodeBlock `json:"codeBlocks,omitempty"`
	Checkpoint *Checkpoint `json:"checkpoint,omitempty"`
}

// TimingInfo contains timing information for a bubble
type TimingInfo struct {
	ClientStartTime int64 `json:"clientStartTime"` // epoch ms
}

// CodeBlock represents a code block in an AI response
type CodeBlock struct {
	URI struct {
		FSPath string `json:"fsPath"`
	} `json:"uri"`
}

// Checkpoint represents a checkpoint with file information
type Checkpoint struct {
	Files []struct {
		URI struct {
			FSPath string `json:"fsPath"`
		} `json:"uri"`
	} `json:"files"`
}

// GetDBPath returns the platform-specific path to Cursor's state.vscdb
func GetDBPath() string {
	homeDir, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "Cursor", "User", "globalStorage", "state.vscdb")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(homeDir, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Cursor", "User", "globalStorage", "state.vscdb")
	default: // linux
		return filepath.Join(homeDir, ".config", "Cursor", "User", "globalStorage", "state.vscdb")
	}
}

// DiscoverSessions finds Cursor sessions for a repo path
func (p *Provider) DiscoverSessions(repoPath string, startWork, endWork time.Time) ([]provider.RawSession, error) {
	dbPath := GetDBPath()

	// Check if DB exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, nil // Cursor not installed or no data
	}

	// Open database in read-only mode
	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Query for composerData entries
	rows, err := db.Query(`SELECT key, value FROM cursorDiskKV WHERE key LIKE 'composerData:%'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	absRepoPath, _ := filepath.Abs(repoPath)
	var sessions []provider.RawSession

	for rows.Next() {
		var key string
		var value []byte
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}

		// Parse the JSON
		var data ComposerData
		if err := json.Unmarshal(value, &data); err != nil {
			continue
		}
		data.RawJSON = value

		// Extract timestamps
		created := time.UnixMilli(data.CreatedAt)
		modified := getLastTimestamp(&data)
		if modified.IsZero() {
			modified = created
		}

		// Time filter: session must overlap with work period
		if modified.Before(startWork) || created.After(endWork) {
			continue
		}

		// Extract workspace path from file paths in conversation
		workspacePath := extractWorkspacePath(&data)
		if workspacePath == "" {
			continue // Can't determine workspace
		}

		// Check if workspace matches our repo
		if !pathMatches(workspacePath, absRepoPath) {
			continue
		}

		sessions = append(sessions, provider.RawSession{
			ID:       data.ComposerID,
			Tool:     "cursor",
			Path:     key, // Store the key for later retrieval
			Created:  created,
			Modified: modified,
			RepoPath: workspacePath,
		})
	}

	return sessions, nil
}

// ReadTranscript reads the raw JSON content for storage
// This combines composerData with all associated bubbles
func (p *Provider) ReadTranscript(sess provider.RawSession) ([]byte, error) {
	dbPath := GetDBPath()

	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Read main composer data
	var composerValue []byte
	err = db.QueryRow(`SELECT value FROM cursorDiskKV WHERE key = ?`, sess.Path).Scan(&composerValue)
	if err != nil {
		return nil, err
	}

	// Parse composer data
	var data map[string]interface{}
	if err := json.Unmarshal(composerValue, &data); err != nil {
		return composerValue, nil // Return as-is if we can't parse
	}

	// Fetch associated bubbles (stored separately in new Cursor format)
	bubbleRows, err := db.Query(`SELECT value FROM cursorDiskKV WHERE key LIKE ?`, "bubbleId:"+sess.ID+":%")
	if err == nil {
		defer bubbleRows.Close()
		var bubbles []map[string]interface{}
		for bubbleRows.Next() {
			var bubbleValue []byte
			if bubbleRows.Scan(&bubbleValue) == nil {
				var bubble map[string]interface{}
				if json.Unmarshal(bubbleValue, &bubble) == nil {
					bubbles = append(bubbles, bubble)
				}
			}
		}
		if len(bubbles) > 0 {
			// Add bubbles to data for storage
			data["_bubbles"] = bubbles
		}
	}

	// Re-encode with bubbles included
	return json.Marshal(data)
}

// getLastTimestamp finds the most recent timestamp in the conversation
func getLastTimestamp(data *ComposerData) time.Time {
	var latest time.Time

	// Check lastUpdatedAt field (new format)
	if data.LastUpdatedAt > 0 {
		latest = time.UnixMilli(data.LastUpdatedAt)
	}

	// Also check conversation bubbles (old format)
	for _, bubble := range data.Conversation {
		if bubble.TimingInfo.ClientStartTime > 0 {
			ts := time.UnixMilli(bubble.TimingInfo.ClientStartTime)
			if ts.After(latest) {
				latest = ts
			}
		}
	}
	return latest
}

// extractWorkspacePath derives the workspace path from file paths in the conversation
func extractWorkspacePath(data *ComposerData) string {
	var paths []string

	// New format: extract from originalFileStates (keys are file:///path URIs)
	for uri := range data.OriginalFileStates {
		if strings.HasPrefix(uri, "file://") {
			// Convert file:///Users/... to /Users/...
			path := strings.TrimPrefix(uri, "file://")
			if path != "" {
				paths = append(paths, path)
			}
		}
	}

	// Old format: extract from conversation bubbles
	for _, bubble := range data.Conversation {
		// Collect file paths from codeBlocks
		for _, cb := range bubble.CodeBlocks {
			if cb.URI.FSPath != "" {
				paths = append(paths, cb.URI.FSPath)
			}
		}

		// Collect file paths from checkpoints
		if bubble.Checkpoint != nil {
			for _, f := range bubble.Checkpoint.Files {
				if f.URI.FSPath != "" {
					paths = append(paths, f.URI.FSPath)
				}
			}
		}
	}

	if len(paths) == 0 {
		return ""
	}

	// Find common ancestor directory
	common := findCommonAncestor(paths)
	if common == "" {
		return ""
	}

	// Walk up to find .git directory (repo root)
	return findGitRoot(common)
}

// findCommonAncestor finds the longest common directory prefix of all paths
func findCommonAncestor(paths []string) string {
	if len(paths) == 0 {
		return ""
	}

	// Start with the directory of the first path
	common := filepath.Dir(paths[0])

	for _, p := range paths[1:] {
		dir := filepath.Dir(p)
		for !strings.HasPrefix(dir, common) && common != "/" && common != "" {
			common = filepath.Dir(common)
		}
	}

	return common
}

// findGitRoot walks up from a path to find the git repository root
func findGitRoot(path string) string {
	current := path
	for current != "/" && current != "" {
		gitDir := filepath.Join(current, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return current
		}
		// Also check for .git file (worktrees)
		if info, err := os.Stat(gitDir); err == nil && !info.IsDir() {
			return current
		}
		current = filepath.Dir(current)
	}
	return ""
}

// pathMatches checks if workspace path matches the repo path
func pathMatches(workspace, repo string) bool {
	workspace = filepath.Clean(workspace)
	repo = filepath.Clean(repo)

	// Exact match
	if workspace == repo {
		return true
	}

	// Workspace is under repo (subfolder)
	if strings.HasPrefix(workspace, repo+string(filepath.Separator)) {
		return true
	}

	// Repo is under workspace (parent folder case)
	if strings.HasPrefix(repo, workspace+string(filepath.Separator)) {
		return true
	}

	return false
}
