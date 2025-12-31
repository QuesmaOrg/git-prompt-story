#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

docker build -t git-prompt-story-e2e -f e2e/Dockerfile .
docker run --rm git-prompt-story-e2e
