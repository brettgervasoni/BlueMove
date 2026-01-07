package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"blueMove/internal/blueutil"
)

// Config holds the parsed configuration
type Config struct {
	Devices   map[string]string // label -> MAC address
	WakeOnLAN string            // MAC address for wake-on-LAN target
	Path      string            // path to config file
}

// Load reads and parses the blueMove.conf file
// It looks for the config in current directory first, then ~/.config/blueMove/
// If no config exists, it attempts to generate one from connected keyboard/mouse devices
func Load() (*Config, error) {
	// Try current directory first
	configPath := "blueMove.conf"
	configExists := true

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Try ~/.config/blueMove/blueMove.conf
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot get home directory: %w", err)
		}
		configPath = filepath.Join(home, ".config", "blueMove", "blueMove.conf")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			configExists = false
		}
	}

	// If no config exists, try to generate one from connected devices
	// Always generate in ~/.config/blueMove/blueMove.conf
	if !configExists {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot get home directory: %w", err)
		}
		configPath = filepath.Join(home, ".config", "blueMove", "blueMove.conf")
		if err := generateConfigFromConnectedDevices(configPath); err != nil {
			return nil, fmt.Errorf("config file not found and could not auto-generate: %w", err)
		}
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open config file: %w", err)
	}
	defer file.Close()

	cfg := &Config{
		Devices: make(map[string]string),
		Path:    configPath,
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Parse label=MAC format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format at line %d: %s (expected label=MAC)", lineNum, line)
		}

		label := strings.TrimSpace(parts[0])
		mac := strings.TrimSpace(parts[1])

		if label == "" || mac == "" {
			return nil, fmt.Errorf("empty label or MAC at line %d", lineNum)
		}

		// Check if this is a wake-on-LAN configuration
		if label == "wakeonlan" {
			cfg.WakeOnLAN = mac
		} else {
			cfg.Devices[label] = mac
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	if len(cfg.Devices) == 0 {
		return nil, fmt.Errorf("no devices found in config file")
	}

	return cfg, nil
}

// generateConfigFromConnectedDevices creates a config file from connected keyboard/mouse devices
func generateConfigFromConnectedDevices(configPath string) error {
	// Find blueutil
	bu, err := blueutil.Find()
	if err != nil {
		return fmt.Errorf("blueutil not found: %w", err)
	}

	// Get connected devices
	devices, err := bu.Connected()
	if err != nil {
		return fmt.Errorf("failed to get connected devices: %w", err)
	}

	// Filter for keyboard and mouse devices
	var filteredDevices []blueutil.InquiryDevice
	for _, device := range devices {
		nameLower := strings.ToLower(device.Name)
		if strings.Contains(nameLower, "keyboard") || strings.Contains(nameLower, "mouse") {
			filteredDevices = append(filteredDevices, device)
		}
	}

	if len(filteredDevices) == 0 {
		return fmt.Errorf("no keyboard or mouse devices currently connected")
	}

	// Ensure the directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create the config file
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	// Write header comment
	fmt.Fprintln(file, "# blueMove configuration file")
	fmt.Fprintln(file, "# Auto-generated from connected devices")
	fmt.Fprintln(file, "# Format: label=MAC_ADDRESS")
	fmt.Fprintln(file, "#")
	fmt.Fprintln(file, "# Optional: To enable Wake-on-LAN, add the following line:")
	fmt.Fprintln(file, "# wakeonlan=AA-BB-CC-DD-EE-FF")
	fmt.Fprintln(file)

	// Write device entries
	for _, device := range filteredDevices {
		label := generateLabel(device.Name)
		fmt.Fprintf(file, "%s=%s\n", label, device.Address)
	}

	return nil
}

// generateLabel creates a simple label from a device name
func generateLabel(name string) string {
	// Convert to lowercase and replace spaces with hyphens
	label := strings.ToLower(name)
	label = strings.ReplaceAll(label, " ", "-")
	label = strings.ReplaceAll(label, "'", "")

	// Remove common possessive patterns like "brett's" -> "brett"
	label = strings.ReplaceAll(label, "s-", "-")

	// Simplify common patterns
	if strings.Contains(label, "keyboard") {
		label = "keyboard"
	} else if strings.Contains(label, "mouse") {
		label = "mouse"
	}

	return label
}
