package cloud

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

const (
	baseURL         = "https://api.anthropic.com"
	anthropicVersion = "2023-06-01"
	keychainService = "Claude Code-credentials"
)

// Client is the Claude Code Cloud API client
type Client struct {
	token   string
	orgUUID string
	http    *http.Client
}

// claudeConfig represents the ~/.claude.json file structure
type claudeConfig struct {
	OAuthAccount struct {
		OrganizationUUID string `json:"organizationUuid"`
	} `json:"oauthAccount"`
}

// keychainCredentials represents the structure in keychain
type keychainCredentials struct {
	ClaudeAIOAuth struct {
		AccessToken string `json:"accessToken"`
	} `json:"claudeAiOauth"`
}

// NewClient creates a new Cloud API client using local credentials
func NewClient() (*Client, error) {
	token, err := loadTokenFromKeychain()
	if err != nil {
		return nil, fmt.Errorf("failed to load token from keychain: %w", err)
	}

	orgUUID, err := loadOrgUUIDFromConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load org UUID from config: %w", err)
	}

	return &Client{
		token:   token,
		orgUUID: orgUUID,
		http:    &http.Client{},
	}, nil
}

// loadTokenFromKeychain reads the OAuth token from macOS Keychain
func loadTokenFromKeychain() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	cmd := exec.Command("security", "find-generic-password",
		"-a", usr.Username,
		"-s", keychainService,
		"-w")

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("keychain lookup failed (is Claude Code installed and logged in?): %w", err)
	}

	var creds keychainCredentials
	if err := json.Unmarshal(out, &creds); err != nil {
		return "", fmt.Errorf("failed to parse keychain credentials: %w", err)
	}

	if creds.ClaudeAIOAuth.AccessToken == "" {
		return "", fmt.Errorf("no access token found in keychain")
	}

	return creds.ClaudeAIOAuth.AccessToken, nil
}

// loadOrgUUIDFromConfig reads the organization UUID from ~/.claude.json
func loadOrgUUIDFromConfig() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	configPath := filepath.Join(usr.HomeDir, ".claude.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read ~/.claude.json: %w", err)
	}

	var config claudeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("failed to parse ~/.claude.json: %w", err)
	}

	if config.OAuthAccount.OrganizationUUID == "" {
		return "", fmt.Errorf("no organization UUID found in ~/.claude.json")
	}

	return config.OAuthAccount.OrganizationUUID, nil
}

// doRequest performs an authenticated API request
func (c *Client) doRequest(method, path string) ([]byte, error) {
	url := baseURL + path

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-organization-uuid", c.orgUUID)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// ListSessions returns recent cloud sessions
func (c *Client) ListSessions(limit int) (*SessionsResponse, error) {
	path := fmt.Sprintf("/v1/sessions?limit=%d", limit)

	body, err := c.doRequest("GET", path)
	if err != nil {
		return nil, err
	}

	var resp SessionsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse sessions response: %w", err)
	}

	return &resp, nil
}

// GetSession returns a specific session by ID
func (c *Client) GetSession(sessionID string) (*Session, error) {
	path := "/v1/sessions/" + sessionID

	body, err := c.doRequest("GET", path)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, fmt.Errorf("failed to parse session response: %w", err)
	}

	return &session, nil
}

// GetSessionEvents returns all events for a session
func (c *Client) GetSessionEvents(sessionID string, limit int) (*EventsResponse, error) {
	path := fmt.Sprintf("/v1/sessions/%s/events?limit=%d", sessionID, limit)

	body, err := c.doRequest("GET", path)
	if err != nil {
		return nil, err
	}

	var resp EventsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse events response: %w", err)
	}

	return &resp, nil
}

// GetAllSessionEvents fetches all events for a session
// Uses the maximum allowed limit (1000) since the API doesn't support cursor pagination
func (c *Client) GetAllSessionEvents(sessionID string) ([]Event, error) {
	// API max limit is 1000, which should cover most sessions
	resp, err := c.GetSessionEvents(sessionID, 1000)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// FindSessionByBranch finds a session that matches the given branch name
func (c *Client) FindSessionByBranch(branchName string) (*Session, error) {
	// Fetch recent sessions
	resp, err := c.ListSessions(50)
	if err != nil {
		return nil, err
	}

	// Look for a session with matching branch
	for _, sess := range resp.Data {
		for _, outcome := range sess.SessionContext.Outcomes {
			if outcome.Type == "git_repository" {
				for _, branch := range outcome.GitInfo.Branches {
					if branch == branchName || strings.HasSuffix(branchName, branch) || strings.HasSuffix(branch, branchName) {
						return &sess, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("no session found for branch: %s", branchName)
}
