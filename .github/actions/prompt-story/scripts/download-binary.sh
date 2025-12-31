#!/bin/bash
set -e

echo "Downloading git-prompt-story..."

# Determine version
if [ "$VERSION" = "latest" ] || [ -z "$VERSION" ]; then
  VERSION=$(curl -sL https://api.github.com/repos/QuesmaOrg/git-prompt-story/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  if [ -z "$VERSION" ]; then
    echo "Error: Could not determine latest version"
    exit 1
  fi
fi

echo "  Version: $VERSION"

# Determine platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)
    ARCH="amd64"
    ;;
  aarch64|arm64)
    ARCH="arm64"
    ;;
  *)
    echo "Error: Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

echo "  Platform: ${OS}_${ARCH}"

# Download and extract
DOWNLOAD_URL="https://github.com/QuesmaOrg/git-prompt-story/releases/download/${VERSION}/git-prompt-story_${OS}_${ARCH}.tar.gz"
echo "  URL: $DOWNLOAD_URL"

if ! curl -sL "$DOWNLOAD_URL" | tar xz; then
  echo "Error: Failed to download git-prompt-story"
  echo "The release may not exist yet. Please ensure releases are published."
  exit 1
fi

# Verify binary
if [ ! -f "./git-prompt-story" ]; then
  echo "Error: Binary not found after extraction"
  exit 1
fi

chmod +x ./git-prompt-story
echo "  Downloaded successfully: $(./git-prompt-story --version)"
