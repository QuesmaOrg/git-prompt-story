package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseMessages_ValidJSONL(t *testing.T) {
	content := `{"type":"user","sessionId":"test-123","timestamp":"2025-01-15T09:15:00Z","message":{"role":"user","content":"Hello"}}
{"type":"assistant","sessionId":"test-123","timestamp":"2025-01-15T09:16:00Z","message":{"role":"assistant","content":[{"type":"text","text":"Hi there!"}]}}`

	entries, err := ParseMessages([]byte(content))
	if err != nil {
		t.Fatalf("ParseMessages() error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}

	// Verify first entry (user)
	if entries[0].Type != "user" {
		t.Errorf("Entry 0: expected type 'user', got %q", entries[0].Type)
	}
	if entries[0].SessionID != "test-123" {
		t.Errorf("Entry 0: expected sessionId 'test-123', got %q", entries[0].SessionID)
	}
	if entries[0].Message == nil || entries[0].Message.Role != "user" {
		t.Errorf("Entry 0: expected message role 'user'")
	}

	// Verify second entry (assistant)
	if entries[1].Type != "assistant" {
		t.Errorf("Entry 1: expected type 'assistant', got %q", entries[1].Type)
	}
	if entries[1].Message == nil || entries[1].Message.Role != "assistant" {
		t.Errorf("Entry 1: expected message role 'assistant'")
	}
}

func TestParseMessages_MalformedLines(t *testing.T) {
	// Mix of valid and invalid lines - invalid should be skipped
	content := `{"type":"user","sessionId":"test","timestamp":"2025-01-15T09:15:00Z","message":{"role":"user","content":"Valid"}}
{invalid json line
{"type":"assistant","sessionId":"test","timestamp":"2025-01-15T09:16:00Z","message":{"role":"assistant","content":"Also valid"}}`

	entries, err := ParseMessages([]byte(content))
	if err != nil {
		t.Fatalf("ParseMessages() error: %v", err)
	}

	// Should have 2 entries (malformed line skipped)
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries (malformed skipped), got %d", len(entries))
	}

	if entries[0].Type != "user" {
		t.Errorf("Entry 0: expected type 'user', got %q", entries[0].Type)
	}
	if entries[1].Type != "assistant" {
		t.Errorf("Entry 1: expected type 'assistant', got %q", entries[1].Type)
	}
}

func TestParseMessages_EmptyContent(t *testing.T) {
	entries, err := ParseMessages([]byte(""))
	if err != nil {
		t.Fatalf("ParseMessages() error: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries for empty content, got %d", len(entries))
	}
}

func TestParseMessages_ToolUseMessages(t *testing.T) {
	content := `{"type":"assistant","sessionId":"test","timestamp":"2025-01-15T09:15:00Z","message":{"role":"assistant","content":[{"type":"text","text":"Let me read that file"},{"type":"tool_use","id":"tool1","name":"Read","input":{"file_path":"/test.go"}}]}}
{"type":"user","sessionId":"test","timestamp":"2025-01-15T09:16:00Z","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool1","content":"file contents"}]}}`

	entries, err := ParseMessages([]byte(content))
	if err != nil {
		t.Fatalf("ParseMessages() error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}

	// Verify tool use entry
	if entries[0].Type != "assistant" {
		t.Errorf("Entry 0: expected type 'assistant', got %q", entries[0].Type)
	}
}

func TestParseMessages_LargeContent(t *testing.T) {
	// Create a large message (> 64KB) to test buffer handling
	largeText := strings.Repeat("a", 100*1024) // 100KB of text
	content := `{"type":"user","sessionId":"test","timestamp":"2025-01-15T09:15:00Z","message":{"role":"user","content":"` + largeText + `"}}`

	entries, err := ParseMessages([]byte(content))
	if err != nil {
		t.Fatalf("ParseMessages() error: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	// Verify the content was preserved
	text := entries[0].Message.GetTextContent()
	if len(text) != 100*1024 {
		t.Errorf("Expected content length 102400, got %d", len(text))
	}
}

func TestParseMessages_Timestamps(t *testing.T) {
	content := `{"type":"user","sessionId":"test","timestamp":"2025-01-15T09:15:30Z","message":{"role":"user","content":"Hello"}}
{"type":"assistant","sessionId":"test","timestamp":"2025-01-15T14:30:00Z","message":{"role":"assistant","content":"World"}}`

	entries, err := ParseMessages([]byte(content))
	if err != nil {
		t.Fatalf("ParseMessages() error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}

	// Verify timestamps parsed correctly
	expectedFirst := time.Date(2025, 1, 15, 9, 15, 30, 0, time.UTC)
	if !entries[0].Timestamp.Equal(expectedFirst) {
		t.Errorf("Entry 0: expected timestamp %v, got %v", expectedFirst, entries[0].Timestamp)
	}

	expectedSecond := time.Date(2025, 1, 15, 14, 30, 0, 0, time.UTC)
	if !entries[1].Timestamp.Equal(expectedSecond) {
		t.Errorf("Entry 1: expected timestamp %v, got %v", expectedSecond, entries[1].Timestamp)
	}
}

func TestParseMessages_SnapshotTimestamp(t *testing.T) {
	// File history snapshot entries have timestamp in snapshot object
	content := `{"type":"file-history-snapshot","sessionId":"test","snapshot":{"timestamp":"2025-01-15T10:00:00Z"}}`

	entries, err := ParseMessages([]byte(content))
	if err != nil {
		t.Fatalf("ParseMessages() error: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].Type != "file-history-snapshot" {
		t.Errorf("Expected type 'file-history-snapshot', got %q", entries[0].Type)
	}

	// Verify snapshot timestamp
	expectedTs := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	if entries[0].Snapshot == nil {
		t.Fatal("Expected snapshot to be non-nil")
	}
	if !entries[0].Snapshot.Timestamp.Equal(expectedTs) {
		t.Errorf("Expected snapshot timestamp %v, got %v", expectedTs, entries[0].Snapshot.Timestamp)
	}
}

func TestParseSessionMetadata(t *testing.T) {
	// Create a temp file with session content
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "test-session.jsonl")

	content := `{"type":"user","sessionId":"test","timestamp":"2025-01-15T09:15:00Z","gitBranch":"main","message":{"role":"user","content":"Start"}}
{"type":"assistant","sessionId":"test","timestamp":"2025-01-15T09:20:00Z","message":{"role":"assistant","content":"Middle"}}
{"type":"user","sessionId":"test","timestamp":"2025-01-15T09:30:00Z","gitBranch":"feature","message":{"role":"user","content":"End"}}`

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	created, modified, branch, err := ParseSessionMetadata(sessionPath)
	if err != nil {
		t.Fatalf("ParseSessionMetadata() error: %v", err)
	}

	// Verify created (first timestamp)
	expectedCreated := time.Date(2025, 1, 15, 9, 15, 0, 0, time.UTC)
	if !created.Equal(expectedCreated) {
		t.Errorf("Expected created %v, got %v", expectedCreated, created)
	}

	// Verify modified (last timestamp)
	expectedModified := time.Date(2025, 1, 15, 9, 30, 0, 0, time.UTC)
	if !modified.Equal(expectedModified) {
		t.Errorf("Expected modified %v, got %v", expectedModified, modified)
	}

	// Verify branch (last non-empty branch)
	if branch != "feature" {
		t.Errorf("Expected branch 'feature', got %q", branch)
	}
}

func TestParseSessionMetadata_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "empty-session.jsonl")

	if err := os.WriteFile(sessionPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	created, modified, branch, err := ParseSessionMetadata(sessionPath)
	if err != nil {
		t.Fatalf("ParseSessionMetadata() error: %v", err)
	}

	// Empty file should return zero times and empty branch
	if !created.IsZero() {
		t.Errorf("Expected zero created time, got %v", created)
	}
	if !modified.IsZero() {
		t.Errorf("Expected zero modified time, got %v", modified)
	}
	if branch != "" {
		t.Errorf("Expected empty branch, got %q", branch)
	}
}

func TestParseSessionMetadata_NonExistentFile(t *testing.T) {
	_, _, _, err := ParseSessionMetadata("/non/existent/file.jsonl")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestParseSessionMetadata_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "large-session.jsonl")

	// Create content with large messages
	largeText := strings.Repeat("x", 100*1024) // 100KB
	var lines []string
	lines = append(lines, `{"type":"user","sessionId":"test","timestamp":"2025-01-15T09:00:00Z","gitBranch":"main","message":{"role":"user","content":"Start"}}`)
	lines = append(lines, `{"type":"assistant","sessionId":"test","timestamp":"2025-01-15T09:30:00Z","message":{"role":"assistant","content":"`+largeText+`"}}`)
	lines = append(lines, `{"type":"user","sessionId":"test","timestamp":"2025-01-15T10:00:00Z","gitBranch":"develop","message":{"role":"user","content":"End"}}`)

	content := strings.Join(lines, "\n")
	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	created, modified, branch, err := ParseSessionMetadata(sessionPath)
	if err != nil {
		t.Fatalf("ParseSessionMetadata() error: %v", err)
	}

	// Should handle large content and extract correct metadata
	expectedCreated := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
	if !created.Equal(expectedCreated) {
		t.Errorf("Expected created %v, got %v", expectedCreated, created)
	}

	expectedModified := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	if !modified.Equal(expectedModified) {
		t.Errorf("Expected modified %v, got %v", expectedModified, modified)
	}

	if branch != "develop" {
		t.Errorf("Expected branch 'develop', got %q", branch)
	}
}

func TestReadSessionContent(t *testing.T) {
	tmpDir := t.TempDir()
	sessionPath := filepath.Join(tmpDir, "read-test.jsonl")

	expected := `{"type":"user","message":"test"}`
	if err := os.WriteFile(sessionPath, []byte(expected), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	content, err := ReadSessionContent(sessionPath)
	if err != nil {
		t.Fatalf("ReadSessionContent() error: %v", err)
	}

	if string(content) != expected {
		t.Errorf("Expected %q, got %q", expected, string(content))
	}
}

func TestReadSessionContent_NonExistentFile(t *testing.T) {
	_, err := ReadSessionContent("/non/existent/file.jsonl")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestMessage_GetTextContent_StringContent(t *testing.T) {
	msg := &Message{
		Role:       "user",
		RawContent: []byte(`"Hello, world!"`),
	}

	text := msg.GetTextContent()
	if text != "Hello, world!" {
		t.Errorf("Expected 'Hello, world!', got %q", text)
	}
}

func TestMessage_GetTextContent_ArrayContent(t *testing.T) {
	msg := &Message{
		Role:       "assistant",
		RawContent: []byte(`[{"type":"text","text":"First part"},{"type":"tool_use","name":"Read"},{"type":"text","text":"Second part"}]`),
	}

	// GetTextContent returns the first text part
	text := msg.GetTextContent()
	if text != "First part" {
		t.Errorf("Expected 'First part', got %q", text)
	}
}

func TestMessage_GetTextContent_EmptyContent(t *testing.T) {
	msg := &Message{
		Role:       "user",
		RawContent: nil,
	}

	text := msg.GetTextContent()
	if text != "" {
		t.Errorf("Expected empty string, got %q", text)
	}
}

func TestMessage_GetTextContent_NoTextParts(t *testing.T) {
	msg := &Message{
		Role:       "assistant",
		RawContent: []byte(`[{"type":"tool_use","name":"Read"}]`),
	}

	text := msg.GetTextContent()
	if text != "" {
		t.Errorf("Expected empty string for no text parts, got %q", text)
	}
}

func TestMessage_GetTextContent_NilMessage(t *testing.T) {
	var msg *Message = nil
	text := msg.GetTextContent()
	if text != "" {
		t.Errorf("Expected empty string for nil message, got %q", text)
	}
}

func TestParseMessages_GitBranch(t *testing.T) {
	content := `{"type":"user","sessionId":"test","timestamp":"2025-01-15T09:15:00Z","cwd":"/workspace","gitBranch":"feature/test","message":{"role":"user","content":"Hello"}}`

	entries, err := ParseMessages([]byte(content))
	if err != nil {
		t.Fatalf("ParseMessages() error: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].GitBranch != "feature/test" {
		t.Errorf("Expected gitBranch 'feature/test', got %q", entries[0].GitBranch)
	}
}
