package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/QuesmaOrg/git-prompt-story/internal/ci"
	"github.com/spf13/cobra"
)

var (
	prPreviewNoOpen bool
)

var prPreviewCmd = &cobra.Command{
	Use:   "preview [commit-range]",
	Short: "Preview summary as rendered GitHub markdown",
	Long: `Generate an HTML preview of how the summary will look in GitHub Actions.

Opens the preview in your default browser. Use --no-open to just generate the file.

If no commit range is specified, defaults to showing the last commit (HEAD~1..HEAD).

Examples:
  git-prompt-story pr preview
  git-prompt-story pr preview HEAD~3..HEAD
  git-prompt-story pr preview main..feature --no-open`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		commitRange := "HEAD~1..HEAD"
		if len(args) > 0 {
			commitRange = args[0]
		}

		summary, err := ci.GenerateSummary(commitRange, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: %v\n", err)
			os.Exit(1)
		}

		markdown := ci.RenderMarkdown(summary, "", GetVersion())

		html := prWrapMarkdownAsGitHubHTML(markdown)

		// Write to temp file
		tmpDir := os.TempDir()
		htmlPath := filepath.Join(tmpDir, "git-prompt-story-preview.html")
		if err := os.WriteFile(htmlPath, []byte(html), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "git-prompt-story: failed to write preview: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Preview generated: %s\n", htmlPath)

		if !prPreviewNoOpen {
			if err := prOpenBrowser(htmlPath); err != nil {
				fmt.Fprintf(os.Stderr, "git-prompt-story: failed to open browser: %v\n", err)
				fmt.Fprintf(os.Stderr, "Open manually: file://%s\n", htmlPath)
			}
		}
	},
}

func init() {
	prPreviewCmd.Flags().BoolVar(&prPreviewNoOpen, "no-open", false, "Don't open browser, just generate the file")
	prCmd.AddCommand(prPreviewCmd)
}

// prOpenBrowser opens the specified URL in the default browser
func prOpenBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// prWrapMarkdownAsGitHubHTML wraps markdown content in HTML that renders it like GitHub
func prWrapMarkdownAsGitHubHTML(markdown string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Git Prompt Story - PR Preview</title>
  <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/github-markdown-css/5.5.1/github-markdown.min.css">
  <script src="https://cdnjs.cloudflare.com/ajax/libs/marked/12.0.0/marked.min.js"></script>
  <style>
    body {
      box-sizing: border-box;
      min-width: 200px;
      max-width: 980px;
      margin: 0 auto;
      padding: 45px;
      background: #0d1117;
    }
    .markdown-body {
      background: #161b22;
      border: 1px solid #30363d;
      border-radius: 6px;
      padding: 32px;
    }
    @media (max-width: 767px) {
      body {
        padding: 15px;
      }
    }
    .preview-header {
      color: #8b949e;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans", Helvetica, Arial, sans-serif;
      font-size: 14px;
      margin-bottom: 16px;
      padding-bottom: 16px;
      border-bottom: 1px solid #30363d;
    }
    .preview-header h1 {
      color: #e6edf3;
      font-size: 20px;
      margin: 0 0 8px 0;
    }
  </style>
</head>
<body>
  <div class="preview-header">
    <h1>GitHub Actions Comment Preview</h1>
    <p>This is how your PR summary will appear in GitHub PR comments.</p>
  </div>
  <article class="markdown-body" id="content"></article>
  <script>
    const markdown = %s;
    document.getElementById('content').innerHTML = marked.parse(markdown);
  </script>
</body>
</html>`, prEscapeJSString(markdown))
}

// prEscapeJSString escapes a string for use in JavaScript
func prEscapeJSString(s string) string {
	result := "`"
	for _, r := range s {
		switch r {
		case '`':
			result += "\\`"
		case '\\':
			result += "\\\\"
		case '$':
			result += "\\$"
		default:
			result += string(r)
		}
	}
	result += "`"
	return result
}
