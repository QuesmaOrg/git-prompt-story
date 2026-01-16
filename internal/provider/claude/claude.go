package claude

import (
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/provider"
	"github.com/QuesmaOrg/git-prompt-story/internal/session"
)

func init() {
	provider.Register(&Provider{})
}

// Provider implements the provider.Provider interface for Claude Code
type Provider struct{}

// Name returns the tool identifier
func (p *Provider) Name() string {
	return "claude-code"
}

// FileExtension returns the file extension for Claude Code transcripts
func (p *Provider) FileExtension() string {
	return ".jsonl"
}

// DiscoverSessions finds Claude Code sessions for a repo path
func (p *Provider) DiscoverSessions(repoPath string, startWork, endWork time.Time) ([]provider.RawSession, error) {
	// Use existing session discovery logic
	sessions, err := session.FindSessions(repoPath, startWork, endWork, nil)
	if err != nil {
		return nil, err
	}

	// Filter to sessions with actual user messages
	sessions = session.FilterSessionsByUserMessages(sessions, startWork, endWork, nil)

	// Convert to RawSession
	result := make([]provider.RawSession, len(sessions))
	for i, s := range sessions {
		result[i] = provider.RawSession{
			ID:       s.ID,
			Tool:     "claude-code",
			Path:     s.Path,
			Created:  s.Created,
			Modified: s.Modified,
			RepoPath: repoPath,
		}
	}
	return result, nil
}

// ReadTranscript reads the raw JSONL content for storage
func (p *Provider) ReadTranscript(sess provider.RawSession) ([]byte, error) {
	return session.ReadSessionContent(sess.Path)
}
