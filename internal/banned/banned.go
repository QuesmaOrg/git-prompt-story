package banned

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
)

// BannedSession represents a session that has been banned from future captures
type BannedSession struct {
	ID       string    `json:"id"`
	Tool     string    `json:"tool"`
	BannedAt time.Time `json:"banned_at"`
	Reason   string    `json:"reason,omitempty"`
}

// BannedList is the structure stored in .git/prompt-story/banned.json
type BannedList struct {
	Banned []BannedSession `json:"banned"`
}

// getBannedFilePath returns the path to the banned sessions file
func getBannedFilePath() (string, error) {
	gitDir, err := git.GetGitDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(gitDir, "prompt-story", "banned.json"), nil
}

// Load reads the banned sessions list from disk
func Load() (*BannedList, error) {
	path, err := getBannedFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &BannedList{Banned: []BannedSession{}}, nil
		}
		return nil, err
	}

	var list BannedList
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return &list, nil
}

// Save writes the banned sessions list to disk
func Save(list *BannedList) error {
	path, err := getBannedFilePath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// IsBanned checks if a session ID is in the banned list
func IsBanned(sessionID string) bool {
	list, err := Load()
	if err != nil {
		return false
	}
	for _, s := range list.Banned {
		if s.ID == sessionID {
			return true
		}
	}
	return false
}

// Ban adds a session to the banned list
func Ban(id, tool, reason string) error {
	list, err := Load()
	if err != nil {
		return err
	}

	// Check if already banned
	for _, s := range list.Banned {
		if s.ID == id {
			return nil // Already banned
		}
	}

	list.Banned = append(list.Banned, BannedSession{
		ID:       id,
		Tool:     tool,
		BannedAt: time.Now(),
		Reason:   reason,
	})

	return Save(list)
}

// Unban removes a session from the banned list
func Unban(id string) error {
	list, err := Load()
	if err != nil {
		return err
	}

	filtered := make([]BannedSession, 0, len(list.Banned))
	for _, s := range list.Banned {
		if s.ID != id {
			filtered = append(filtered, s)
		}
	}
	list.Banned = filtered

	return Save(list)
}
