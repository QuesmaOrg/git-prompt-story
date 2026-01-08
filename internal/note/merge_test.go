package note

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMergeNotes_Empty(t *testing.T) {
	result := MergeNotes(nil)
	if result != nil {
		t.Errorf("Expected nil for empty input, got %v", result)
	}

	result = MergeNotes([]*PromptStoryNote{})
	if result != nil {
		t.Errorf("Expected nil for empty slice, got %v", result)
	}
}

func TestMergeNotes_Single(t *testing.T) {
	note := &PromptStoryNote{
		Version:   1,
		StartWork: time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC),
		Sessions: []SessionEntry{
			{PromptTool: "claude-code", ID: "session-A", Created: time.Date(2025, 1, 15, 9, 15, 0, 0, time.UTC)},
		},
	}

	result := MergeNotes([]*PromptStoryNote{note})

	// Single note should be returned as-is
	if result != note {
		t.Error("Single note should be returned unchanged")
	}
}

func TestMergeNotes_TwoNotes(t *testing.T) {
	note1 := &PromptStoryNote{
		Version:   1,
		StartWork: time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC),
		Sessions: []SessionEntry{
			{
				PromptTool: "claude-code",
				ID:       "session-A",
				Path:     "claude-code/session-A.jsonl",
				Created:  time.Date(2025, 1, 15, 9, 15, 0, 0, time.UTC),
				Modified: time.Date(2025, 1, 15, 9, 25, 0, 0, time.UTC),
			},
		},
	}

	note2 := &PromptStoryNote{
		Version:   1,
		StartWork: time.Date(2025, 1, 15, 9, 30, 0, 0, time.UTC),
		Sessions: []SessionEntry{
			{
				PromptTool: "claude-code",
				ID:       "session-B",
				Path:     "claude-code/session-B.jsonl",
				Created:  time.Date(2025, 1, 15, 9, 45, 0, 0, time.UTC),
				Modified: time.Date(2025, 1, 15, 9, 55, 0, 0, time.UTC),
			},
		},
	}

	result := MergeNotes([]*PromptStoryNote{note1, note2})

	// Should have 2 sessions
	if len(result.Sessions) != 2 {
		t.Fatalf("Expected 2 sessions, got %d", len(result.Sessions))
	}

	// Sessions should be sorted by created time
	if result.Sessions[0].ID != "session-A" {
		t.Errorf("Expected first session to be 'session-A', got %q", result.Sessions[0].ID)
	}
	if result.Sessions[1].ID != "session-B" {
		t.Errorf("Expected second session to be 'session-B', got %q", result.Sessions[1].ID)
	}

	// StartWork should be the earliest
	expectedStartWork := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
	if !result.StartWork.Equal(expectedStartWork) {
		t.Errorf("Expected StartWork %v, got %v", expectedStartWork, result.StartWork)
	}
}

func TestMergeNotes_Deduplication(t *testing.T) {
	// Same session ID in both notes should be deduplicated
	note1 := &PromptStoryNote{
		Version:   1,
		StartWork: time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC),
		Sessions: []SessionEntry{
			{PromptTool: "claude-code", ID: "session-A", Created: time.Date(2025, 1, 15, 9, 15, 0, 0, time.UTC)},
		},
	}

	note2 := &PromptStoryNote{
		Version:   1,
		StartWork: time.Date(2025, 1, 15, 9, 30, 0, 0, time.UTC),
		Sessions: []SessionEntry{
			{PromptTool: "claude-code", ID: "session-A", Created: time.Date(2025, 1, 15, 9, 15, 0, 0, time.UTC)}, // Duplicate
			{PromptTool: "claude-code", ID: "session-B", Created: time.Date(2025, 1, 15, 9, 45, 0, 0, time.UTC)},
		},
	}

	result := MergeNotes([]*PromptStoryNote{note1, note2})

	// Should have 2 sessions (session-A deduplicated)
	if len(result.Sessions) != 2 {
		t.Fatalf("Expected 2 sessions after deduplication, got %d", len(result.Sessions))
	}

	// Count occurrences of session-A
	countA := 0
	for _, s := range result.Sessions {
		if s.ID == "session-A" {
			countA++
		}
	}
	if countA != 1 {
		t.Errorf("Expected session-A to appear once, appeared %d times", countA)
	}
}

func TestMergeNotes_EarliestStartWork(t *testing.T) {
	// Second note has earlier StartWork - should use that
	note1 := &PromptStoryNote{
		Version:   1,
		StartWork: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		Sessions:  []SessionEntry{{PromptTool: "claude-code", ID: "session-A", Created: time.Date(2025, 1, 15, 10, 15, 0, 0, time.UTC)}},
	}

	note2 := &PromptStoryNote{
		Version:   1,
		StartWork: time.Date(2025, 1, 15, 8, 0, 0, 0, time.UTC), // Earlier
		Sessions:  []SessionEntry{{PromptTool: "claude-code", ID: "session-B", Created: time.Date(2025, 1, 15, 8, 15, 0, 0, time.UTC)}},
	}

	result := MergeNotes([]*PromptStoryNote{note1, note2})

	expectedStartWork := time.Date(2025, 1, 15, 8, 0, 0, 0, time.UTC)
	if !result.StartWork.Equal(expectedStartWork) {
		t.Errorf("Expected StartWork %v (earliest), got %v", expectedStartWork, result.StartWork)
	}
}

func TestMergeNotes_LatestVersion(t *testing.T) {
	note1 := &PromptStoryNote{
		Version:   1,
		StartWork: time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC),
		Sessions:  []SessionEntry{{PromptTool: "claude-code", ID: "session-A", Created: time.Date(2025, 1, 15, 9, 15, 0, 0, time.UTC)}},
	}

	note2 := &PromptStoryNote{
		Version:   2, // Higher version
		StartWork: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		Sessions:  []SessionEntry{{PromptTool: "claude-code", ID: "session-B", Created: time.Date(2025, 1, 15, 10, 15, 0, 0, time.UTC)}},
	}

	result := MergeNotes([]*PromptStoryNote{note1, note2})

	if result.Version != 2 {
		t.Errorf("Expected version 2 (latest), got %d", result.Version)
	}
}

func TestMergeNotes_MultipleTools(t *testing.T) {
	note1 := &PromptStoryNote{
		Version:   1,
		StartWork: time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC),
		Sessions: []SessionEntry{
			{PromptTool: "claude-code", ID: "session-A", Created: time.Date(2025, 1, 15, 9, 15, 0, 0, time.UTC)},
		},
	}

	note2 := &PromptStoryNote{
		Version:   1,
		StartWork: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		Sessions: []SessionEntry{
			{PromptTool: "cursor", ID: "session-B", Created: time.Date(2025, 1, 15, 10, 15, 0, 0, time.UTC)},
		},
	}

	result := MergeNotes([]*PromptStoryNote{note1, note2})

	if len(result.Sessions) != 2 {
		t.Fatalf("Expected 2 sessions, got %d", len(result.Sessions))
	}

	// Verify both tools are present
	tools := make(map[string]bool)
	for _, s := range result.Sessions {
		tools[s.PromptTool] = true
	}
	if !tools["claude-code"] || !tools["cursor"] {
		t.Errorf("Expected both claude-code and cursor tools, got %v", tools)
	}
}

func TestMergeNotes_ThreeNotes(t *testing.T) {
	note1 := &PromptStoryNote{
		Version:   1,
		StartWork: time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC),
		Sessions:  []SessionEntry{{PromptTool: "claude-code", ID: "session-A", Created: time.Date(2025, 1, 15, 9, 15, 0, 0, time.UTC)}},
	}

	note2 := &PromptStoryNote{
		Version:   1,
		StartWork: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		Sessions:  []SessionEntry{{PromptTool: "claude-code", ID: "session-B", Created: time.Date(2025, 1, 15, 10, 15, 0, 0, time.UTC)}},
	}

	note3 := &PromptStoryNote{
		Version:   1,
		StartWork: time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC),
		Sessions:  []SessionEntry{{PromptTool: "claude-code", ID: "session-C", Created: time.Date(2025, 1, 15, 11, 15, 0, 0, time.UTC)}},
	}

	result := MergeNotes([]*PromptStoryNote{note1, note2, note3})

	if len(result.Sessions) != 3 {
		t.Fatalf("Expected 3 sessions, got %d", len(result.Sessions))
	}

	// Verify sorted by created time
	if result.Sessions[0].ID != "session-A" || result.Sessions[1].ID != "session-B" || result.Sessions[2].ID != "session-C" {
		t.Errorf("Sessions not sorted correctly: %v, %v, %v",
			result.Sessions[0].ID, result.Sessions[1].ID, result.Sessions[2].ID)
	}
}

func TestMergeNotes_PreservesAllMetadata(t *testing.T) {
	note := &PromptStoryNote{
		Version:   1,
		StartWork: time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC),
		Sessions: []SessionEntry{
			{
				PromptTool: "claude-code",
				ID:       "session-A",
				Path:     "claude-code/session-A.jsonl",
				Created:  time.Date(2025, 1, 15, 9, 15, 0, 0, time.UTC),
				Modified: time.Date(2025, 1, 15, 9, 45, 0, 0, time.UTC),
			},
		},
	}

	note2 := &PromptStoryNote{
		Version:   1,
		StartWork: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		Sessions: []SessionEntry{
			{
				PromptTool: "claude-code",
				ID:       "session-B",
				Path:     "claude-code/session-B.jsonl",
				Created:  time.Date(2025, 1, 15, 10, 15, 0, 0, time.UTC),
				Modified: time.Date(2025, 1, 15, 10, 45, 0, 0, time.UTC),
			},
		},
	}

	result := MergeNotes([]*PromptStoryNote{note, note2})

	// Verify all session metadata is preserved
	for _, s := range result.Sessions {
		if s.Path == "" {
			t.Errorf("Session %s: Path should be preserved", s.ID)
		}
		if s.PromptTool == "" {
			t.Errorf("Session %s: Tool should be preserved", s.ID)
		}
		if s.Created.IsZero() {
			t.Errorf("Session %s: Created should be preserved", s.ID)
		}
		if s.Modified.IsZero() {
			t.Errorf("Session %s: Modified should be preserved", s.ID)
		}
	}
}

func TestMergeNotes_EmptySessions(t *testing.T) {
	note1 := &PromptStoryNote{
		Version:   1,
		StartWork: time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC),
		Sessions:  []SessionEntry{}, // Empty
	}

	note2 := &PromptStoryNote{
		Version:   1,
		StartWork: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		Sessions: []SessionEntry{
			{PromptTool: "claude-code", ID: "session-A", Created: time.Date(2025, 1, 15, 10, 15, 0, 0, time.UTC)},
		},
	}

	result := MergeNotes([]*PromptStoryNote{note1, note2})

	if len(result.Sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(result.Sessions))
	}

	// StartWork should still be the earliest
	expectedStartWork := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
	if !result.StartWork.Equal(expectedStartWork) {
		t.Errorf("Expected StartWork %v, got %v", expectedStartWork, result.StartWork)
	}
}

func TestParseNote_Valid(t *testing.T) {
	data := `{
		"v": 1,
		"start_work": "2025-01-15T09:00:00Z",
		"sessions": [
			{
				"tool": "claude-code",
				"id": "session-A",
				"path": "claude-code/session-A.jsonl",
				"created": "2025-01-15T09:15:00Z",
				"modified": "2025-01-15T09:45:00Z"
			}
		]
	}`

	note, err := ParseNote([]byte(data))
	if err != nil {
		t.Fatalf("ParseNote() error: %v", err)
	}

	if note.Version != 1 {
		t.Errorf("Expected version 1, got %d", note.Version)
	}

	expectedStartWork := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
	if !note.StartWork.Equal(expectedStartWork) {
		t.Errorf("Expected StartWork %v, got %v", expectedStartWork, note.StartWork)
	}

	if len(note.Sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(note.Sessions))
	}

	if note.Sessions[0].ID != "session-A" {
		t.Errorf("Expected session ID 'session-A', got %q", note.Sessions[0].ID)
	}
}

func TestParseNote_Invalid(t *testing.T) {
	data := `{invalid json}`

	_, err := ParseNote([]byte(data))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestParseNote_Empty(t *testing.T) {
	data := `{}`

	note, err := ParseNote([]byte(data))
	if err != nil {
		t.Fatalf("ParseNote() error: %v", err)
	}

	if note.Version != 0 {
		t.Errorf("Expected version 0, got %d", note.Version)
	}

	if len(note.Sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(note.Sessions))
	}
}

func TestMergeNotes_ResultIsSerializable(t *testing.T) {
	note1 := &PromptStoryNote{
		Version:   1,
		StartWork: time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC),
		Sessions: []SessionEntry{
			{PromptTool: "claude-code", ID: "session-A", Path: "claude-code/session-A.jsonl", Created: time.Date(2025, 1, 15, 9, 15, 0, 0, time.UTC), Modified: time.Date(2025, 1, 15, 9, 25, 0, 0, time.UTC)},
		},
	}

	note2 := &PromptStoryNote{
		Version:   1,
		StartWork: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		Sessions: []SessionEntry{
			{PromptTool: "claude-code", ID: "session-B", Path: "claude-code/session-B.jsonl", Created: time.Date(2025, 1, 15, 10, 15, 0, 0, time.UTC), Modified: time.Date(2025, 1, 15, 10, 25, 0, 0, time.UTC)},
		},
	}

	result := MergeNotes([]*PromptStoryNote{note1, note2})

	// Verify result can be serialized
	data, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	// Verify it can be parsed back
	parsed, err := ParseNote(data)
	if err != nil {
		t.Fatalf("ParseNote() error: %v", err)
	}

	if len(parsed.Sessions) != 2 {
		t.Errorf("Expected 2 sessions after roundtrip, got %d", len(parsed.Sessions))
	}

	// Verify JSON structure is correct
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if _, ok := raw["v"]; !ok {
		t.Error("Expected 'v' field in JSON")
	}
	if _, ok := raw["start_work"]; !ok {
		t.Error("Expected 'start_work' field in JSON")
	}
	if _, ok := raw["sessions"]; !ok {
		t.Error("Expected 'sessions' field in JSON")
	}
}
