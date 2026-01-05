package git

import (
	"bufio"
	"os/exec"
	"strings"
	"time"
)

const timestampLayout = "2006-01-02 15:04:05 -0700"

// GetPreviousCommitTimestamp returns the timestamp of the previous commit
// The ref parameter specifies which commit to get:
// - For normal commits: use "HEAD" (HEAD is the previous commit during prepare-commit-msg)
// - For amend: use "HEAD^" (the parent of the commit being amended)
// Returns zero time if there is no commit yet
func GetPreviousCommitTimestamp(ref string) (time.Time, error) {
	if ref == "" {
		ref = "HEAD"
	}
	cmd := exec.Command("git", "log", "-1", "--format=%ai", ref)
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

// WorkPeriodTrace captures how the work period was calculated (for explainability)
type WorkPeriodTrace struct {
	IsAmend             bool
	Ref                 string
	PrevCommitTimestamp time.Time
	BranchSwitchTime    time.Time
	CalculatedStart     time.Time
	Explanation         string
}

// CalculateWorkStartTime determines the start of work for the current commit
// Returns the most recent of: previous commit timestamp or branch switch timestamp
// isAmend: set to true when amending a commit (uses HEAD^ instead of HEAD)
func CalculateWorkStartTime(isAmend bool) (time.Time, error) {
	result, _, err := CalculateWorkStartTimeWithTrace(isAmend)
	return result, err
}

// CalculateWorkStartTimeWithTrace is like CalculateWorkStartTime but also returns trace info
func CalculateWorkStartTimeWithTrace(isAmend bool) (time.Time, *WorkPeriodTrace, error) {
	trace := &WorkPeriodTrace{
		IsAmend: isAmend,
	}

	ref := "HEAD"
	if isAmend {
		ref = "HEAD^"
	}
	trace.Ref = ref

	prevTime, err := GetPreviousCommitTimestamp(ref)
	if err != nil {
		prevTime = time.Time{}
	}
	trace.PrevCommitTimestamp = prevTime

	switchTime, err := GetLastBranchSwitchTimestamp()
	if err != nil {
		switchTime = time.Time{}
	}
	trace.BranchSwitchTime = switchTime

	// Return the more recent of the two timestamps
	if switchTime.IsZero() && prevTime.IsZero() {
		trace.Explanation = "No previous commit or branch switch found (initial commit)"
		trace.CalculatedStart = time.Time{}
		return time.Time{}, trace, nil
	}

	if switchTime.IsZero() {
		trace.Explanation = "Using previous commit timestamp (no branch switch found)"
		trace.CalculatedStart = prevTime
		return prevTime, trace, nil
	}

	if prevTime.IsZero() {
		trace.Explanation = "Using branch switch timestamp (no previous commit)"
		trace.CalculatedStart = switchTime
		return switchTime, trace, nil
	}

	// Return max (most recent)
	if switchTime.After(prevTime) {
		trace.Explanation = "Using branch switch timestamp (more recent than commit)"
		trace.CalculatedStart = switchTime
		return switchTime, trace, nil
	}

	trace.Explanation = "Using previous commit timestamp (more recent than branch switch)"
	trace.CalculatedStart = prevTime
	return prevTime, trace, nil
}
