#!/bin/bash
set -e

echo "Downloading git-prompt-story..."

# Build auth header if token provided
AUTH_HEADER=""
if [ -n "$GITHUB_TOKEN" ]; then
  AUTH_HEADER="Authorization: token $GITHUB_TOKEN"
fi

# Determine version
if [ "$VERSION" = "latest" ] || [ -z "$VERSION" ]; then
  if [ -n "$AUTH_HEADER" ]; then
    VERSION=$(curl -sL -H "$AUTH_HEADER" https://api.github.com/repos/QuesmaOrg/git-prompt-story/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  else
    VERSION=$(curl -sL https://api.github.com/repos/QuesmaOrg/git-prompt-story/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  fi
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
ASSET_NAME="git-prompt-story_${OS}_${ARCH}.tar.gz"

if [ -n "$AUTH_HEADER" ]; then
  # For private repos, use API to get asset URL and download with octet-stream
  echo "  Fetching release assets..."
  RELEASE_DATA=$(curl -sL -H "$AUTH_HEADER" \
    "https://api.github.com/repos/QuesmaOrg/git-prompt-story/releases/tags/${VERSION}")

  # Extract asset URL for our platform
  ASSET_URL=$(echo "$RELEASE_DATA" | grep -B3 "\"name\": \"${ASSET_NAME}\"" | grep '"url"' | head -1 | sed -E 's/.*"url": "([^"]+)".*/\1/')

  if [ -z "$ASSET_URL" ]; then
    echo "Error: Could not find asset ${ASSET_NAME} in release ${VERSION}"
    exit 1
  fi

  echo "  Asset URL: $ASSET_URL"

  if ! curl -sL -H "$AUTH_HEADER" -H "Accept: application/octet-stream" "$ASSET_URL" | tar xz; then
    echo "Error: Failed to download git-prompt-story"
    echo "The release may not exist yet. Please ensure releases are published."
    exit 1
  fi
else
  # For public repos, direct download works
  DOWNLOAD_URL="https://github.com/QuesmaOrg/git-prompt-story/releases/download/${VERSION}/${ASSET_NAME}"
  echo "  URL: $DOWNLOAD_URL"

  if ! curl -sL "$DOWNLOAD_URL" | tar xz; then
    echo "Error: Failed to download git-prompt-story"
    echo "The release may not exist yet. Please ensure releases are published."
    exit 1
  fi
fi

# Verify binary
if [ ! -f "./git-prompt-story" ]; then
  echo "Error: Binary not found after extraction"
  exit 1
fi

chmod +x ./git-prompt-story
echo "  Downloaded successfully: $(./git-prompt-story --version)"
