#!/bin/bash

# Script to build blueMove as a macOS .app bundle
# This creates a proper macOS application for ARM64 (Apple Silicon) architecture

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
APP_NAME="BlueMove"

# Function to build for a specific architecture
build_for_arch() {
    local GOARCH=$1
    local BUNDLE_NAME="$APP_NAME"
    local APP_BUNDLE="$PROJECT_DIR/$BUNDLE_NAME.app"
    
    echo ""
    echo "Building $BUNDLE_NAME.app for $GOARCH..."
    
    # Clean previous build
    if [ -d "$APP_BUNDLE" ]; then
        echo "Removing existing app bundle..."
        rm -rf "$APP_BUNDLE"
    fi
    
    # Create app bundle structure
    echo "Creating app bundle structure..."
    mkdir -p "$APP_BUNDLE/Contents/MacOS"
    mkdir -p "$APP_BUNDLE/Contents/Resources"
    
    # Build the Go binary
    echo "Building Go binary for $GOARCH..."
    cd "$PROJECT_DIR"
    CGO_ENABLED=1 GOOS=darwin GOARCH=$GOARCH go build -o "$APP_BUNDLE/Contents/MacOS/$APP_NAME" ./cmd/blueMove
    
    # Copy Info.plist
    echo "Copying Info.plist..."
    cp "$PROJECT_DIR/resources/Info.plist" "$APP_BUNDLE/Contents/Info.plist"
    
    # Create icon if it doesn't exist
    if [ ! -f "$PROJECT_DIR/resources/AppIcon.icns" ]; then
        echo "Creating app icon..."
        bash "$SCRIPT_DIR/create-icns.sh"
    fi
    
    # Copy icon
    echo "Copying app icon..."
    cp "$PROJECT_DIR/resources/AppIcon.icns" "$APP_BUNDLE/Contents/Resources/AppIcon.icns"
    
    # Make binary executable
    chmod +x "$APP_BUNDLE/Contents/MacOS/$APP_NAME"
    
    echo "✓ Build complete for $GOARCH!"
    echo "App bundle created at: $APP_BUNDLE"
}

# Detect host architecture
HOST_ARCH=$(uname -m)
echo "Detected host architecture: $HOST_ARCH"

# Build for ARM64 (Apple Silicon)
build_for_arch "arm64"

echo ""
echo "========================================="
echo "✓ Build complete!"
echo "========================================="
echo ""
echo "App bundle created:"
echo "  - $PROJECT_DIR/$APP_NAME.app (Apple Silicon)"
echo ""
echo "To install:"
echo "  cp -r $PROJECT_DIR/$APP_NAME.app /Applications/"
echo ""
echo "To run directly:"
echo "  open $PROJECT_DIR/$APP_NAME.app"
echo ""
echo "To make it start at login:"
echo "  1. Open System Settings > General > Login Items"
echo "  2. Click the '+' button and add blueMove.app"
echo ""

