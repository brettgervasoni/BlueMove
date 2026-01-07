#!/bin/bash

# Script to create menu bar icon from the monochrome logo
# Menu bar icons should be black/white and around 22px high for proper macOS appearance

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
RESOURCES_DIR="$PROJECT_DIR/resources"
MONOCHROME_LOGO="$RESOURCES_DIR/blueMove-logo-monochrome.png"
MENUBAR_ICON="$RESOURCES_DIR/menubar-icon.png"
MENUBAR_ICON_2X="$RESOURCES_DIR/menubar-icon@2x.png"

echo "Creating menu bar icons from $MONOCHROME_LOGO..."

# Check if source logo exists
if [ ! -f "$MONOCHROME_LOGO" ]; then
    echo "Error: Monochrome logo not found at $MONOCHROME_LOGO"
    exit 1
fi

# Create menu bar icons (38px and 76px for retina - larger to match other menu bar icons)
echo "Generating menu bar icon sizes..."
sips -z 38 38 "$MONOCHROME_LOGO" --out "$MENUBAR_ICON" > /dev/null 2>&1
sips -z 76 76 "$MONOCHROME_LOGO" --out "$MENUBAR_ICON_2X" > /dev/null 2>&1

echo "✓ Menu bar icons created successfully"
echo "  - $MENUBAR_ICON"
echo "  - $MENUBAR_ICON_2X"
