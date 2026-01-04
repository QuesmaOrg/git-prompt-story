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
		{"/Users/jacek/projects/myapp", "/<REDACTED>/projects/myapp"},
		{"/home/ubuntu/code/test.py", "/<REDACTED>/code/test.py"},
		{"File at /Users/john.doe/Documents/secret.txt", "File at /<REDACTED>/Documents/secret.txt"},
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

func TestScrubOpenRouterKey(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"OPENROUTER_API_KEY=sk-or-v1-abcdefghijklmnopqrstuvwxyz1234567890abcd", "OPENROUTER_API_KEY=<OPENROUTER_API_KEY>"},
		{"key: sk-or-abcdefghijklmnopqrstuvwxyz1234567890abcdef", "key: <OPENROUTER_API_KEY>"},
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestScrubGoogleKey(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"GOOGLE_API_KEY=AIzaSyA1234567890abcdefghijklmnopqrstuv", "GOOGLE_API_KEY=<GOOGLE_API_KEY>"},
		{"api_key: AIzaXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX", "api_key: <GOOGLE_API_KEY>"},
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
		{"-----BEGIN RSA PRIVATE KEY-----\nMIIEpQIBAAKCAQEA3Tz2mr7SZiAMfQyuvBjM9Oi...\n-----END RSA PRIVATE KEY-----", "<PRIVATE_KEY>"},
		{"-----BEGIN PRIVATE KEY-----\nMIIBVgIBADANBgkqhkiG9w0BAQEFAASCNWAwgg...\n-----END PRIVATE KEY-----", "<PRIVATE_KEY>"},
		{"-----BEGIN PRIVATE KEY-----\nMIIBVgIBADANBgkqhkiG9w0BAQEFAASCNWAwgg...\n", "<PRIVATE_KEY>"},
		{"-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW\n-----END OPENSSH PRIVATE KEY-----", "<PRIVATE_KEY>"},
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
	if !strings.Contains(string(result), "/<REDACTED>/") && !strings.Contains(string(result), "/\\u003cREDACTED\\u003e/") {
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

func TestGenericEnvironmentVariable(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"MYAPP_TOKEN=1234567890-1234567890123-abcdefghijklmnop", "MYAPP_TOKEN=<SECRET>"},
		{"MYAPP_SECRET=51234567890-1234567890123-abcdefghijklmnop", "MYAPP_SECRET=<SECRET>"},
		{"MYAPP_API_KEY=71234567890-1234567890123-abcdefghijklmnop", "MYAPP_API_KEY=<SECRET>"},
		{"MYAPP_TOKEN=\"1234567890-1234567890123-abcdefghijklmnop\"", "MYAPP_TOKEN=<SECRET>"},
		{"MYAPP_SECRET=\"51234567890-1234567890123-abcdefghijklmnop\"", "MYAPP_SECRET=<SECRET>"},
		{"MYAPP_API_KEY=\"71234567890-1234567890123-abcdefghijklmnop\"", "MYAPP_API_KEY=<SECRET>"},
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestScrubDatabaseURL(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"postgres://user:password@localhost:5432/mydb", "postgres://<CREDENTIALS>@<HOST>"},
		{"postgresql://admin:secret123@db.example.com/prod", "postgresql://<CREDENTIALS>@<HOST>"},
		{"mysql://root:pass@127.0.0.1:3306/app", "mysql://<CREDENTIALS>@<HOST>"},
		{"mongodb://user:pass@cluster.mongodb.net/db", "mongodb://<CREDENTIALS>@<HOST>"},
		{"mongodb+srv://user:pass@cluster.mongodb.net/db", "mongodb+srv://<CREDENTIALS>@<HOST>"},
		{"redis://:secretpass@redis.example.com:6379", "redis://<CREDENTIALS>@<HOST>"},
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestScrubURLCredentials(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"https://user:pass@example.com/path", "https://<CREDENTIALS>@example.com/path"},
		{"http://admin:secret@internal.server.local:8080", "http://<CREDENTIALS>@internal.server.local:8080"},
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestScrubStripeKey(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"STRIPE_KEY=sk_live_abcdefghijklmnopqrstuvwx", "STRIPE_KEY=<STRIPE_KEY>"},
		{"pk_test_1234567890abcdefghijklmn", "<STRIPE_KEY>"},
		{"sk_test_ABCDEFGHIJKLMNOPQRSTUVWXYZab", "<STRIPE_KEY>"},
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestScrubDiscordToken(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"MTk4NjIyNDgzNDcxOTI1MjQ4.Cl2FMQ.ZnCjm1XVW7vRze4b7Cq4se7kKWs", "<DISCORD_TOKEN>"},
		{"NjE2MTk0NTI2NDMwODI3NTMx.XVtXKg.abcdefghijklmnopqrstuvwxyz1", "<DISCORD_TOKEN>"},
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestScrubNPMToken(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"npm_abcdefghijklmnopqrstuvwxyz1234567890", "<NPM_TOKEN>"},
		{"NPM_TOKEN=npm_ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890", "NPM_TOKEN=<NPM_TOKEN>"},
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestScrubSendGridKey(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"SG.abcdefghijklmnopqrstuv.ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopq", "<SENDGRID_KEY>"},
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestScrubTwilioKey(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"TWILIO_API_KEY=SK1234567890abcdef1234567890abcdef", "TWILIO_API_KEY=<TWILIO_KEY>"},
		{"SK0123456789abcdef0123456789abcdef", "<TWILIO_KEY>"},
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestScrubCookie(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"Cookie: session_id=1234567890; user=john.doe", "Cookie: <COOKIE>"},
		{"Set-Cookie: session=abcdef; Path=/", "Set-Cookie: <COOKIE>"},
		{"cookie: token=secret", "cookie: <COOKIE>"},
	}

	for _, tc := range tests {
		result := s.ScrubText(tc.input)
		if result != tc.expected {
			t.Errorf("ScrubText(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestRemoveToolUseResult(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	// JSONL with toolUseResult field (should be removed)
	input := `{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_123","content":"file contents"}]},"toolUseResult":{"type":"text","file":{"filePath":"/path/to/file.go","content":"package main"}}}`

	result, err := s.Scrub([]byte(input))
	if err != nil {
		t.Fatalf("Scrub() error: %v", err)
	}

	// Verify toolUseResult was removed
	if strings.Contains(string(result), "toolUseResult") {
		t.Error("toolUseResult field was not removed")
	}

	// Verify the entry is still valid JSON
	var obj map[string]interface{}
	if err := json.Unmarshal(result, &obj); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	// Verify other fields are preserved
	if obj["type"] != "user" {
		t.Error("type field was not preserved")
	}
}

func TestRedactReadToolOutput(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	// Two JSONL lines: assistant with tool_use (Read), then user with tool_result
	input := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_read_123","name":"Read","input":{"file_path":"/path/to/file.go"}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_read_123","content":"package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}"}]}}`

	result, err := s.Scrub([]byte(input))
	if err != nil {
		t.Fatalf("Scrub() error: %v", err)
	}

	// Verify the file content was redacted
	if strings.Contains(string(result), "package main") {
		t.Error("Read tool output was not redacted")
	}
	if strings.Contains(string(result), "fmt.Println") {
		t.Error("Read tool output was not redacted")
	}

	// Verify REDACTED placeholder is present (may be unicode-escaped)
	if !strings.Contains(string(result), "<REDACTED>") && !strings.Contains(string(result), "\\u003cREDACTED\\u003e") {
		t.Error("REDACTED placeholder not found")
	}

	// Verify each line is still valid JSON
	lines := strings.Split(string(result), "\n")
	for i, line := range lines {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i, err)
		}
	}
}

func TestNonReadToolOutputNotRedacted(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	// Two JSONL lines: assistant with tool_use (Bash, not Read), then user with tool_result
	input := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_bash_123","name":"Bash","input":{"command":"ls -la"}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_bash_123","content":"total 8\ndrwxr-xr-x  2 user group 4096 Jan 1 12:00 .\ndrwxr-xr-x 10 user group 4096 Jan 1 12:00 .."}]}}`

	result, err := s.Scrub([]byte(input))
	if err != nil {
		t.Fatalf("Scrub() error: %v", err)
	}

	// Verify the Bash output was NOT redacted (Bash is not in the default redactors)
	if !strings.Contains(string(result), "drwxr-xr-x") {
		t.Error("Bash tool output was incorrectly redacted")
	}

	// Verify REDACTED placeholder is NOT present
	if strings.Contains(string(result), "<REDACTED>") || strings.Contains(string(result), "\\u003cREDACTED\\u003e") {
		t.Error("REDACTED placeholder found for non-Read tool")
	}
}

func TestMultipleReadToolOutputsRedacted(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	// Multiple Read tool uses and results
	input := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_1","name":"Read","input":{"file_path":"/file1.go"}},{"type":"tool_use","id":"toolu_2","name":"Read","input":{"file_path":"/file2.go"}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"file1 content"},{"type":"tool_result","tool_use_id":"toolu_2","content":"file2 content"}]}}`

	result, err := s.Scrub([]byte(input))
	if err != nil {
		t.Fatalf("Scrub() error: %v", err)
	}

	// Verify both contents were redacted
	if strings.Contains(string(result), "file1 content") {
		t.Error("First Read tool output was not redacted")
	}
	if strings.Contains(string(result), "file2 content") {
		t.Error("Second Read tool output was not redacted")
	}
}

func TestMixedToolOutputs(t *testing.T) {
	s, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault() error: %v", err)
	}

	// Read should be redacted, Bash should not
	input := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_read","name":"Read","input":{"file_path":"/file.go"}},{"type":"tool_use","id":"toolu_bash","name":"Bash","input":{"command":"echo hello"}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_read","content":"SECRET FILE CONTENT"},{"type":"tool_result","tool_use_id":"toolu_bash","content":"hello"}]}}`

	result, err := s.Scrub([]byte(input))
	if err != nil {
		t.Fatalf("Scrub() error: %v", err)
	}

	// Read content should be redacted
	if strings.Contains(string(result), "SECRET FILE CONTENT") {
		t.Error("Read tool output was not redacted")
	}

	// Bash content should be preserved
	if !strings.Contains(string(result), "hello") {
		t.Error("Bash tool output was incorrectly redacted")
	}
}
