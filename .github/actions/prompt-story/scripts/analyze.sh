#!/bin/bash
set -e

echo "Analyzing commits..."

COMMIT_RANGE="${BASE_SHA}..${HEAD_SHA}"
echo "  Range: $COMMIT_RANGE"

# Check if any commits have prompt-story markers in their messages
COMMITS_WITH_MARKERS=$(git log --format=%B ${COMMIT_RANGE} | grep -c "prompt-story-" || echo "0")
echo "  Commits with markers: $COMMITS_WITH_MARKERS"

# Determine flags
FULL_FLAG=""
if [ "$SUMMARY_MODE" = "full" ]; then
  FULL_FLAG="--full"
fi

# Generate summary in JSON to extract stats
./git-prompt-story ci-summary "$COMMIT_RANGE" --format=json --output=./prompt-story-stats.json $FULL_FLAG

# Extract stats from JSON
COMMITS_ANALYZED=$(jq -r '.commits_analyzed' ./prompt-story-stats.json)
COMMITS_WITH_NOTES=$(jq -r '.commits_with_notes' ./prompt-story-stats.json)

echo "  Commits analyzed: $COMMITS_ANALYZED"
echo "  Commits with notes: $COMMITS_WITH_NOTES"

# Set outputs
echo "commits-analyzed=$COMMITS_ANALYZED" >> $GITHUB_OUTPUT
echo "commits-with-notes=$COMMITS_WITH_NOTES" >> $GITHUB_OUTPUT
echo "commits-with-markers=$COMMITS_WITH_MARKERS" >> $GITHUB_OUTPUT

# Check if we should fail
if [ "$FAIL_IF_NO_NOTES" = "true" ] && [ "$COMMITS_WITH_NOTES" = "0" ]; then
  echo "Error: No prompt-story notes found and fail-if-no-notes is set"
  exit 1
fi

# Generate markdown summary
./git-prompt-story ci-summary "$COMMIT_RANGE" --format=markdown --output=./prompt-story-summary.md $FULL_FLAG

echo "  Generated ./prompt-story-summary.md"
echo "Done analyzing."
