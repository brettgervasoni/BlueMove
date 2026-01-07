# BlueMove

A macOS tray app in Go that switches Magic Mouse/Keyboard (or any BT devices) between Macs using blueutil. Provides per-device toggle to connect/disconnect (unpair), optional auto-connect monitoring, and desktop notifications.

## Requirements
- Go 1.21+
- macOS
- blueutil (install via Homebrew):

```bash
brew install blueutil
```

## Configuration

The app looks for `blueMove.conf` in the following locations (in order):
1. `./blueMove.conf` (current working directory)
2. `~/.config/blueMove/blueMove.conf` (user config directory)

Create the config file:

```bash
mkdir -p ~/.config/blueMove
nano ~/.config/blueMove/blueMove.conf
```

### Config Format

Each line should be in the format `label=MAC-address`:

```ini
keyboard=AA-BB-CC-DD-EE-FF
mouse=FF-EE-DD-CC-BB-AA
```

- Labels are arbitrary (you choose the name)
- MACs may use `:` or `-` or no separator; they are normalized internally
- Lines starting with `#` or `;` are treated as comments

## Build

The recommended way to build blueMove is as a macOS .app bundle that runs only in the system tray (no terminal window, no dock icon):

```bash
bash scripts/build-app.sh
```

This creates `blueMove.app` with a proper app icon and Info.plist configuration.

### Alternative: Build as standalone binary

If you prefer a command-line binary:

**Intel (x86_64):**
```bash
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o blueMove-amd64 ./cmd/blueMove
```

**Apple Silicon (ARM64):**
```bash
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o blueMove-arm64 ./cmd/blueMove
```

**Default (current architecture):**
```bash
go build ./cmd/blueMove
```

**Note:** Cross-compilation requires CGO enabled because the app uses native macOS system tray APIs.

## Installation

### Install Using Release Build

- Download BlueMove.app.zip
- Unzip
- Drag BlueMove.app to /Applications
- Right-click → Open (first launch)

### Install the App Bundle by Building Yourself

1. Build the app:
   ```bash
   bash scripts/build-app.sh
   ```

2. Copy to Applications:
   ```bash
   cp -r blueMove.app /Applications/
   ```

### Initial Configuration

1. Review INSTALLATION.md to get up and going with your configuration files.

2. Launch from Applications or Spotlight

3. (Optional) Add to Login Items for automatic startup:
   - Open **System Settings** > **General** > **Login Items**
   - Click the **+** button and add **blueMove.app**

## Usage

The app runs only in the system tray with no terminal window or dock icon. Look for the blue lightning bolt icon in your menu bar. The menu includes:

- **Toggle Button**: Dynamically switches between "Connect All" and "Disconnect All" based on current device states
- **Device Checkboxes**: One per configured device showing label and MAC (checked when connected; click to toggle)
- **Quit**: Exit the application

The app automatically monitors and attempts to reconnect disconnected devices every 5 seconds using inquiry scans. After manually disconnecting a device, auto-reconnect is blocked for 120 seconds to respect user intent.

## Icon and Branding

The app uses a custom lightning bolt icon:
- **Menu Bar Icon**: Monochrome silhouette (22px) from `resources/blueMove-logo-monochrome.png` - adapts to light/dark mode
- **App Icon**: Full-color icon (.icns) from `resources/blueMove-logo.png` for Finder and dock

To regenerate the icons after updating the source images:

```bash
# Create monochrome menu bar icon from blueMove-logo-monochrome.png
bash scripts/create-menubar-icon.sh

# Create app icon (.icns) from blueMove-logo.png
bash scripts/create-icns.sh
```

## Development Notes

- Disconnect uses Unpair by design
- The app parses JSON output via `--format json-pretty` from listing commands. Some commands (e.g., `--pair`, `--connect`, `--unpair`) don't support JSON; their exit codes are mapped to helpful messages (see [blueutil docs](https://github.com/toy/blueutil))
- UI implemented with [systray](https://github.com/getlantern/systray). Notifications via [beeep](https://github.com/gen2brain/beeep)
- App bundle configuration includes `LSUIElement=true` in Info.plist to run as a menu bar-only app (no dock icon)

## Author

Brett Gervasoni