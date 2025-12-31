# PII Scrubbing

Transcripts are automatically scrubbed of sensitive data before storage.

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
