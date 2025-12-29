package session

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"time"
)

// ParseSessionMetadata extracts first/last timestamps and branch from JSONL
func ParseSessionMetadata(sessionPath string) (created, modified time.Time, branch string, err error) {
	file, err := os.Open(sessionPath)
	if err != nil {
		return time.Time{}, time.Time{}, "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Increase buffer size for large lines (Claude responses can be big)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var first, last time.Time
	var lastBranch string

	for scanner.Scan() {
		var entry MessageEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // Skip malformed lines
		}

		// Get timestamp from appropriate location
		ts := entry.Timestamp
		if ts.IsZero() && entry.Snapshot != nil {
			ts = entry.Snapshot.Timestamp
		}

		if !ts.IsZero() {
			if first.IsZero() {
				first = ts
			}
			last = ts
		}

		if entry.GitBranch != "" {
			lastBranch = entry.GitBranch
		}
	}

	if err := scanner.Err(); err != nil {
		return time.Time{}, time.Time{}, "", err
	}

	return first, last, lastBranch, nil
}

// ReadSessionContent reads the entire session file for storage
func ReadSessionContent(sessionPath string) ([]byte, error) {
	return os.ReadFile(sessionPath)
}

// ParseMessages parses JSONL content and returns all message entries
func ParseMessages(content []byte) ([]MessageEntry, error) {
	var entries []MessageEntry

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	// Increase buffer size for large lines (Claude responses can be big)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		var entry MessageEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // Skip malformed lines
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}
