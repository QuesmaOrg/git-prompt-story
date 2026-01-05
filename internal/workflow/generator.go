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
    types: [opened, synchronize, reopened]

permissions:
  contents: write
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
          deploy-pages: true
`

const workflowTemplateNoPages = `name: Prompt Story

on:
  pull_request:
    types: [opened, synchronize, reopened]

permissions:
  contents: write
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
          deploy-pages: false
`

// Generate creates the GitHub workflow file with interactive prompts
func Generate() error {
	fmt.Println("Generating GitHub Action workflow for prompt-story...")
	fmt.Println()

	enablePages := askYesNo("Enable GitHub Pages for full transcripts?", true)

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
