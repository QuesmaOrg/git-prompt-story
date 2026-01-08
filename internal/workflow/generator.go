package workflow

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const workflowTemplatePages = `name: Prompt Story

on:
  pull_request:
    types: [opened, synchronize, reopened, closed]

permissions:
  contents: write
  pull-requests: write
  pages: read

jobs:
  prompt-story:
    if: github.event.action != 'closed'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: QuesmaOrg/git-prompt-story/.github/actions/prompt-story-with-pages@main
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          comment: true

  cleanup-old-previews:
    if: github.event.action == 'closed'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          ref: gh-pages
          fetch-depth: 1

      - name: Remove old PR preview directories
        env:
          GH_TOKEN: ${{ github.token }}
          RETENTION_DAYS: 30
        run: |
          cd prompt-story 2>/dev/null || exit 0
          for dir in pr-*; do
            [ -d "$dir" ] || continue
            pr_num="${dir#pr-}"

            closed_at=$(gh pr view "$pr_num" --json closedAt -q '.closedAt' 2>/dev/null || echo "")
            [ -z "$closed_at" ] && continue

            closed_epoch=$(date -d "$closed_at" +%s)
            now_epoch=$(date +%s)
            age_days=$(( (now_epoch - closed_epoch) / 86400 ))

            if [ "$age_days" -ge "$RETENTION_DAYS" ]; then
              echo "Removing prompt-story/$dir (closed $age_days days ago)"
              rm -rf "$dir"
            fi
          done

      - name: Commit cleanup
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git add -A
          git diff --staged --quiet || git commit -m "Cleanup: remove old PR previews" && git push
`

const workflowTemplateNoPages = `name: Prompt Story

on:
  pull_request:
    types: [opened, synchronize, reopened]

permissions:
  contents: read
  pull-requests: write

jobs:
  prompt-story:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: QuesmaOrg/git-prompt-story/.github/actions/prompt-story@main
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          comment: true
`

// Generate creates the GitHub workflow file with interactive prompts
func Generate() error {
	fmt.Println("Generating GitHub Action workflow for prompt-story...")
	fmt.Println()

	enablePages := askYesNo("Enable GitHub Pages for full transcripts?", false)

	// Determine workflow content
	var content string
	if enablePages {
		content = workflowTemplatePages
	} else {
		content = workflowTemplateNoPages
	}

	// Create .github/workflows directory
	workflowDir := filepath.Join(".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s: %w", workflowDir, err)
	}

	// Write workflow file
	workflowPath := filepath.Join(workflowDir, "prompt-story.yml")
	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", workflowPath, err)
	}

	fmt.Println()
	fmt.Printf("Created %s\n", workflowPath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("1. Commit and push the workflow file")

	if enablePages {
		fmt.Println("2. After the first PR with this workflow runs, enable GitHub Pages:")
		fmt.Println("   - Go to repository Settings -> Pages")
		fmt.Println("   - Under \"Source\", select \"Deploy from a branch\"")
		fmt.Println("   - Select \"gh-pages\" branch and \"/ (root)\" folder")
		fmt.Println("   - Click Save")
		fmt.Println()
		fmt.Println("   The gh-pages branch is created automatically by the workflow.")
	}

	return nil
}

// askYesNo prompts the user with a yes/no question and returns the answer
func askYesNo(question string, defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)

	prompt := question
	if defaultYes {
		prompt += " [Y/n]: "
	} else {
		prompt += " [y/N]: "
	}

	fmt.Print(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultYes
	}

	return input == "y" || input == "yes"
}
