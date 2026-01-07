package blueutil

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Blueutil wraps the blueutil command-line utility
type Blueutil struct {
	path string
}

// Find locates the blueutil binary
func Find() (*Blueutil, error) {
	// Check common installation paths
	paths := []string{
		"/opt/homebrew/bin/blueutil",
		"/usr/local/bin/blueutil",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return &Blueutil{path: p}, nil
		}
	}

	// Try PATH as last resort
	p, err := exec.LookPath("blueutil")
	if err != nil {
		return nil, fmt.Errorf("blueutil not found in common paths or PATH")
	}

	return &Blueutil{path: p}, nil
}

// IsConnected checks if a device is connected
func (b *Blueutil) IsConnected(mac string) (bool, error) {
	output, err := b.run("--is-connected", mac)
	if err != nil {
		return false, err
	}

	result := strings.TrimSpace(output)
	return result == "1", nil
}

// Pair pairs with a device
// If the device is already paired, it will be unpaired first, this can occur because bluetooth is flakey
func (b *Blueutil) Pair(mac string) error {
	// Check if device is already paired
	output, err := b.run("--paired", "--format", "json-pretty")
	if err != nil {
		// If we can't get paired devices, proceed with pairing anyway
		_, err := b.run("--pair", mac)
		return err
	}

	// Parse JSON output to check if this device is already paired
	var pairedDevices []InquiryDevice
	if err := json.Unmarshal([]byte(output), &pairedDevices); err == nil {
		// Check if our target device is in the paired list
		normalizedMac := NormalizeMac(mac)
		for _, device := range pairedDevices {
			if NormalizeMac(device.Address) == normalizedMac {
				// Device is already paired, unpair it first
				if err := b.Unpair(mac); err != nil {
					return fmt.Errorf("failed to unpair already-paired device: %w", err)
				}
				break
			}
		}
	}

	// Now pair the device
	_, err = b.run("--pair", mac)
	return err
}

// Unpair unpairs a device
func (b *Blueutil) Unpair(mac string) error {
	_, err := b.run("--unpair", mac)
	return err
}

// Connect connects to a device
func (b *Blueutil) Connect(mac string) error {
	_, err := b.run("--connect", mac)
	return err
}

// Disconnect disconnects from a device
func (b *Blueutil) Disconnect(mac string) error {
	_, err := b.run("--disconnect", mac)
	return err
}

// InquiryDevice represents a device found during inquiry
type InquiryDevice struct {
	Address          string `json:"address"`
	Name             string `json:"name"`
	RecentAccessDate string `json:"recentAccessDate"` // ISO 8601 timestamp
	Favourite        bool   `json:"favourite"`
	Connected        bool   `json:"connected"`
	Paired           bool   `json:"paired"`
}

// Inquiry scans for nearby Bluetooth devices
// duration is in seconds (default 10 if set to 0)
func (b *Blueutil) Inquiry(duration int) ([]InquiryDevice, error) {
	args := []string{"--inquiry"}
	if duration > 0 {
		args = append(args, fmt.Sprintf("%d", duration))
	}
	args = append(args, "--format", "json-pretty")

	output, err := b.run(args...)
	if err != nil {
		return nil, err
	}

	// Parse JSON output
	var devices []InquiryDevice
	if err := json.Unmarshal([]byte(output), &devices); err != nil {
		return nil, fmt.Errorf("failed to parse inquiry JSON: %w", err)
	}

	return devices, nil
}

// Connected returns all currently connected Bluetooth devices
func (b *Blueutil) Connected() ([]InquiryDevice, error) {
	output, err := b.run("--connected", "--format", "json-pretty")
	if err != nil {
		return nil, err
	}

	// Parse JSON output
	var devices []InquiryDevice
	if err := json.Unmarshal([]byte(output), &devices); err != nil {
		return nil, fmt.Errorf("failed to parse connected devices JSON: %w", err)
	}

	return devices, nil
}

// NormalizeMac normalizes a MAC address to lowercase with dashes
func NormalizeMac(mac string) string {
	mac = strings.ToLower(strings.TrimSpace(mac))
	// Replace colons with dashes for consistency
	mac = strings.ReplaceAll(mac, ":", "-")
	return mac
}

// run executes blueutil with the given arguments
func (b *Blueutil) run(args ...string) (string, error) {
	cmd := exec.Command(b.path, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", fmt.Errorf("blueutil %v failed: %w: %s", args, err, string(output))
	}

	return string(output), nil
}
