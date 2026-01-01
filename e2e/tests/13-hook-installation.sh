#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[13/13] Hook Installation"

# ============================================
# Test 1: Fresh install (no existing hooks)
# ============================================
echo "  Test 1: Fresh install with no existing hooks..."

cleanup_sessions
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Verify no hooks exist yet
[ ! -f .git/hooks/prepare-commit-msg ] || fail "prepare-commit-msg should not exist yet"
[ ! -f .git/hooks/post-commit ] || fail "post-commit should not exist yet"
[ ! -f .git/hooks/post-rewrite ] || fail "post-rewrite should not exist yet"

# Install hooks
git-prompt-story install-hooks

# Verify all hooks created
[ -f .git/hooks/prepare-commit-msg ] || fail "prepare-commit-msg not created"
[ -f .git/hooks/post-commit ] || fail "post-commit not created"
[ -f .git/hooks/post-rewrite ] || fail "post-rewrite not created"

# Verify no backup files
[ ! -f .git/hooks/prepare-commit-msg.orig ] || fail "should not have orig on fresh install"
[ ! -f .git/hooks/post-commit.orig ] || fail "should not have orig on fresh install"
[ ! -f .git/hooks/post-rewrite.orig ] || fail "should not have orig on fresh install"

# Verify hooks contain our marker
grep -q "git-prompt-story" .git/hooks/prepare-commit-msg || fail "prepare-commit-msg missing marker"
grep -q "git-prompt-story" .git/hooks/post-commit || fail "post-commit missing marker"
grep -q "git-prompt-story" .git/hooks/post-rewrite || fail "post-rewrite missing marker"

echo "    - Fresh install creates all hooks without backups"

# ============================================
# Test 2: Install with existing hooks - hooks still run
# ============================================
echo "  Test 2: Install with existing hooks (must still run)..."

cleanup_sessions
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Create custom hooks BEFORE our install
mkdir -p .git/hooks

# Custom post-commit that creates a marker file
cat > .git/hooks/post-commit << 'HOOK'
#!/bin/sh
touch /tmp/custom-hook-ran
HOOK
chmod +x .git/hooks/post-commit

# Custom prepare-commit-msg that adds a line
cat > .git/hooks/prepare-commit-msg << 'HOOK'
#!/bin/sh
echo "Custom-Hook: active" >> "$1"
HOOK
chmod +x .git/hooks/prepare-commit-msg

# Save original content for verification
ORIGINAL_POST_COMMIT=$(cat .git/hooks/post-commit)
ORIGINAL_PREPARE_MSG=$(cat .git/hooks/prepare-commit-msg)

# Clean marker file
rm -f /tmp/custom-hook-ran

# Install our hooks
git-prompt-story install-hooks

# Verify orig files were created
[ -f .git/hooks/post-commit.orig ] || fail "post-commit.orig not created"
[ -f .git/hooks/prepare-commit-msg.orig ] || fail "prepare-commit-msg.orig not created"

# Verify orig content matches original
[ "$(cat .git/hooks/post-commit.orig)" = "$ORIGINAL_POST_COMMIT" ] || fail "orig content mismatch"
[ "$(cat .git/hooks/prepare-commit-msg.orig)" = "$ORIGINAL_PREPARE_MSG" ] || fail "orig content mismatch"

# Now make a commit and verify BOTH hooks ran
echo "initial" > file.txt
git add file.txt

create_mock_session "hook-test-session" "2025-01-15T09:15:00Z" "2025-01-15T09:45:00Z"

export GIT_AUTHOR_DATE="2025-01-15T10:00:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:00:00Z"
faketime '2025-01-15 10:00:00' git commit -m "Test commit"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Verify custom post-commit hook ran (created marker file)
[ -f /tmp/custom-hook-ran ] || fail "Custom post-commit hook did not run"

# Verify our post-commit hook ran (note created)
git notes --ref=refs/notes/prompt-story show HEAD > /dev/null 2>&1 || fail "Our post-commit hook did not run (no note)"

# Verify custom prepare-commit-msg hook ran (added line to message)
git log -1 --format=%B | grep -q "Custom-Hook: active" || fail "Custom prepare-commit-msg hook did not run"

# Verify our prepare-commit-msg hook ran (added Prompt-Story line)
git log -1 --format=%B | grep -q "Prompt-Story:" || fail "Our prepare-commit-msg hook did not run"

echo "    - Both custom and our hooks ran successfully"

# ============================================
# Test 3: Reinstall is idempotent
# ============================================
echo "  Test 3: Reinstall is idempotent..."

# Save current hook content
HOOK_BEFORE=$(cat .git/hooks/post-commit)

# Reinstall
OUTPUT=$(git-prompt-story install-hooks 2>&1)

# Verify "already installed" message
echo "$OUTPUT" | grep -q "already installed" || fail "Should show 'already installed' message"

# Verify hook content unchanged
HOOK_AFTER=$(cat .git/hooks/post-commit)
[ "$HOOK_BEFORE" = "$HOOK_AFTER" ] || fail "Hook content changed on reinstall"

# Verify orig not modified (no double-backup)
[ "$(cat .git/hooks/post-commit.orig)" = "$ORIGINAL_POST_COMMIT" ] || fail "Orig was modified on reinstall"

echo "    - Reinstall is idempotent (no changes made)"

echo "  All hook installation tests passed"
