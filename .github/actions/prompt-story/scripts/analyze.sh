#!/bin/bash
set -e

echo "Analyzing commits..."

# Use branch reference to get commits new on PR branch
COMMIT_RANGE="origin/${BASE_REF}..HEAD"
echo "  Range: $COMMIT_RANGE"

# Use the analyze-pr command which encapsulates all analysis logic
# This avoids fragile bash-based marker detection
./git-prompt-story analyze-pr "$COMMIT_RANGE" \
  --output-json=./prompt-story-analysis.json \
  --output-markdown=./prompt-story-summary.md \
  ${PAGES_URL:+--pages-url="$PAGES_URL"}

# Extract stats from JSON using jq
COMMITS_ANALYZED=$(jq -r '.commits_analyzed' ./prompt-story-analysis.json)
COMMITS_WITH_NOTES=$(jq -r '.commits_with_notes' ./prompt-story-analysis.json)
SHOULD_POST=$(jq -r '.should_post_comment' ./prompt-story-analysis.json)

echo "  Commits analyzed: $COMMITS_ANALYZED"
echo "  Commits with notes: $COMMITS_WITH_NOTES"
echo "  Should post comment: $SHOULD_POST"

# Set outputs for GitHub Actions
echo "commits-analyzed=$COMMITS_ANALYZED" >> $GITHUB_OUTPUT
echo "commits-with-notes=$COMMITS_WITH_NOTES" >> $GITHUB_OUTPUT
echo "should-post-comment=$SHOULD_POST" >> $GITHUB_OUTPUT

# Check if we should fail
if [ "$FAIL_IF_NO_NOTES" = "true" ] && [ "$COMMITS_WITH_NOTES" = "0" ]; then
  echo "Error: No prompt-story notes found and fail-if-no-notes is set"
  exit 1
fi

echo "Done analyzing."
