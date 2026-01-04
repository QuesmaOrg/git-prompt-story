#!/bin/bash
set -euo pipefail
source /e2e/lib/helpers.sh

echo "[14/14] Pre-Push Hook"

# ============================================
# Test 1: Install without --auto-push does NOT create pre-push hook
# ============================================
echo "  Test 1: Install without --auto-push does NOT create pre-push hook..."

cleanup_sessions
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Install without --auto-push
git-prompt-story install-hooks

# Verify pre-push hook NOT created
[ ! -f .git/hooks/pre-push ] || fail "pre-push should not exist without --auto-push"

echo "    - Without --auto-push, no pre-push hook created"

# ============================================
# Test 2: Install with --auto-push creates pre-push hook
# ============================================
echo "  Test 2: Install with --auto-push creates pre-push hook..."

cleanup_sessions
rm -rf /workspace/test-repo
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init

# Install with --auto-push
git-prompt-story install-hooks --auto-push

# Verify pre-push hook IS created
[ -f .git/hooks/pre-push ] || fail "pre-push not created with --auto-push"
grep -q "git-prompt-story" .git/hooks/pre-push || fail "pre-push missing marker"
grep -q "pre-push" .git/hooks/pre-push || fail "pre-push missing pre-push command"

echo "    - --auto-push flag creates pre-push hook"

# ============================================
# Test 3: Pre-push hook pushes notes to remote
# ============================================
echo "  Test 3: Pre-push hook pushes notes to remote..."

cleanup_sessions
rm -rf /workspace/test-repo /workspace/remote-repo

# Create bare remote repo
mkdir -p /workspace/remote-repo
cd /workspace/remote-repo
git init --bare

# Create local repo with remote
mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init
git remote add origin /workspace/remote-repo

# Install hooks with auto-push
git-prompt-story install-hooks --auto-push

# Create a commit with session
echo "content" > file.txt
git add file.txt

create_mock_session "pre-push-test" "2025-01-15T09:15:00Z" "2025-01-15T09:45:00Z"

export GIT_AUTHOR_DATE="2025-01-15T10:00:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:00:00Z"
faketime '2025-01-15 10:00:00' git commit -m "Test commit"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Verify note exists locally
git notes --ref=refs/notes/prompt-story show HEAD > /dev/null 2>&1 || fail "Note not created locally"

# Push to remote (this should trigger pre-push hook)
git push -u origin main 2>&1

# Verify notes were pushed to remote
cd /workspace/remote-repo
git notes --ref=refs/notes/prompt-story list > /dev/null 2>&1 || fail "Notes not pushed to remote"

echo "    - Pre-push hook successfully pushed notes to remote"

# ============================================
# Test 4: Pre-push works without notes (no error)
# ============================================
echo "  Test 4: Pre-push works when no notes exist..."

cleanup_sessions
rm -rf /workspace/test-repo /workspace/remote-repo

mkdir -p /workspace/remote-repo
cd /workspace/remote-repo
git init --bare

mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init
git remote add origin /workspace/remote-repo

git-prompt-story install-hooks --auto-push

# Create commit WITHOUT session (no notes)
echo "content" > file.txt
git add file.txt
git commit -m "No session commit"

# Push should succeed without error
git push -u origin main 2>&1 || fail "Push failed when no notes exist"

echo "    - Pre-push succeeds gracefully when no notes exist"

# ============================================
# Test 5: Pre-push works when pushing specific branch
# ============================================
echo "  Test 5: Pre-push works when pushing a specific branch..."

cleanup_sessions
rm -rf /workspace/test-repo /workspace/remote-repo

mkdir -p /workspace/remote-repo
cd /workspace/remote-repo
git init --bare

mkdir -p /workspace/test-repo
cd /workspace/test-repo
git init
git remote add origin /workspace/remote-repo

git-prompt-story install-hooks --auto-push

# Create initial commit on main
echo "initial" > file.txt
git add file.txt
git commit -m "Initial commit"
git push -u origin main 2>&1

# Create feature branch with session
git checkout -b feature-branch
echo "feature" > feature.txt
git add feature.txt

create_mock_session "feature-test" "2025-01-15T09:15:00Z" "2025-01-15T09:45:00Z"

export GIT_AUTHOR_DATE="2025-01-15T10:00:00Z"
export GIT_COMMITTER_DATE="2025-01-15T10:00:00Z"
faketime '2025-01-15 10:00:00' git commit -m "Feature commit"
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Push feature branch explicitly (git push origin feature-branch)
git push origin feature-branch 2>&1

# Verify notes were pushed to remote
cd /workspace/remote-repo
git notes --ref=refs/notes/prompt-story list > /dev/null 2>&1 || fail "Notes not pushed when pushing specific branch"

echo "    - Pre-push works when pushing specific branch"

echo "  All pre-push hook tests passed"
