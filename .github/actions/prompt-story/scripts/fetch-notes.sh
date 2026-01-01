#!/bin/bash
set -e

echo "Fetching git notes..."

# Fetch the prompt-story notes ref (primary)
if git fetch origin 'refs/notes/prompt-story:refs/notes/prompt-story' --force 2>/dev/null; then
  echo "  - Fetched refs/notes/prompt-story"
else
  echo "  - No refs/notes/prompt-story found (this is OK if no commits have prompt-story notes)"
fi

# Fetch legacy notes ref for backward compatibility
if git fetch origin 'refs/notes/commits:refs/notes/commits' --force 2>/dev/null; then
  echo "  - Fetched refs/notes/commits (legacy)"
else
  echo "  - No refs/notes/commits found (legacy ref, OK if not present)"
fi

# Fetch the transcripts notes ref
if git fetch origin 'refs/notes/prompt-story-transcripts:refs/notes/prompt-story-transcripts' --force 2>/dev/null; then
  echo "  - Fetched refs/notes/prompt-story-transcripts"
else
  echo "  - No refs/notes/prompt-story-transcripts found (this is OK)"
fi

echo "Done fetching notes."
