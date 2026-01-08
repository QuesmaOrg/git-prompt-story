package repair

import (
	"fmt"
	"strings"
	"time"

	"github.com/QuesmaOrg/git-prompt-story/internal/git"
	"github.com/QuesmaOrg/git-prompt-story/internal/note"
	"github.com/QuesmaOrg/git-prompt-story/internal/scrubber"
	"github.com/QuesmaOrg/git-prompt-story/internal/session"
)

// RepairResult holds the result of a repair operation
type RepairResult struct {
	CommitSHA      string
	ShortSHA       string
	SessionsFound  int
	NoteCreated    bool
	NoteSHA        string
	AlreadyHasNote bool
	Error          error
}

// Options configures repair behavior
type Options struct {
	DryRun  bool
	Force   bool // overwrite existing notes
	NoScrub bool
}

// RepairCommit attempts to recreate a missing note for a commit
func RepairCommit(sha string, opts Options) (*RepairResult, error) {
	result := &RepairResult{
		CommitSHA: sha,
	}

	// Resolve full SHA
	fullSHA, err := git.ResolveCommit(sha)
	if err != nil {
		return nil, fmt.Errorf("invalid commit: %w", err)
	}
	result.CommitSHA = fullSHA
	result.ShortSHA = fullSHA[:7]

	// Check if note already exists
	existingNote, err := note.GetNote(fullSHA)
	if err == nil && existingNote != "" {
		result.AlreadyHasNote = true
		if !opts.Force {
			return result, nil
		}
	}

	// Get repo root
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("not in a git repository: %w", err)
	}

	// Calculate work period for this commit
	startWork, endWork, err := getWorkPeriodForCommit(fullSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to get work period: %w", err)
	}

	// Find sessions from all registered prompt tools (includes time and user message filtering)
	sessions, err := session.FindAllSessions(repoRoot, startWork, endWork, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to find sessions: %w", err)
	}

	result.SessionsFound = len(sessions)

	if len(sessions) == 0 {
		return result, nil
	}

	if opts.DryRun {
		result.NoteCreated = true // would be created
		return result, nil
	}

	// Create scrubber
	var piiScrubber scrubber.Scrubber
	if !opts.NoScrub {
		piiScrubber, err = scrubber.NewDefault()
		if err != nil {
			return nil, fmt.Errorf("failed to create scrubber: %w", err)
		}
	}

	// Store transcripts
	blobs, err := note.StoreTranscripts(sessions, piiScrubber)
	if err != nil {
		return nil, fmt.Errorf("failed to store transcripts: %w", err)
	}

	// Update transcript tree
	if err := note.UpdateTranscriptTree(blobs); err != nil {
		return nil, fmt.Errorf("failed to update transcript tree: %w", err)
	}

	// Create note with explicit start time (not using CalculateWorkStartTime)
	psNote := note.NewPromptStoryNote(sessions, false, startWork)
	noteJSON, err := psNote.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize note: %w", err)
	}

	// Attach note to commit
	if err := git.AddNote(note.NotesRef, string(noteJSON), fullSHA); err != nil {
		return nil, fmt.Errorf("failed to attach note: %w", err)
	}

	// Get the note SHA for reporting
	noteSHA, _ := git.HashObject(noteJSON)
	result.NoteSHA = noteSHA
	result.NoteCreated = true

	return result, nil
}

// getWorkPeriodForCommit calculates the work period for an existing commit
func getWorkPeriodForCommit(sha string) (startWork, endWork time.Time, err error) {
	// End of work period = commit timestamp
	endWork, err = git.GetCommitTimestamp(sha)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to get commit timestamp: %w", err)
	}

	// Start of work period = parent commit timestamp
	parentSHA, err := getParentCommit(sha)
	if err != nil || parentSHA == "" {
		// First commit or error - use a reasonable default (24 hours before)
		startWork = endWork.Add(-24 * time.Hour)
		return startWork, endWork, nil
	}

	startWork, err = git.GetCommitTimestamp(parentSHA)
	if err != nil {
		// Fallback to 24 hours before
		startWork = endWork.Add(-24 * time.Hour)
		return startWork, endWork, nil
	}

	return startWork, endWork, nil
}

// getParentCommit returns the SHA of the parent commit
func getParentCommit(sha string) (string, error) {
	return git.RunGit("rev-parse", sha+"^")
}

// getCommitMessage returns the commit message for a commit
func getCommitMessage(sha string) (string, error) {
	return git.RunGit("log", "-1", "--format=%B", sha)
}

// ScanCommitsNeedingRepair finds commits that have Prompt-Story markers but no notes
func ScanCommitsNeedingRepair(commitRange string) ([]string, error) {
	var commits []string

	// Get commits in range
	var shas []string
	var err error
	if strings.Contains(commitRange, "..") {
		shas, err = git.RevList(commitRange)
	} else if commitRange != "" {
		// Single commit or ref
		sha, err := git.ResolveCommit(commitRange)
		if err != nil {
			return nil, err
		}
		shas = []string{sha}
	} else {
		// Default: scan recent commits on current branch
		shas, err = git.RevList("HEAD~20..HEAD")
		if err != nil {
			// Maybe fewer than 20 commits, try all
			shas, err = git.RevList("HEAD")
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list commits: %w", err)
	}

	for _, sha := range shas {
		// Check if commit has Prompt-Story marker
		msg, err := getCommitMessage(sha)
		if err != nil {
			continue
		}
		if !strings.Contains(msg, "Prompt-Story:") {
			continue
		}

		// Check if note exists
		_, err = note.GetNote(sha)
		if err != nil {
			// No note found - needs repair
			commits = append(commits, sha)
		}
	}

	return commits, nil
}
