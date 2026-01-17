#!/bin/bash
set -e

echo "Analyzing commits..."

# Use branch reference to get commits new on PR branch
COMMIT_RANGE="origin/${BASE_REF}..HEAD"
echo "  Range: $COMMIT_RANGE"

# Use pr summary with --gha flag
# Outputs metadata to stdout (goes to GITHUB_OUTPUT)
# Writes markdown to file if there are notes
./git-prompt-story pr summary "$COMMIT_RANGE" \
  --gha \
  --output=./prompt-story-summary.md \
  ${PAGES_URL:+--pages-url="$PAGES_URL"} \
  >> $GITHUB_OUTPUT

# Parse output for logging (metadata is also in GITHUB_OUTPUT now)
COMMITS_ANALYZED=$(grep "commits-analyzed=" $GITHUB_OUTPUT | tail -1 | cut -d= -f2)
COMMITS_WITH_NOTES=$(grep "commits-with-notes=" $GITHUB_OUTPUT | tail -1 | cut -d= -f2)
COMMITS_MISSING=$(grep "commits-missing-notes=" $GITHUB_OUTPUT | tail -1 | cut -d= -f2)
NOTES_MISSING=$(grep "notes-missing=" $GITHUB_OUTPUT | tail -1 | cut -d= -f2)
SHOULD_POST=$(grep "should-post-comment=" $GITHUB_OUTPUT | tail -1 | cut -d= -f2)

echo "  Commits analyzed: $COMMITS_ANALYZED"
echo "  Commits with notes: $COMMITS_WITH_NOTES"
echo "  Commits missing notes: $COMMITS_MISSING"
echo "  Notes missing: $NOTES_MISSING"
echo "  Should post comment: $SHOULD_POST"

# Check if we should fail
if [ "$FAIL_IF_NO_NOTES" = "true" ] && [ "$COMMITS_WITH_NOTES" = "0" ]; then
  echo "Error: No prompt-story notes found and fail-if-no-notes is set"
  exit 1
fi

echo "Done analyzing."
