# PII Scrubbing

Transcripts are automatically scrubbed of sensitive data before storage. This includes:
- PII pattern matching (emails, API keys, passwords, etc.)
- Tool output redaction (Read tool outputs removed by default)
- Duplicate data removal (internal fields like `toolUseResult`)

## Tool Output Redaction

By default, **Read tool outputs are redacted** with `<REDACTED>`. This saves ~8% storage and removes potentially sensitive file contents that are already available in git.

| Tool | Redacted | Reason |
|------|----------|--------|
| Read | Yes | File contents already in git, may contain secrets |
| Bash | No | Command outputs often needed for context |
| Edit | No | Shows what was changed |
| Write | No | Shows what was written |

The tool input (file path, command) is preserved - only the output is redacted.

## Storage Optimization

The `toolUseResult` field is automatically removed from session entries. This field duplicates data from `message.content` in a different format and saves ~37% storage.

## Scrubbed Data Types

| Type | Example | Replacement |
|------|---------|-------------|
| User paths | `/Users/jacek/code/` | `/<REDACTED>/` |
| Database URLs | `postgres://user:pass@host/db` | `postgres://<CREDENTIALS>@<HOST>` |
| URLs with creds | `https://user:pass@host` | `https://<CREDENTIALS>@host` |
| Emails | `john@example.com` | `<EMAIL>` |
| Credit cards | `4111111111111111` | `<CREDIT_CARD>` |
| AWS keys | `AKIAIOSFODNN7EXAMPLE` | `<AWS_ACCESS_KEY>` |
| Stripe keys | `sk_live_xxxx...` | `<STRIPE_KEY>` |
| GitHub tokens | `ghp_xxxx...` | `<GITHUB_TOKEN>` |
| Anthropic keys | `sk-ant-xxxx...` | `<ANTHROPIC_API_KEY>` |
| OpenAI keys | `sk-xxxx...` | `<OPENAI_API_KEY>` |
| OpenRouter keys | `sk-or-xxxx...` | `<OPENROUTER_API_KEY>` |
| Google AI keys | `AIzaXXXX...` | `<GOOGLE_API_KEY>` |
| Discord tokens | `MTk4NjIy...` | `<DISCORD_TOKEN>` |
| NPM tokens | `npm_xxxx...` | `<NPM_TOKEN>` |
| SendGrid keys | `SG.xxxx...` | `<SENDGRID_KEY>` |
| Twilio keys | `SKxxxx...` | `<TWILIO_KEY>` |
| Slack tokens | `xoxb-xxx-xxx` | `<SLACK_TOKEN>` |
| Bearer tokens | `Bearer eyJ...` | `Bearer <TOKEN>` |
| Cookies | `Cookie: session=abc...` | `Cookie: <COOKIE>` |
| Passwords | `password="secret"` | `<PASSWORD>` |
| Env secrets | `MYAPP_TOKEN=secret` | `MYAPP_TOKEN=<SECRET>` |
| Private keys | `-----BEGIN RSA PRIVATE KEY-----` | `<PRIVATE_KEY>` |

## Disabling Scrubbing

```bash
# For hooks (environment variable)
GIT_PROMPT_STORY_NO_SCRUB=1 git commit -m "message"

# For cloud command (flag)
git-prompt-story annotate-cloud HEAD --auto --no-scrub
```

## Adding Custom Patterns

Add a custom recognizer in `internal/scrubber/scrubber.go`:

```go
// In DefaultRecognizers(), add:
{
    Name:       "my_custom_key",
    EntityType: "MY_KEY",
    Patterns: []Pattern{
        {Regex: `my-prefix-[a-zA-Z0-9]{32}`},
    },
    Replacement: "<MY_KEY>",
},
```

Pattern order matters: specific patterns (like `sk-ant-`) must come before generic ones (like `api_key=`).

## Adding Custom Tool Redactors

To redact outputs from additional tools, add to `DefaultToolRedactors()`:

```go
{
    Name:        "bash_tool_output",
    ToolName:    "Bash",
    Replacement: "<REDACTED>",
    Comment:     "Redact Bash outputs for privacy",
},
```

## Adding Custom Node Removers

To remove additional JSON fields from session entries, add to `DefaultNodeRemovers()`:

```go
{
    Name:    "thinkingMetadata_removal",
    Path:    "thinkingMetadata",
    Comment: "Remove thinking metadata to save space",
},
```
