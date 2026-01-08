#!/bin/bash
set -e

echo "Fetching git notes..."

# Force fetch notes refs (--force) because notes can be amended/rebased.
# If notes were pushed after this workflow started, you may need to
# re-trigger the GitHub Action job to see the updated summary.

# Fetch the prompt-story notes ref
if git fetch origin 'refs/notes/prompt-story:refs/notes/prompt-story' --force 2>/dev/null; then
  echo "  - Fetched refs/notes/prompt-story"
else
  echo "  - No refs/notes/prompt-story found (this is OK if no commits have prompt-story notes)"
fi

# Fetch the transcripts notes ref
if git fetch origin 'refs/notes/prompt-story-transcripts:refs/notes/prompt-story-transcripts' --force 2>/dev/null; then
  echo "  - Fetched refs/notes/prompt-story-transcripts"
else
  echo "  - No refs/notes/prompt-story-transcripts found (this is OK)"
fi

echo "Done fetching notes."
