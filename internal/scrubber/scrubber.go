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
	recognizers []CompiledRecognizer
}

// New creates a new PIIScrubber with the given recognizers
func New(recognizers []Recognizer) (*PIIScrubber, error) {
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

	return &PIIScrubber{recognizers: compiled}, nil
}

// NewDefault creates a PIIScrubber with built-in patterns
func NewDefault() (*PIIScrubber, error) {
	return New(DefaultRecognizers())
}

// Scrub implements the Scrubber interface for JSONL content
func (s *PIIScrubber) Scrub(content []byte) ([]byte, error) {
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

		// Scrub JSON values recursively
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
			Replacement: "/Users/<REDACTED>/",
		},
		{
			Name:       "windows_user_path",
			EntityType: "USER_PATH",
			Patterns: []Pattern{
				{Regex: `C:\\Users\\[a-zA-Z0-9._-]+\\`},
			},
			Replacement: `C:\Users\<REDACTED>\`,
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

		// Private keys
		{
			Name:       "private_key",
			EntityType: "PRIVATE_KEY",
			Patterns: []Pattern{
				{Regex: `-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`},
			},
			Replacement: "<PRIVATE_KEY_HEADER>",
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
