#!/bin/bash

# Script to create macOS .icns file from PNG logo
# This creates all the required icon sizes for macOS

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
RESOURCES_DIR="$PROJECT_DIR/resources"
LOGO_PNG="$RESOURCES_DIR/blueMove-logo.png"
ICONSET_DIR="$RESOURCES_DIR/AppIcon.iconset"
OUTPUT_ICNS="$RESOURCES_DIR/AppIcon.icns"

echo "Creating icon set from $LOGO_PNG..."

# Check if source logo exists
if [ ! -f "$LOGO_PNG" ]; then
    echo "Error: Logo not found at $LOGO_PNG"
    exit 1
fi

# Check if sips command is available (should be on all macOS systems)
if ! command -v sips &> /dev/null; then
    echo "Error: sips command not found. This script requires macOS."
    exit 1
fi

# Create iconset directory
rm -rf "$ICONSET_DIR"
mkdir -p "$ICONSET_DIR"

# Generate all required icon sizes
# macOS requires these specific sizes for .icns files
echo "Generating icon sizes..."

sips -z 16 16     "$LOGO_PNG" --out "$ICONSET_DIR/icon_16x16.png" 2>&1 | grep -v "^/" || true
sips -z 32 32     "$LOGO_PNG" --out "$ICONSET_DIR/icon_16x16@2x.png" 2>&1 | grep -v "^/" || true
sips -z 32 32     "$LOGO_PNG" --out "$ICONSET_DIR/icon_32x32.png" 2>&1 | grep -v "^/" || true
sips -z 64 64     "$LOGO_PNG" --out "$ICONSET_DIR/icon_32x32@2x.png" 2>&1 | grep -v "^/" || true
sips -z 128 128   "$LOGO_PNG" --out "$ICONSET_DIR/icon_128x128.png" 2>&1 | grep -v "^/" || true
sips -z 256 256   "$LOGO_PNG" --out "$ICONSET_DIR/icon_128x128@2x.png" 2>&1 | grep -v "^/" || true
sips -z 256 256   "$LOGO_PNG" --out "$ICONSET_DIR/icon_256x256.png" 2>&1 | grep -v "^/" || true
sips -z 512 512   "$LOGO_PNG" --out "$ICONSET_DIR/icon_256x256@2x.png" 2>&1 | grep -v "^/" || true
sips -z 512 512   "$LOGO_PNG" --out "$ICONSET_DIR/icon_512x512.png" 2>&1 | grep -v "^/" || true
sips -z 1024 1024 "$LOGO_PNG" --out "$ICONSET_DIR/icon_512x512@2x.png" 2>&1 | grep -v "^/" || true

# Convert iconset to icns file
echo "Creating .icns file..."
iconutil -c icns "$ICONSET_DIR" -o "$OUTPUT_ICNS"

# Clean up iconset directory
rm -rf "$ICONSET_DIR"

echo "✓ Icon created successfully: $OUTPUT_ICNS"

