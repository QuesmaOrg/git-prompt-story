package git

import (
	"bufio"
	"os/exec"
	"strings"
	"time"
)

const timestampLayout = "2006-01-02 15:04:05 -0700"

// GetPreviousCommitTimestamp returns the timestamp of HEAD (current commit)
// During prepare-commit-msg, HEAD is the previous commit (what will become HEAD~1)
// Returns zero time if there is no commit yet
func GetPreviousCommitTimestamp() (time.Time, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%ai", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		// No parent commit (initial commit case)
		return time.Time{}, nil
	}

	ts := strings.TrimSpace(string(out))
	if ts == "" {
		return time.Time{}, nil
	}

	t, err := time.Parse(timestampLayout, ts)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// GetLastBranchSwitchTimestamp finds the most recent checkout action in reflog
// Returns zero time if no checkout is found
func GetLastBranchSwitchTimestamp() (time.Time, error) {
	// Get reflog with timestamps and actions
	cmd := exec.Command("git", "reflog", "--format=%ai %gs")
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, nil
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()

		// Look for checkout entries (branch switches)
		if strings.Contains(line, "checkout:") {
			// Line format: "2025-12-29 20:08:35 +0200 checkout: moving from main to feature"
			// Extract timestamp (first 25 characters)
			if len(line) >= 25 {
				tsStr := line[:25]
				t, err := time.Parse(timestampLayout, tsStr)
				if err == nil {
					return t.UTC(), nil
				}
			}
		}
	}

	return time.Time{}, nil
}

// CalculateWorkStartTime determines the start of work for the current commit
// Returns the most recent of: previous commit timestamp or branch switch timestamp
func CalculateWorkStartTime() (time.Time, error) {
	prevTime, err := GetPreviousCommitTimestamp()
	if err != nil {
		prevTime = time.Time{}
	}

	switchTime, err := GetLastBranchSwitchTimestamp()
	if err != nil {
		switchTime = time.Time{}
	}

	// Return the more recent of the two timestamps
	if switchTime.IsZero() && prevTime.IsZero() {
		return time.Time{}, nil
	}

	if switchTime.IsZero() {
		return prevTime, nil
	}

	if prevTime.IsZero() {
		return switchTime, nil
	}

	// Return max (most recent)
	if switchTime.After(prevTime) {
		return switchTime, nil
	}

	return prevTime, nil
}
