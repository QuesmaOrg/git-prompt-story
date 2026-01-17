package scrubber

import (
	"bufio"
	"bytes"
	"encoding/json"
	"regexp"
)

// Scrubber is the interface for PII scrubbing implementations
type Scrubber interface {
	// Scrub processes content and replaces PII with placeholders
	Scrub(content []byte) ([]byte, error)
}

// Recognizer defines a PII pattern recognizer
type Recognizer struct {
	Name        string   `yaml:"name"`
	EntityType  string   `yaml:"entity_type"`
	Patterns    []Pattern `yaml:"patterns"`
	Replacement string   `yaml:"replacement"`
}

// Pattern defines a single regex pattern
type Pattern struct {
	Regex string `yaml:"regex"`
}

// ToolOutputRedactor configures redaction of specific tool outputs
// This allows redacting the output content of specific Claude Code tools
type ToolOutputRedactor struct {
	Name        string `yaml:"name"`
	ToolName    string `yaml:"tool_name"`   // Tool to redact (e.g., "Read", "Bash")
	Replacement string `yaml:"replacement"` // Replacement text (e.g., "<REDACTED>")
	Comment     string `yaml:"comment"`     // Explanation of why this redaction exists
}

// NodeRemover configures removal of entire JSON nodes from session entries
// This allows removing duplicate or unnecessary fields to save space
type NodeRemover struct {
	Name         string               `yaml:"name"`
	Path         string               `yaml:"path"`    // JSON field path to remove (e.g., "toolUseResult")
	ShouldRemove func(value any) bool `yaml:"-"`       // Predicate: return true to remove (nil = always remove)
	Comment      string               `yaml:"comment"` // Explanation of why this removal exists
}

// CompiledRecognizer is a recognizer with compiled regex patterns
type CompiledRecognizer struct {
	Name        string
	EntityType  string
	Patterns    []*regexp.Regexp
	Replacement string
}

// Config holds scrubber configuration
type Config struct {
	Enabled           bool
	CustomPatternFile string
}

// PIIScrubber implements the Scrubber interface
type PIIScrubber struct {
	recognizers   []CompiledRecognizer
	toolRedactors []ToolOutputRedactor
	nodeRemovers  []NodeRemover
}

// New creates a new PIIScrubber with the given recognizers, tool redactors, and node removers
func New(recognizers []Recognizer, toolRedactors []ToolOutputRedactor, nodeRemovers []NodeRemover) (*PIIScrubber, error) {
	compiled := make([]CompiledRecognizer, 0, len(recognizers))

	for _, r := range recognizers {
		cr := CompiledRecognizer{
			Name:        r.Name,
			EntityType:  r.EntityType,
			Replacement: r.Replacement,
			Patterns:    make([]*regexp.Regexp, 0, len(r.Patterns)),
		}

		for _, p := range r.Patterns {
			re, err := regexp.Compile(p.Regex)
			if err != nil {
				return nil, err
			}
			cr.Patterns = append(cr.Patterns, re)
		}

		compiled = append(compiled, cr)
	}

	return &PIIScrubber{
		recognizers:   compiled,
		toolRedactors: toolRedactors,
		nodeRemovers:  nodeRemovers,
	}, nil
}

// NewDefault creates a PIIScrubber with built-in patterns
func NewDefault() (*PIIScrubber, error) {
	return New(DefaultRecognizers(), DefaultToolRedactors(), DefaultNodeRemovers())
}

// Scrub implements the Scrubber interface for JSONL content
func (s *PIIScrubber) Scrub(content []byte) ([]byte, error) {
	// First pass: build set of tool_use IDs to redact
	toolRedactSet := s.buildToolRedactSet(content)

	// Second pass: process and scrub content
	var result bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(content))

	// Increase buffer for large lines (Claude responses can be big)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	first := true
	for scanner.Scan() {
		line := scanner.Bytes()

		if !first {
			result.WriteByte('\n')
		}
		first = false

		// Try to parse as JSON and scrub recursively
		var obj map[string]interface{}
		if err := json.Unmarshal(line, &obj); err != nil {
			// Not valid JSON, scrub as plain text
			scrubbed := s.scrubText(string(line))
			result.WriteString(scrubbed)
			continue
		}

		// 1. Remove configured nodes (e.g., toolUseResult)
		s.removeNodes(obj)

		// 2. Redact configured tool outputs (e.g., Read tool)
		s.redactToolResults(obj, toolRedactSet)

		// 3. Apply PII patterns recursively
		s.scrubValue(obj)

		// Re-serialize
		scrubbed, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}
		result.Write(scrubbed)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}

// scrubText applies all recognizers to a plain text string
func (s *PIIScrubber) scrubText(text string) string {
	result := text
	for _, r := range s.recognizers {
		for _, pattern := range r.Patterns {
			result = pattern.ReplaceAllString(result, r.Replacement)
		}
	}
	return result
}

// scrubValue recursively scrubs JSON values
func (s *PIIScrubber) scrubValue(v interface{}) {
	switch val := v.(type) {
	case map[string]interface{}:
		for k, inner := range val {
			if str, ok := inner.(string); ok {
				val[k] = s.scrubText(str)
			} else {
				s.scrubValue(inner)
			}
		}
	case []interface{}:
		for i, inner := range val {
			if str, ok := inner.(string); ok {
				val[i] = s.scrubText(str)
			} else {
				s.scrubValue(inner)
			}
		}
	}
}

// removeNodes removes configured JSON fields from the object
func (s *PIIScrubber) removeNodes(obj map[string]interface{}) {
	for _, nr := range s.nodeRemovers {
		if value, exists := obj[nr.Path]; exists {
			// If no predicate, always remove; otherwise check predicate
			if nr.ShouldRemove == nil || nr.ShouldRemove(value) {
				delete(obj, nr.Path)
			}
		}
	}
}

// buildToolRedactSet scans JSONL content and returns a set of tool_use IDs
// that should have their outputs redacted
func (s *PIIScrubber) buildToolRedactSet(content []byte) map[string]string {
	redactSet := make(map[string]string) // tool_use_id -> replacement

	if len(s.toolRedactors) == 0 {
		return redactSet
	}

	// Build map of tool names to redact
	toolsToRedact := make(map[string]string) // tool_name -> replacement
	for _, tr := range s.toolRedactors {
		toolsToRedact[tr.ToolName] = tr.Replacement
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		var obj map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &obj); err != nil {
			continue
		}

		// Look for assistant messages with tool_use content
		if obj["type"] != "assistant" {
			continue
		}

		msg, ok := obj["message"].(map[string]interface{})
		if !ok {
			continue
		}

		content, ok := msg["content"].([]interface{})
		if !ok {
			continue
		}

		for _, part := range content {
			partMap, ok := part.(map[string]interface{})
			if !ok {
				continue
			}

			if partMap["type"] != "tool_use" {
				continue
			}

			toolName, _ := partMap["name"].(string)
			toolID, _ := partMap["id"].(string)

			if replacement, shouldRedact := toolsToRedact[toolName]; shouldRedact && toolID != "" {
				redactSet[toolID] = replacement
			}
		}
	}

	return redactSet
}

// redactToolResults redacts tool_result content for IDs in the redact set
func (s *PIIScrubber) redactToolResults(obj map[string]interface{}, redactSet map[string]string) {
	if len(redactSet) == 0 {
		return
	}

	// Only process user messages (tool results come in user messages)
	if obj["type"] != "user" {
		return
	}

	msg, ok := obj["message"].(map[string]interface{})
	if !ok {
		return
	}

	content, ok := msg["content"].([]interface{})
	if !ok {
		return
	}

	for _, part := range content {
		partMap, ok := part.(map[string]interface{})
		if !ok {
			continue
		}

		if partMap["type"] != "tool_result" {
			continue
		}

		toolUseID, _ := partMap["tool_use_id"].(string)
		if replacement, shouldRedact := redactSet[toolUseID]; shouldRedact {
			partMap["content"] = replacement
		}
	}
}

// ScrubText is a convenience method to scrub plain text
func (s *PIIScrubber) ScrubText(text string) string {
	return s.scrubText(text)
}

// NoopScrubber is a scrubber that does nothing (pass-through)
type NoopScrubber struct{}

// Scrub returns content unchanged
func (n *NoopScrubber) Scrub(content []byte) ([]byte, error) {
	return content, nil
}

// DefaultRecognizers returns the built-in PII recognizers
func DefaultRecognizers() []Recognizer {
	return []Recognizer{
		// File paths with usernames (match username + following slash, replace with redacted + slash)
		{
			Name:       "unix_home_path",
			EntityType: "USER_PATH",
			Patterns: []Pattern{
				{Regex: `/(Users|home)/[a-zA-Z0-9._-]+/`},
			},
			Replacement: "/<REDACTED>/",
		},
		{
			Name:       "windows_user_path",
			EntityType: "USER_PATH",
			Patterns: []Pattern{
				{Regex: `C:\\Users\\[a-zA-Z0-9._-]+\\`},
			},
			Replacement: `C:\Users\<REDACTED>\`,
		},

		// Database connection URLs (must come BEFORE email to avoid partial matches)
		{
			Name:       "database_url",
			EntityType: "DATABASE_URL",
			Patterns: []Pattern{
				// PostgreSQL, MySQL, MongoDB, Redis with credentials (allow empty username for redis)
				{Regex: `(?i)((?:postgres(?:ql)?|mysql|mongodb(?:\+srv)?|redis|mariadb)://)[^:]*:[^@]+@[^\s'"]+`},
			},
			Replacement: "${1}<CREDENTIALS>@<HOST>",
		},

		// URLs with embedded credentials (must come BEFORE email)
		{
			Name:       "url_credentials",
			EntityType: "URL_CREDENTIALS",
			Patterns: []Pattern{
				{Regex: `(https?://)[^:]+:[^@]+@([^\s'"]+)`},
			},
			Replacement: "${1}<CREDENTIALS>@${2}",
		},

		// Email addresses
		{
			Name:       "email",
			EntityType: "EMAIL",
			Patterns: []Pattern{
				{Regex: `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`},
			},
			Replacement: "<EMAIL>",
		},

		// Credit cards (major card patterns)
		{
			Name:       "credit_card",
			EntityType: "CREDIT_CARD",
			Patterns: []Pattern{
				{Regex: `\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|6(?:011|5[0-9]{2})[0-9]{12})\b`},
			},
			Replacement: "<CREDIT_CARD>",
		},

		// AWS credentials
		{
			Name:       "aws_access_key",
			EntityType: "AWS_KEY",
			Patterns: []Pattern{
				{Regex: `AKIA[0-9A-Z]{16}`},
			},
			Replacement: "<AWS_ACCESS_KEY>",
		},
		{
			Name:       "aws_secret_key",
			EntityType: "AWS_SECRET",
			Patterns: []Pattern{
				{Regex: `(?i)aws.{0,20}secret.{0,20}['"][A-Za-z0-9/+=]{40}['"]`},
			},
			Replacement: "<AWS_SECRET_KEY>",
		},

		// Specific API keys (must come before generic patterns)
		// Stripe API keys
		{
			Name:       "stripe_api_key",
			EntityType: "STRIPE_KEY",
			Patterns: []Pattern{
				{Regex: `(?:sk|pk)_(?:live|test)_[a-zA-Z0-9]{24,}`},
			},
			Replacement: "<STRIPE_KEY>",
		},

		// Anthropic API keys
		{
			Name:       "anthropic_api_key",
			EntityType: "ANTHROPIC_KEY",
			Patterns: []Pattern{
				{Regex: `sk-ant-[a-zA-Z0-9_-]{40,}`},
			},
			Replacement: "<ANTHROPIC_API_KEY>",
		},

		// OpenAI API keys
		{
			Name:       "openai_api_key",
			EntityType: "OPENAI_KEY",
			Patterns: []Pattern{
				{Regex: `sk-[a-zA-Z0-9]{48}`},
			},
			Replacement: "<OPENAI_API_KEY>",
		},

		// OpenRouter API keys
		{
			Name:       "openrouter_api_key",
			EntityType: "OPENROUTER_KEY",
			Patterns: []Pattern{
				{Regex: `sk-or-[a-zA-Z0-9_-]{40,}`},
			},
			Replacement: "<OPENROUTER_API_KEY>",
		},

		// Google AI / Gemini API keys
		{
			Name:       "google_api_key",
			EntityType: "GOOGLE_KEY",
			Patterns: []Pattern{
				{Regex: `AIza[0-9A-Za-z_-]{35}`},
			},
			Replacement: "<GOOGLE_API_KEY>",
		},

		// Discord bot tokens (base64-ish, typically 59+ chars with dots)
		{
			Name:       "discord_token",
			EntityType: "DISCORD_TOKEN",
			Patterns: []Pattern{
				{Regex: `[MN][A-Za-z0-9]{23,}\.[A-Za-z0-9_-]{6}\.[A-Za-z0-9_-]{27,}`},
			},
			Replacement: "<DISCORD_TOKEN>",
		},

		// NPM tokens
		{
			Name:       "npm_token",
			EntityType: "NPM_TOKEN",
			Patterns: []Pattern{
				{Regex: `npm_[A-Za-z0-9]{36,}`},
			},
			Replacement: "<NPM_TOKEN>",
		},

		// SendGrid API keys
		{
			Name:       "sendgrid_api_key",
			EntityType: "SENDGRID_KEY",
			Patterns: []Pattern{
				{Regex: `SG\.[A-Za-z0-9_-]{22}\.[A-Za-z0-9_-]{43}`},
			},
			Replacement: "<SENDGRID_KEY>",
		},

		// Twilio API keys
		{
			Name:       "twilio_api_key",
			EntityType: "TWILIO_KEY",
			Patterns: []Pattern{
				{Regex: `SK[a-f0-9]{32}`},
			},
			Replacement: "<TWILIO_KEY>",
		},

		// GitHub tokens
		{
			Name:       "github_token",
			EntityType: "GITHUB_TOKEN",
			Patterns: []Pattern{
				{Regex: `gh[pousr]_[A-Za-z0-9_]{36,}`},
			},
			Replacement: "<GITHUB_TOKEN>",
		},

		// Slack tokens
		{
			Name:       "slack_token",
			EntityType: "SLACK_TOKEN",
			Patterns: []Pattern{
				{Regex: `xox[baprs]-[0-9]+-[0-9]+-[a-zA-Z0-9]+`},
			},
			Replacement: "<SLACK_TOKEN>",
		},

		// Bearer tokens (JWT format)
		{
			Name:       "bearer_token",
			EntityType: "AUTH_TOKEN",
			Patterns: []Pattern{
				{Regex: `(?i)bearer\s+[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+`},
			},
			Replacement: "Bearer <TOKEN>",
		},

		// Cookies
		{
			Name:       "cookie",
			EntityType: "COOKIE",
			Patterns: []Pattern{
				{Regex: `(?i)((?:set-)?cookie:\s*)[^\r\n]+`},
			},
			Replacement: "${1}<COOKIE>",
		},

		// Private keys
		{
			Name:       "private_key",
			EntityType: "PRIVATE_KEY",
			Patterns: []Pattern{
				{Regex: `(?s)-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----.*?(?:-----END (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----|$)`},
			},
			Replacement: "<PRIVATE_KEY>",
		},

		// Environment variables with secrets (e.g. MYAPP_TOKEN=...)
		{
			Name:       "env_secret",
			EntityType: "SECRET",
			Patterns: []Pattern{
				{Regex: `(?i)([A-Z0-9_]*(?:TOKEN|SECRET|API_KEY)[A-Z0-9_]*=)(?:["']?[a-zA-Z0-9_.\-]+["']?)`},
			},
			Replacement: "${1}<SECRET>",
		},

		// Generic API keys/tokens (after specific ones)
		{
			Name:       "generic_api_key",
			EntityType: "API_KEY",
			Patterns: []Pattern{
				// Match api_key=value but NOT as part of larger word like ANTHROPIC_API_KEY
				{Regex: `(?i)(?:^|[^A-Z_])(?:api[_-]?key|api[_-]?secret|access[_-]?token|auth[_-]?token)['":\s=]+[a-zA-Z0-9_-]{20,}`},
			},
			Replacement: "<API_KEY>",
		},

		// Passwords in config/env
		{
			Name:       "password_assignment",
			EntityType: "PASSWORD",
			Patterns: []Pattern{
				// Match password="value" or password: value, capturing the whole thing
				{Regex: `(?i)(?:password|passwd|pwd)['":\s=]+[^\s'"]{8,}`},
			},
			Replacement: "<PASSWORD>",
		},
	}
}

// DefaultToolRedactors returns the built-in tool output redactors
func DefaultToolRedactors() []ToolOutputRedactor {
	return []ToolOutputRedactor{
		{
			Name:        "read_tool_output",
			ToolName:    "Read",
			Replacement: "<REDACTED>",
			// Read tool outputs contain full file contents which are already
			// available in git. Redacting saves ~8% storage and removes
			// potentially sensitive data that shouldn't be stored in transcripts.
			Comment: "Read tool outputs contain full file contents. Redacting saves ~8% storage and removes potentially sensitive data.",
		},
	}
}

// DefaultNodeRemovers returns the built-in node removers
func DefaultNodeRemovers() []NodeRemover {
	return []NodeRemover{
		{
			Name: "toolUseResult_duplicate",
			Path: "toolUseResult",
			// toolUseResult is a Claude Code-specific field that duplicates data
			// from message.content in a different format. However, for AskUserQuestion
			// responses, toolUseResult.answers contains structured decision data
			// that is used by pr summary to display DECISION entries.
			ShouldRemove: func(value any) bool {
				// Preserve if it contains AskUserQuestion answers (decision data)
				if m, ok := value.(map[string]interface{}); ok {
					if _, hasAnswers := m["answers"]; hasAnswers {
						return false // Don't remove - has decision data
					}
				}
				return true // Remove - just duplicate data
			},
			Comment: "toolUseResult duplicates message.content data, except for AskUserQuestion responses which contain decision answers.",
		},
	}
}

// Ensure PIIScrubber implements Scrubber
var _ Scrubber = (*PIIScrubber)(nil)
var _ Scrubber = (*NoopScrubber)(nil)

// scrubText helper for use in tests - unexported version uses receiver
func init() {
	// Ensure patterns compile at init time to catch errors early
	_, err := NewDefault()
	if err != nil {
		panic("invalid default pattern: " + err.Error())
	}
}
