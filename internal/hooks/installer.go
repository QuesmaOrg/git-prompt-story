package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const prepareCommitMsgScript = `#!/bin/sh
# Chain to original hook if it exists (backup from local install)
if [ -x "$(dirname "$0")/prepare-commit-msg.orig" ]; then
    "$(dirname "$0")/prepare-commit-msg.orig" "$@" || exit $?
fi
# Chain to local repo hook (only when running from global hooks, not local)
GIT_HOOKS_DIR="$(git rev-parse --git-dir 2>/dev/null)/hooks"
if [ "$(cd "$(dirname "$0")" && pwd)" != "$(cd "$GIT_HOOKS_DIR" 2>/dev/null && pwd)" ]; then
    LOCAL_HOOK="$GIT_HOOKS_DIR/prepare-commit-msg"
    if [ -x "$LOCAL_HOOK" ]; then
        "$LOCAL_HOOK" "$@" || exit $?
    fi
fi
exec git-prompt-story prepare-commit-msg "$@"
`

const postCommitScript = `#!/bin/sh
# Chain to original hook if it exists (backup from local install)
if [ -x "$(dirname "$0")/post-commit.orig" ]; then
    "$(dirname "$0")/post-commit.orig" "$@" || exit $?
fi
# Chain to local repo hook (only when running from global hooks, not local)
GIT_HOOKS_DIR="$(git rev-parse --git-dir 2>/dev/null)/hooks"
if [ "$(cd "$(dirname "$0")" && pwd)" != "$(cd "$GIT_HOOKS_DIR" 2>/dev/null && pwd)" ]; then
    LOCAL_HOOK="$GIT_HOOKS_DIR/post-commit"
    if [ -x "$LOCAL_HOOK" ]; then
        "$LOCAL_HOOK" "$@" || exit $?
    fi
fi
exec git-prompt-story post-commit
`

const postRewriteScript = `#!/bin/sh
# Chain to original hook if it exists (backup from local install)
if [ -x "$(dirname "$0")/post-rewrite.orig" ]; then
    "$(dirname "$0")/post-rewrite.orig" "$@" || exit $?
fi
# Chain to local repo hook (only when running from global hooks, not local)
GIT_HOOKS_DIR="$(git rev-parse --git-dir 2>/dev/null)/hooks"
if [ "$(cd "$(dirname "$0")" && pwd)" != "$(cd "$GIT_HOOKS_DIR" 2>/dev/null && pwd)" ]; then
    LOCAL_HOOK="$GIT_HOOKS_DIR/post-rewrite"
    if [ -x "$LOCAL_HOOK" ]; then
        "$LOCAL_HOOK" "$@" || exit $?
    fi
fi
exec git-prompt-story post-rewrite "$@"
`

const prePushScript = `#!/bin/sh
# Chain to original hook if it exists (backup from local install)
if [ -x "$(dirname "$0")/pre-push.orig" ]; then
    "$(dirname "$0")/pre-push.orig" "$@" || exit $?
fi
# Chain to local repo hook (only when running from global hooks, not local)
GIT_HOOKS_DIR="$(git rev-parse --git-dir 2>/dev/null)/hooks"
if [ "$(cd "$(dirname "$0")" && pwd)" != "$(cd "$GIT_HOOKS_DIR" 2>/dev/null && pwd)" ]; then
    LOCAL_HOOK="$GIT_HOOKS_DIR/pre-push"
    if [ -x "$LOCAL_HOOK" ]; then
        "$LOCAL_HOOK" "$@" || exit $?
    fi
fi
# Pass remote name and URL to git-prompt-story
# stdin contains ref info, pass it through
exec git-prompt-story pre-push "$@"
`

// InstallOptions configures hook installation
type InstallOptions struct {
	Global   bool
	AutoPush bool
}

// InstallHooks installs the git hooks
func InstallHooks(opts InstallOptions) error {
	hooksDir, err := getHooksDir(opts.Global)
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

	// Optionally install pre-push hook for auto-syncing notes
	if opts.AutoPush {
		if err := writeHookScript(hooksDir, "pre-push", prePushScript); err != nil {
			return err
		}
		fmt.Println("Pre-push hook installed (notes will auto-sync on push)")
	}

	if opts.Global {
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
		backupPath := hookPath + ".orig"
		if err := os.WriteFile(backupPath, existing, 0755); err != nil {
			return fmt.Errorf("failed to backup existing hook: %w", err)
		}
		fmt.Printf("Backed up existing %s to %s.orig\n", hookName, hookName)
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
