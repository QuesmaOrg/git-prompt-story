package session

import "time"

// TraceContext captures decisions made during session discovery and filtering.
// When nil is passed to functions, they operate normally without tracing overhead.
type TraceContext struct {
	RepoPath         string
	EncodedPath      string
	SessionDir       string
	SessionDirExists bool
	FoundFiles       []string

	// Extended discovery fields
	ScanAllSessions bool     // Whether full scan mode was enabled
	CandidateDirs   []string // All candidate directories checked
	SkippedByMtime  int      // Files skipped due to mtime pre-filter

	WorkPeriod WorkPeriodTrace
	Sessions   []SessionTrace
}

// WorkPeriodTrace explains how the work period was calculated
type WorkPeriodTrace struct {
	IsAmend             bool
	Ref                 string
	PrevCommitTimestamp time.Time
	BranchSwitchTime    time.Time
	CalculatedStart     time.Time
	EndWork             time.Time
	Explanation         string
}

// SessionTrace explains the decision for a single session
type SessionTrace struct {
	ID       string
	Path     string
	Created  time.Time
	Modified time.Time

	// Time filter results
	TimeFilterPassed bool
	TimeFilterReason string

	// User message filter results
	UserMsgPassed bool
	UserMsgCount  int
	UserMsgReason string

	// Final decision
	Included    bool
	FinalReason string
}

// FindOrCreateSessionTrace finds an existing trace for a session or creates a new one
func (t *TraceContext) FindOrCreateSessionTrace(id string) *SessionTrace {
	for i := range t.Sessions {
		if t.Sessions[i].ID == id {
			return &t.Sessions[i]
		}
	}
	t.Sessions = append(t.Sessions, SessionTrace{ID: id})
	return &t.Sessions[len(t.Sessions)-1]
}
