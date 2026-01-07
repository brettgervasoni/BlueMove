# BlueMove Installation Guide

## Quick Start

### 1. Configure Your Devices

Create the config directory and file:

```bash
mkdir -p ~/.config/blueMove
nano ~/.config/blueMove/blueMove.conf
```

Add your devices to the config file:
```ini
keyboard=AA-BB-CC-DD-EE-FF
mouse=FF-EE-DD-CC-BB-AA
```

### 2. Build the App

Build the macOS application bundle:

```bash
bash scripts/build-app.sh
```

This creates `blueMove.app` - a proper macOS app that runs only in the menu bar.

### 3. Install to Applications

Copy the app to your Applications folder:

```bash
cp -r blueMove.app /Applications/
```

### 4. Launch

Launch the app:
- Open from **Applications** folder
- Search for "blueMove" in **Spotlight** (⌘+Space)
- Or run: `open -a blueMove`

Look for the blue lightning bolt icon in your menu bar!

### 5. Start at Login (Optional)

To make blueMove start automatically when you log in:

1. Open **System Settings** > **General** > **Login Items**
2. Click the **+** button
3. Select **blueMove** from Applications
4. Ensure it's enabled

## What's Included

### Scripts

- `scripts/build-app.sh` - Build the macOS .app bundle
- `scripts/create-icns.sh` - Generate app icon from logo
- `scripts/create-menubar-icon.sh` - Generate menu bar icon from logo

### App Bundle Structure

```
blueMove.app/
├── Contents/
│   ├── Info.plist          # App configuration (LSUIElement=true for menu bar only)
│   ├── MacOS/
│   │   └── blueMove        # Binary executable
│   └── Resources/
│       └── AppIcon.icns    # App icon (dock, Finder)
```

### Icon Assets

- `resources/blueMove-logo.png` - Source logo for app icon (1024x1024)
- `resources/blueMove-logo-monochrome.png` - Source monochrome logo for menu bar (1024x1024)
- `resources/AppIcon.icns` - macOS app icon (generated from blueMove-logo.png)
- `internal/assets/menubar-icon.png` - Menu bar icon (22x22, monochrome, embedded in binary)

## Features

- **Menu Bar Only**: Runs only in the system tray (no dock icon, no terminal window)
- **Custom Icon**: Monochrome lightning bolt icon in menu bar (adapts to light/dark mode)
- **Auto-Connect**: Automatically reconnects devices every 5 seconds
- **Manual Control**: Toggle devices individually or all at once
- **Smart Cooldown**: Respects manual disconnections for 120 seconds
- **Notifications**: Desktop notifications for connection events

## Troubleshooting

### App Won't Open

If the app says "cannot be opened", try:

```bash
xattr -cr blueMove.app
```

Then rebuild:
```bash
bash scripts/build-app.sh
```

### Config Not Found

Make sure your config is in one of these locations:
- `./blueMove.conf` (current directory)
- `~/.config/blueMove/blueMove.conf` (recommended)

Create it manually:
```bash
mkdir -p ~/.config/blueMove
nano ~/.config/blueMove/blueMove.conf
```

### blueutil Not Found

Install blueutil via Homebrew:
```bash
brew install blueutil
```

### App Not in Menu Bar

1. Check that the app is running: `ps aux | grep blueMove`
2. Try quitting and relaunching
3. Check System Settings > Control Center > Menu Bar Only (should see blueMove)

## Uninstallation

To remove blueMove:

```bash
# Remove the app
rm -rf /Applications/blueMove.app

# Remove the config (optional)
rm -rf ~/.config/blueMove

# Remove from Login Items in System Settings if added
```

