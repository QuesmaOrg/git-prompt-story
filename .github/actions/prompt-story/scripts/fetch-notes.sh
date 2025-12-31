#!/bin/bash
set -e

echo "Fetching git notes..."

# Fetch the commits notes ref
if git fetch origin 'refs/notes/commits:refs/notes/commits' --force 2>/dev/null; then
  echo "  - Fetched refs/notes/commits"
else
  echo "  - No refs/notes/commits found (this is OK if no commits have prompt-story notes)"
fi

# Fetch the transcripts notes ref
if git fetch origin 'refs/notes/prompt-story-transcripts:refs/notes/prompt-story-transcripts' --force 2>/dev/null; then
  echo "  - Fetched refs/notes/prompt-story-transcripts"
else
  echo "  - No refs/notes/prompt-story-transcripts found (this is OK)"
fi

echo "Done fetching notes."
