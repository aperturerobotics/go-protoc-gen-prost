#!/bin/bash
set -euo pipefail

# protoc-gen-prost WASI Update Script
# Embeds dist/protoc-gen-prost.wasm from a commit of aperturerobotics/protoc-gen-prost.
#
# Usage: ./update-prost.bash [ref]
#
# The ref is a commit, branch or tag of the repository and defaults to its wasi
# branch. It is resolved to a full commit so version.go names the exact source.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="aperturerobotics/protoc-gen-prost"
OUTPUT_NAME="protoc-gen-prost.wasm"
REF="${1:-wasi}"

echo "Resolving $REF in $REPO..."
COMMIT=$(gh api "repos/$REPO/commits/$REF" --jq .sha)
if [ -z "$COMMIT" ]; then
    echo "Error: Could not resolve $REF"
    exit 1
fi

DOWNLOAD_URL="https://raw.githubusercontent.com/$REPO/$COMMIT/dist/$OUTPUT_NAME"
echo "Downloading $DOWNLOAD_URL..."
curl --fail --silent --show-error --location "$DOWNLOAD_URL" --output "$SCRIPT_DIR/$OUTPUT_NAME"

echo "Downloaded $OUTPUT_NAME ($(wc -c < "$SCRIPT_DIR/$OUTPUT_NAME" | tr -d ' ') bytes)"

# Generate version info Go file
echo "Generating version.go..."
cat > "$SCRIPT_DIR/version.go" << GO
package prost

// protoc-gen-prost WASI build information
const (
	// Version is the full commit of $REPO whose
	// dist/$OUTPUT_NAME is embedded.
	Version = "$COMMIT"
	// DownloadURL is the URL of the embedded WASM file at that commit.
	DownloadURL = "$DOWNLOAD_URL"
)
GO

echo "Generated version.go with commit $COMMIT"
echo ""
echo "Update complete!"
