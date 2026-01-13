#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[21/21] Global Hook Installation"

# ============================================
# Test 1: Fresh global install sets config and creates hooks
# ============================================
echo "  Test 1: Fresh global install sets config and creates hooks..."

cleanup_sessions
cleanup_global_hooks
rm -rf /workspace/test-repo

# Install globally (can be run from anywhere)
cd /workspace
git-prompt-story install-hooks --global

# Verify core.hooksPath is set
HOOKS_PATH=$(git config --global --get core.hooksPath)
[ "$HOOKS_PATH" = "$HOME/.config/git/hooks" ] || fail "core.hooksPath not set correctly: $HOOKS_PATH"

# Verify all hooks exist
[ -f "$HOME/.config/git/hooks/prepare-commit-msg" ] || fail "prepare-commit-msg not created globally"
[ -f "$HOME/.config/git/hooks/post-commit" ] || fail "post-commit not created globally"
[ -f "$HOME/.config/git/hooks/post-rewrite" ] || fail "post-rewrite not created globally"

# Verify hooks contain our marker
grep -q "git-prompt-story" "$HOME/.config/git/hooks/prepare-commit-msg" || fail "prepare-commit-msg missing marker"
grep -q "git-prompt-story" "$HOME/.config/git/hooks/post-commit" || fail "post-commit missing marker"
grep -q "git-prompt-story" "$HOME/.config/git/hooks/post-rewrite" || fail "post-rewrite missing marker"

echo "    - Global install sets core.hooksPath and creates hooks"

# ============================================
# Test 2: Global hooks work in a fresh repository
# ============================================
echo "  Test 2: Global hooks work in a fresh repository..."

cleanup_sessions
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Verify .git/hooks has no our hooks (they're global)
[ ! -f .git/hooks/prepare-commit-msg ] || fail "Local prepare-commit-msg should not exist"
[ ! -f .git/hooks/post-commit ] || fail "Local post-commit should not exist"

# Create a mock session
create_mock_session "global-test-session" "2025-01-15T09:15:00Z" "2025-01-15T09:45:00Z"

# Make a commit
echo "content" > file.txt
git add file.txt

export GIT_AUTHOR_DATE="2025-01-15T10:00:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:00:00Z"
faketime '2025-01-15 10:00:00' git commit -m "Test commit"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Verify Prompt-Story marker in commit message
git log -1 --format=%B | grep -q "Prompt-Story:" || fail "Prompt-Story marker not in commit message"

# Verify note exists
git notes --ref=refs/notes/prompt-story show HEAD > /dev/null 2>&1 || fail "Git note not created"

echo "    - Global hooks work in fresh repo without local install-hooks"

# ============================================
# Test 3: Global hooks chain to local .git/hooks (THE KEY FIX)
# ============================================
echo "  Test 3: Global hooks chain to local .git/hooks..."

cleanup_sessions
rm -rf /workspace/test-repo
rm -f /tmp/local-hook-ran
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Create a local hook that creates a marker file
mkdir -p .git/hooks
cat > .git/hooks/post-commit << 'HOOK'
#!/bin/sh
touch /tmp/local-hook-ran
HOOK
chmod +x .git/hooks/post-commit

# Create a mock session
create_mock_session "chain-test-session" "2025-01-15T09:15:00Z" "2025-01-15T09:45:00Z"

# Make a commit
echo "content" > file.txt
git add file.txt

export GIT_AUTHOR_DATE="2025-01-15T10:00:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:00:00Z"
faketime '2025-01-15 10:00:00' git commit -m "Test chaining"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Verify local hook DID run (this is the key fix!)
[ -f /tmp/local-hook-ran ] || fail "Local hook did not run - chaining broken!"

# Verify global hook ALSO ran (note created)
git notes --ref=refs/notes/prompt-story show HEAD > /dev/null 2>&1 || fail "Global hook did not run (no note)"

echo "    - Both local and global hooks ran (chaining works!)"

# ============================================
# Test 4: Reinstall is idempotent
# ============================================
echo "  Test 4: Reinstall is idempotent..."

# Save current hook content
HOOK_BEFORE=$(cat "$HOME/.config/git/hooks/post-commit")

# Reinstall
OUTPUT=$(git-prompt-story install-hooks --global 2>&1)

# Verify "already installed" message
echo "$OUTPUT" | grep -q "already installed" || fail "Should show 'already installed' message"

# Verify hook content unchanged
HOOK_AFTER=$(cat "$HOME/.config/git/hooks/post-commit")
[ "$HOOK_BEFORE" = "$HOOK_AFTER" ] || fail "Hook content changed on reinstall"

echo "    - Reinstall is idempotent (no changes made)"

# ============================================
# Test 5: Global install respects existing core.hooksPath
# ============================================
echo "  Test 5: Global install respects existing core.hooksPath..."

cleanup_global_hooks

# Set a custom hooks path before install
mkdir -p /tmp/custom-hooks
git config --global core.hooksPath /tmp/custom-hooks

# Install globally
git-prompt-story install-hooks --global

# Verify hooks went to custom location, NOT default
[ -f /tmp/custom-hooks/post-commit ] || fail "Hook not installed to custom path"
[ ! -f "$HOME/.config/git/hooks/post-commit" ] || fail "Hook should not be in default location"

echo "    - Global install respects existing core.hooksPath"

# ============================================
# Test 6: Global install with --auto-push
# ============================================
echo "  Test 6: Global install with --auto-push..."

cleanup_global_hooks

# Install globally with auto-push
git-prompt-story install-hooks --global --auto-push

# Verify pre-push hook exists
[ -f "$HOME/.config/git/hooks/pre-push" ] || fail "pre-push not created with --auto-push"
grep -q "git-prompt-story" "$HOME/.config/git/hooks/pre-push" || fail "pre-push missing marker"

echo "    - --auto-push flag creates pre-push hook globally"

# Cleanup at end
cleanup_global_hooks

echo "  All global hook installation tests passed"
