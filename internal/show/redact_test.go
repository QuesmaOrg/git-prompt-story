package show

import (
	"strings"
	"testing"
	"time"
)

func TestRedactJSONLEntry(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		timestamp   time.Time
		wantErr     bool
		errContains string
		checkOutput func(t *testing.T, output []byte)
	}{
		{
			name: "redact user message by timestamp",
			content: `{"timestamp":"2025-01-15T10:00:00Z","type":"user","message":{"content":"Hello world"}}
{"timestamp":"2025-01-15T10:01:00Z","type":"assistant","message":{"content":"Hi there"}}
`,
			timestamp: mustParseTime("2025-01-15T10:00:00Z"),
			wantErr:   false,
			checkOutput: func(t *testing.T, output []byte) {
				// JSON encodes < and > as \u003c and \u003e
				if !containsRedacted(string(output)) {
					t.Error("expected <REDACTED BY USER> in output")
				}
				// Second message should be unchanged
				if !strings.Contains(string(output), "Hi there") {
					t.Error("expected second message to be unchanged")
				}
			},
		},
		{
			name: "redact direct content field",
			content: `{"timestamp":"2025-01-15T10:00:00Z","type":"user","content":"Secret data"}
`,
			timestamp: mustParseTime("2025-01-15T10:00:00Z"),
			wantErr:   false,
			checkOutput: func(t *testing.T, output []byte) {
				// JSON encodes < and > as \u003c and \u003e
				if !containsRedacted(string(output)) {
					t.Error("expected <REDACTED BY USER> in output")
				}
				if strings.Contains(string(output), "Secret data") {
					t.Error("expected secret data to be redacted")
				}
			},
		},
		{
			name: "timestamp not found",
			content: `{"timestamp":"2025-01-15T10:00:00Z","type":"user","message":{"content":"Hello"}}
`,
			timestamp:   mustParseTime("2025-01-15T11:00:00Z"),
			wantErr:     true,
			errContains: "not found",
		},
		{
			name: "timestamp with millisecond tolerance",
			content: `{"timestamp":"2025-01-15T10:00:00.500Z","type":"user","message":{"content":"Hello world"}}
`,
			timestamp: mustParseTime("2025-01-15T10:00:00Z"),
			wantErr:   false,
			checkOutput: func(t *testing.T, output []byte) {
				if !containsRedacted(string(output)) {
					t.Error("expected <REDACTED BY USER> in output within tolerance")
				}
			},
		},
		{
			name: "preserves other entries",
			content: `{"timestamp":"2025-01-15T10:00:00Z","type":"user","message":{"content":"First"}}
{"timestamp":"2025-01-15T10:01:00Z","type":"assistant","message":{"content":"Second"}}
{"timestamp":"2025-01-15T10:02:00Z","type":"user","message":{"content":"Third"}}
`,
			timestamp: mustParseTime("2025-01-15T10:01:00Z"),
			wantErr:   false,
			checkOutput: func(t *testing.T, output []byte) {
				// First should be preserved
				if !strings.Contains(string(output), "First") {
					t.Error("expected first message to be preserved")
				}
				// Third should be preserved
				if !strings.Contains(string(output), "Third") {
					t.Error("expected third message to be preserved")
				}
				// Second should be redacted
				if strings.Contains(string(output), "Second") {
					t.Error("expected second message to be redacted")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := redactJSONLEntry([]byte(tt.content), tt.timestamp)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if tt.checkOutput != nil {
				tt.checkOutput(t, output)
			}
		})
	}
}

func TestShouldRedact(t *testing.T) {
	tests := []struct {
		name      string
		entry     map[string]interface{}
		timestamp time.Time
		want      bool
	}{
		{
			name: "exact match",
			entry: map[string]interface{}{
				"timestamp": "2025-01-15T10:00:00Z",
			},
			timestamp: mustParseTime("2025-01-15T10:00:00Z"),
			want:      true,
		},
		{
			name: "within tolerance",
			entry: map[string]interface{}{
				"timestamp": "2025-01-15T10:00:00.500Z",
			},
			timestamp: mustParseTime("2025-01-15T10:00:00Z"),
			want:      true,
		},
		{
			name: "outside tolerance",
			entry: map[string]interface{}{
				"timestamp": "2025-01-15T10:00:02Z",
			},
			timestamp: mustParseTime("2025-01-15T10:00:00Z"),
			want:      false,
		},
		{
			name: "no timestamp field",
			entry: map[string]interface{}{
				"type": "user",
			},
			timestamp: mustParseTime("2025-01-15T10:00:00Z"),
			want:      false,
		},
		{
			name: "invalid timestamp format",
			entry: map[string]interface{}{
				"timestamp": "not-a-timestamp",
			},
			timestamp: mustParseTime("2025-01-15T10:00:00Z"),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRedact(tt.entry, tt.timestamp)
			if got != tt.want {
				t.Errorf("shouldRedact() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedactEntry(t *testing.T) {
	tests := []struct {
		name  string
		entry map[string]interface{}
		check func(t *testing.T, entry map[string]interface{})
	}{
		{
			name: "redact message.content",
			entry: map[string]interface{}{
				"message": map[string]interface{}{
					"content": "secret",
				},
			},
			check: func(t *testing.T, entry map[string]interface{}) {
				msg := entry["message"].(map[string]interface{})
				if msg["content"] != "<REDACTED BY USER>" {
					t.Errorf("message.content = %v, want <REDACTED BY USER>", msg["content"])
				}
			},
		},
		{
			name: "redact direct content",
			entry: map[string]interface{}{
				"content": "secret",
			},
			check: func(t *testing.T, entry map[string]interface{}) {
				if entry["content"] != "<REDACTED BY USER>" {
					t.Errorf("content = %v, want <REDACTED BY USER>", entry["content"])
				}
			},
		},
		{
			name: "redact both message.content and content",
			entry: map[string]interface{}{
				"content": "secret1",
				"message": map[string]interface{}{
					"content": "secret2",
				},
			},
			check: func(t *testing.T, entry map[string]interface{}) {
				if entry["content"] != "<REDACTED BY USER>" {
					t.Errorf("content = %v, want <REDACTED BY USER>", entry["content"])
				}
				msg := entry["message"].(map[string]interface{})
				if msg["content"] != "<REDACTED BY USER>" {
					t.Errorf("message.content = %v, want <REDACTED BY USER>", msg["content"])
				}
			},
		},
		{
			name: "no content fields - no change",
			entry: map[string]interface{}{
				"type": "user",
			},
			check: func(t *testing.T, entry map[string]interface{}) {
				if _, ok := entry["content"]; ok {
					t.Error("should not add content field")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redactEntry(tt.entry)
			tt.check(t, tt.entry)
		})
	}
}

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// containsRedacted checks for the redacted placeholder in output.
// JSON marshaling escapes < and > as \u003c and \u003e, so we check for both.
func containsRedacted(s string) bool {
	return strings.Contains(s, "<REDACTED BY USER>") ||
		strings.Contains(s, `\u003cREDACTED BY USER\u003e`)
}
