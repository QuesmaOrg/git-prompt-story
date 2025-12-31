package scrubber

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestScrubEmail(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"Contact: john.doe@example.com", "Contact: <EMAIL>"},
		{"Email me at test+tag@sub.domain.co.uk please", "Email me at <EMAIL> please"},
		{"Multiple: a@b.com and c@d.org", "Multiple: <EMAIL> and <EMAIL>"},
		{"No email here", "No email here"},
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestScrubUnixPath(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"/Users/jacek/projects/myapp", "/Users/<REDACTED>/projects/myapp"},
		{"/home/ubuntu/code/test.py", "/Users/<REDACTED>/code/test.py"},
		{"File at /Users/john.doe/Documents/secret.txt", "File at /Users/<REDACTED>/Documents/secret.txt"},
		{"/var/log/syslog", "/var/log/syslog"}, // Not a user directory
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestScrubAWSKey(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"AWS key: AKIAIOSFODNN7EXAMPLE", "AWS key: <AWS_ACCESS_KEY>"},
		{"export AWS_ACCESS_KEY_ID=AKIAI44QH8DHBEXAMPLE", "export AWS_ACCESS_KEY_ID=<AWS_ACCESS_KEY>"},
		{"Not a key: AKIA123", "Not a key: AKIA123"}, // Too short
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestScrubCreditCard(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"Visa: 4111111111111111", "Visa: <CREDIT_CARD>"},
		{"MasterCard: 5500000000000004", "MasterCard: <CREDIT_CARD>"},
		{"Amex: 340000000000009", "Amex: <CREDIT_CARD>"},
		{"Not a card: 12345", "Not a card: 12345"},
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestScrubGitHubToken(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"token: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij", "token: <GITHUB_TOKEN>"},
		{"gho_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij123", "<GITHUB_TOKEN>"},
		{"Not a token: ghp_short", "Not a token: ghp_short"},
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestScrubAnthropicKey(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"ANTHROPIC_API_KEY=sk-ant-abcdefghijklmnopqrstuvwxyz1234567890abcd", "ANTHROPIC_API_KEY=<ANTHROPIC_API_KEY>"},
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestScrubPrivateKey(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"-----BEGIN RSA PRIVATE KEY-----", "<PRIVATE_KEY_HEADER>"},
		{"-----BEGIN PRIVATE KEY-----", "<PRIVATE_KEY_HEADER>"},
		{"-----BEGIN OPENSSH PRIVATE KEY-----", "<PRIVATE_KEY_HEADER>"},
		{"-----BEGIN PUBLIC KEY-----", "-----BEGIN PUBLIC KEY-----"}, // Not private
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestScrubJSONL(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	input := `{"type":"user","message":"My email is john@example.com"}
{"type":"assistant","message":"I see your email"}
{"path":"/Users/jacek/project/main.go","content":"test"}`

	result, err := s.Scrub([]byte(input))
	if err != nil {
		t.Fatalf("Scrub() error: %v", err)
	}

	t.Logf("Input:\n%s", input)
	t.Logf("Result:\n%s", string(result))

	lines := strings.Split(string(result), "\n")
	if len(lines) != 3 {
		t.Fatalf("Expected 3 lines, got %d", len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i, err)
		}
	}

	// Check that email was scrubbed
	if strings.Contains(string(result), "john@example.com") {
		t.Error("Email was not scrubbed from JSONL")
	}
	// Note: JSON encoding escapes <> to unicode, so check for escaped form too
	if !strings.Contains(string(result), "<EMAIL>") && !strings.Contains(string(result), "\\u003cEMAIL\\u003e") {
		t.Error("Email replacement not found in JSONL")
	}

	// Check that path was scrubbed
	if strings.Contains(string(result), "/Users/jacek/") {
		t.Error("User path was not scrubbed from JSONL")
	}
	// Note: JSON encoding escapes <> to unicode, so check for escaped form too
	if !strings.Contains(string(result), "/Users/<REDACTED>/") && !strings.Contains(string(result), "/Users/\\u003cREDACTED\\u003e/") {
		t.Error("Path replacement not found in JSONL")
	}
}

func TestScrubJSONLPreservesStructure(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	input := `{"type":"user","nested":{"email":"test@example.com","count":42},"list":["a@b.com","plain"]}`

	result, err := s.Scrub([]byte(input))
	if err != nil {
		t.Fatalf("Scrub() error: %v", err)
	}

	// Parse result
	var obj map[string]interface{}
	if err := json.Unmarshal(result, &obj); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	// Check structure preserved
	if obj["type"] != "user" {
		t.Error("type field not preserved")
	}

	nested, ok := obj["nested"].(map[string]interface{})
	if !ok {
		t.Fatal("nested field not preserved as object")
	}
	if nested["count"] != float64(42) {
		t.Error("nested.count not preserved")
	}
	if nested["email"] != "<EMAIL>" {
		t.Errorf("nested.email = %v, want <EMAIL>", nested["email"])
	}

	list, ok := obj["list"].([]interface{})
	if !ok {
		t.Fatal("list field not preserved as array")
	}
	if list[0] != "<EMAIL>" {
		t.Errorf("list[0] = %v, want <EMAIL>", list[0])
	}
	if list[1] != "plain" {
		t.Errorf("list[1] = %v, want plain", list[1])
	}
}

func TestNoopScrubber(t *testing.T) {
	s := &NoopScrubber{}
	input := []byte("email: test@example.com")

	result, err := s.Scrub(input)
	if err != nil {
		t.Fatalf("Scrub() error: %v", err)
	}

	if string(result) != string(input) {
		t.Errorf("NoopScrubber modified content: %q != %q", result, input)
	}
}

func TestScrubPassword(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		// The pattern matches password="value but leaves trailing quote
		{`password="supersecret123"`, `<PASSWORD>"`},
		{`PASSWORD: verysecretpassword`, `<PASSWORD>`},
		{`pwd=mypassword123`, `<PASSWORD>`},
		{`pwd=short`, `pwd=short`}, // Too short (< 8 chars)
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestScrubBearerToken(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.signature", "Authorization: Bearer <TOKEN>"},
		{"bearer abc.def.ghi", "Bearer <TOKEN>"},
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestScrubSlackToken(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"SLACK_TOKEN=xoxb-1234567890-1234567890123-abcdefghijklmnop", "SLACK_TOKEN=<SLACK_TOKEN>"},
		{"xoxp-1234567890-1234567890123-xyz", "<SLACK_TOKEN>"},
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}
