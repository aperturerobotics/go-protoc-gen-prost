#!/bin/bash
set -euo pipefail

# protoc-gen-prost WASI Update Script
# Downloads the WASM binary from aperturerobotics/protoc-gen-prost releases

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="aperturerobotics/protoc-gen-prost"
ASSET_NAME="protoc-gen-prost.wasm"
OUTPUT_NAME="protoc-gen-prost.wasm"

echo "Fetching latest release from $REPO..."

# Get the latest release
RELEASE_INFO=$(gh release view --repo "$REPO" --json tagName,assets)

TAG=$(echo "$RELEASE_INFO" | jq -r '.tagName')
DOWNLOAD_URL=$(echo "$RELEASE_INFO" | jq -r ".assets[] | select(.name == \"$ASSET_NAME\") | .url")

if [ -z "$TAG" ] || [ "$TAG" = "null" ]; then
    echo "Error: Could not find release"
    exit 1
fi

if [ -z "$DOWNLOAD_URL" ] || [ "$DOWNLOAD_URL" = "null" ]; then
    echo "Error: Could not find $ASSET_NAME in release $TAG"
    exit 1
fi

echo "Found release: $TAG"
echo "Downloading $ASSET_NAME..."

# Download the WASM file
gh release download "$TAG" --repo "$REPO" --pattern "$ASSET_NAME" --output "$SCRIPT_DIR/$OUTPUT_NAME" --clobber

echo "Downloaded $OUTPUT_NAME ($(wc -c < "$SCRIPT_DIR/$OUTPUT_NAME" | tr -d ' ') bytes)"

# Generate version info Go file
echo "Generating version.go..."
cat > "$SCRIPT_DIR/version.go" << EOF
package prost

// protoc-gen-prost WASI version information
const (
	// Version is the protoc-gen-prost version
	Version = "$TAG"
	// DownloadURL is the URL where this WASM file was downloaded from
	DownloadURL = "https://github.com/$REPO/releases/download/$TAG/$ASSET_NAME"
)
EOF

echo "Generated version.go with version $TAG"
echo ""
echo "Update complete!"
