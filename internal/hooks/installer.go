package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const prepareCommitMsgScript = `#!/bin/sh
exec git-prompt-story prepare-commit-msg "$@"
`

const postCommitScript = `#!/bin/sh
exec git-prompt-story post-commit
`

const postRewriteScript = `#!/bin/sh
exec git-prompt-story post-rewrite "$@"
`

// InstallHooks installs the git hooks
func InstallHooks(global bool) error {
	hooksDir, err := getHooksDir(global)
	if err != nil {
		return err
	}

	// Create hooks directory if needed
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	// Install prepare-commit-msg hook
	if err := writeHookScript(hooksDir, "prepare-commit-msg", prepareCommitMsgScript); err != nil {
		return err
	}

	// Install post-commit hook
	if err := writeHookScript(hooksDir, "post-commit", postCommitScript); err != nil {
		return err
	}

	// Install post-rewrite hook (for squash/rebase note transfer)
	if err := writeHookScript(hooksDir, "post-rewrite", postRewriteScript); err != nil {
		return err
	}

	if global {
		fmt.Printf("Hooks installed globally to %s\n", hooksDir)
	} else {
		fmt.Printf("Hooks installed to %s\n", hooksDir)
	}

	return nil
}

// getHooksDir returns the appropriate hooks directory
func getHooksDir(global bool) (string, error) {
	if global {
		// Get or create global hooks directory
		cmd := exec.Command("git", "config", "--global", "--get", "core.hooksPath")
		out, err := cmd.Output()
		if err == nil {
			path := strings.TrimSpace(string(out))
			if path != "" {
				return expandPath(path), nil
			}
		}

		// Set up default global hooks path
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		hooksPath := filepath.Join(homeDir, ".config", "git", "hooks")

		// Configure git to use this path
		cmd = exec.Command("git", "config", "--global", "core.hooksPath", hooksPath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to set global hooks path: %w", err)
		}

		return hooksPath, nil
	}

	// Local repo hooks
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository: %w", err)
	}

	gitDir := strings.TrimSpace(string(out))
	return filepath.Join(gitDir, "hooks"), nil
}

// writeHookScript writes a hook script file
func writeHookScript(hooksDir, hookName, content string) error {
	hookPath := filepath.Join(hooksDir, hookName)

	// Check if hook already exists and contains our marker
	if existing, err := os.ReadFile(hookPath); err == nil {
		if strings.Contains(string(existing), "git-prompt-story") {
			fmt.Printf("Hook %s already installed, skipping\n", hookName)
			return nil
		}
		// Backup existing hook
		backupPath := hookPath + ".backup"
		if err := os.WriteFile(backupPath, existing, 0755); err != nil {
			return fmt.Errorf("failed to backup existing hook: %w", err)
		}
		fmt.Printf("Backed up existing %s to %s.backup\n", hookName, hookName)
	}

	if err := os.WriteFile(hookPath, []byte(content), 0755); err != nil {
		return fmt.Errorf("failed to write %s hook: %w", hookName, err)
	}

	return nil
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
