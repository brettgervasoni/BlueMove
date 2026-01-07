package main

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"

	"blueMove/internal/assets"
	"blueMove/internal/blueutil"
	"blueMove/internal/config"
)

// Device represents a Bluetooth device to manage
type Device struct {
	Label          string
	MAC            string
	Connected      bool
	MenuItem       *systray.MenuItem
	DisconnectTime time.Time // when device was manually disconnected
}

// App is the main application state
type App struct {
	bu                    *blueutil.Blueutil
	cfg                   *config.Config
	devices               []*Device
	mu                    sync.Mutex // the mutex is used to protect the app state (devices slice, states, ui status) from concurrent access
	toggleButton          *systray.MenuItem
	wakeMenuItem          *systray.MenuItem
	disconnectAndWakeItem *systray.MenuItem
	disconnectAndSleepItem *systray.MenuItem
	monitorQuit           chan struct{}
	autoConnecting        bool // prevents overlapping auto-connect operations
}

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	// Set the menu bar icon
	if len(assets.IconData) > 0 {
		systray.SetIcon(assets.IconData)
	}
	systray.SetTooltip("blueMove - Bluetooth device switcher")

	app := &App{
		monitorQuit: make(chan struct{}),
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		beeep.Alert("blueMove Error", fmt.Sprintf("Failed to load config: %v", err), "")
		quitItem := systray.AddMenuItem("Quit (Config Error)", "")
		go func() {
			<-quitItem.ClickedCh
			systray.Quit()
		}()
		return
	}
	app.cfg = cfg

	// Find blueutil
	bu, err := blueutil.Find()
	if err != nil {
		log.Printf("Failed to find blueutil: %v", err)
		beeep.Alert("blueMove Error", "blueutil not found. Install with: brew install blueutil", "")
		quitItem := systray.AddMenuItem("Quit (blueutil not found)", "")
		go func() {
			<-quitItem.ClickedCh
			systray.Quit()
		}()
		return
	}
	app.bu = bu

	// Initialize devices
	for label, mac := range cfg.Devices {
		device := &Device{
			Label: label,
			MAC:   mac,
		}
		app.devices = append(app.devices, device)
	}

	// Build menu
	app.buildMenu()

	// Start monitoring loop
	go app.monitorLoop()

	// Initial status update
	go app.updateAllStatus()
}

func onExit() {
	log.Println("Exiting blueMove")
}

func (app *App) buildMenu() {
	// Toggle button (will update label dynamically)
	app.toggleButton = systray.AddMenuItem("Loading...", "Toggle all devices")

	// Disconnect All & Wake Host (only if Wake-on-LAN is configured)
	if app.cfg.WakeOnLAN != "" {
		app.disconnectAndWakeItem = systray.AddMenuItem("Disconnect All & Wake Host", "Disconnect all devices and send Wake-on-LAN packet")
	}

	// Disconnect All & Sleep
	app.disconnectAndSleepItem = systray.AddMenuItem("Disconnect All & Sleep", "Disconnect all devices and put this Mac to sleep")

	systray.AddSeparator()

	// Individual device items
	for _, device := range app.devices {
		title := fmt.Sprintf("%s (%s)", capitalize(device.Label), device.MAC)
		device.MenuItem = systray.AddMenuItemCheckbox(title, "Toggle this device", false)

		// Handle individual device clicks
		go func(d *Device) {
			for {
				<-d.MenuItem.ClickedCh
				go app.toggleDevice(d)
			}
		}(device)
	}

	systray.AddSeparator()

	// Wake Host menu item (always shown)
	if app.cfg.WakeOnLAN != "" {
		app.wakeMenuItem = systray.AddMenuItem(fmt.Sprintf("Wake Host (%s)", app.cfg.WakeOnLAN), "Send Wake-on-LAN packet")
	} else {
		app.wakeMenuItem = systray.AddMenuItem("Wake Host (not configured)", "Wake-on-LAN not configured")
		app.wakeMenuItem.Disable()
	}

	systray.AddSeparator()

	// Quit button
	quitItem := systray.AddMenuItem("Quit", "Quit blueMove")

	// Handle menu clicks
	go func() {
		for {
			select {
			case <-app.toggleButton.ClickedCh:
				go app.toggleAll()
			case <-quitItem.ClickedCh:
				close(app.monitorQuit)
				systray.Quit()
				return
			}
		}
	}()

	// Handle Wake-on-LAN menu clicks (only if configured)
	if app.cfg.WakeOnLAN != "" {
		go func() {
			for {
				select {
				case <-app.wakeMenuItem.ClickedCh:
					go app.sendWakeOnLAN()
				case <-app.disconnectAndWakeItem.ClickedCh:
					go app.wakeAndDisconnectAll()
				}
			}
		}()
	}

	// Handle Disconnect & Sleep menu clicks
	go func() {
		for {
			<-app.disconnectAndSleepItem.ClickedCh
			go app.disconnectAllAndSleep()
		}
	}()
}

func (app *App) toggleAll() {
	app.mu.Lock()
	defer app.mu.Unlock()

	// Check if all devices are connected
	allConnected := true
	for _, device := range app.devices {
		if !device.Connected {
			allConnected = false
			break
		}
	}

	if allConnected {
		// Disconnect all
		log.Println("Disconnecting all devices...")
		app.disconnectAll()
	} else {
		// Connect all
		log.Println("Connecting all devices...")
		app.connectAll()
	}
}

func (app *App) toggleDevice(device *Device) {
	app.mu.Lock()
	defer app.mu.Unlock()

	if device.Connected {
		// Disconnect this device
		log.Printf("Disconnecting %s...", device.Label)
		app.disconnectDevice(device)
	} else {
		// Connect this device
		log.Printf("Connecting %s...", device.Label)
		app.connectDevice(device)
	}
}

func (app *App) disconnectAll() {
	// This function is called with lock held
	for _, device := range app.devices {
		if device.Connected {
			app.disconnectDevice(device)
		}
	}
}

func (app *App) connectAll() {
	// This function is called with lock held
	for _, device := range app.devices {
		if !device.Connected {
			app.connectDevice(device)
		}
	}
}

func (app *App) disconnectDevice(device *Device) {
	// This function is called with lock held

	// Unpair the device
	if err := app.bu.Unpair(device.MAC); err != nil {
		log.Printf("Failed to unpair %s: %v", device.Label, err)
		beeep.Alert("blueMove", fmt.Sprintf("Failed to disconnect %s", device.Label), "")
		return
	}

	device.Connected = false
	device.MenuItem.Uncheck()
	device.DisconnectTime = time.Now()

	log.Printf("Disconnected %s, auto-connect blocked for 120 seconds", device.Label)
	beeep.Notify("blueMove", fmt.Sprintf("Disconnected %s", device.Label), "")

	app.updateToggleButton()
}

func (app *App) connectDevice(device *Device) {
	// This function is called with lock held

	// Pair the device
	if err := app.bu.Pair(device.MAC); err != nil {
		log.Printf("Failed to pair %s: %v", device.Label, err)
		beeep.Alert("blueMove", fmt.Sprintf("Failed to pair %s", device.Label), "")
		return
	}

	time.Sleep(2 * time.Second)

	// Connect the device
	if err := app.bu.Connect(device.MAC); err != nil {
		log.Printf("Failed to connect %s: %v", device.Label, err)
		beeep.Alert("blueMove", fmt.Sprintf("Failed to connect %s", device.Label), "")
		return
	}

	// Verify connection with retries (give it more time to establish)
	var connected bool
	var err error
	for i := 0; i < 3; i++ {
		time.Sleep(2 * time.Second)
		connected, err = app.bu.IsConnected(device.MAC)
		if err != nil {
			log.Printf("Failed to check connection status for %s (attempt %d): %v", device.Label, i+1, err)
		}
		if connected {
			break
		}
		log.Printf("Connection check %d/3 for %s: not yet connected", i+1, device.Label)
	}

	if connected {
		device.Connected = true
		device.MenuItem.Check()
		log.Printf("Connected %s", device.Label)
		beeep.Notify("blueMove", fmt.Sprintf("Connected %s", device.Label), "")
	} else {
		log.Printf("Connection to %s did not succeed after verification", device.Label)
		beeep.Alert("blueMove", fmt.Sprintf("Failed to connect %s", device.Label), "")
	}

	app.updateToggleButton()
}

func (app *App) monitorLoop() {
	// Poll every 5 seconds to:
	// 1. Update connection status
	// 2. Auto-connect disconnected devices that are available
	// Note: blueutil handles concurrent inquiry calls gracefully (returns quickly if already scanning)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			app.updateAllStatus()
			app.autoConnectAvailable()
		case <-app.monitorQuit:
			return
		}
	}
}

func (app *App) updateAllStatus() {
	app.mu.Lock()
	defer app.mu.Unlock()

	for _, device := range app.devices {
		connected, err := app.bu.IsConnected(device.MAC)
		if err != nil {
			log.Printf("Failed to check status for %s: %v", device.Label, err)
			continue
		}

		if connected != device.Connected {
			device.Connected = connected
			if connected {
				device.MenuItem.Check()
			} else {
				device.MenuItem.Uncheck()
			}
			log.Printf("Status changed for %s: connected=%v", device.Label, connected)
		}
	}

	app.updateToggleButton()
}

func (app *App) autoConnectAvailable() {
	// Check if already running an auto-connect operation
	app.mu.Lock()
	if app.autoConnecting {
		log.Println("Auto-connect already in progress, skipping")
		app.mu.Unlock()
		return
	}
	app.autoConnecting = true
	app.mu.Unlock()

	// Ensure we clear the flag when done
	defer func() {
		time.Sleep(3 * time.Second) // breathing room between inquiry scans, not needed, just a delay
		app.mu.Lock()
		app.autoConnecting = false
		app.mu.Unlock()
	}()

	// Check if all devices are already connected
	app.mu.Lock()
	allConnected := true
	for _, device := range app.devices {
		if !device.Connected {
			allConnected = false
			break
		}
	}
	app.mu.Unlock()

	// If all devices are connected, skip inquiry and sit idle
	if allConnected {
		// log.Println("All devices connected, skipping inquiry scan")
		return
	}

	// Run inquiry scan to find available devices
	log.Println("Running inquiry scan to find disconnected devices...")
	devices, err := app.bu.Inquiry(10)
	if err != nil {
		log.Printf("Inquiry scan failed: %v", err)
		return
	}

	// Build a set of available devices for quick lookup
	availableDevices := make(map[string]struct{})
	for _, dev := range devices {
		normalizedMAC := blueutil.NormalizeMac(dev.Address)
		availableDevices[normalizedMAC] = struct{}{}
		log.Printf("Inquiry found device: %s (%s)", dev.Name, dev.Address)
	}

	// Collect devices that need connecting (while holding lock)
	app.mu.Lock()
	var devicesToConnect []*Device
	for _, device := range app.devices {
		// Skip if already connected
		if device.Connected {
			continue
		}

		// Skip if within 120 seconds of manual disconnect
		if !device.DisconnectTime.IsZero() && time.Since(device.DisconnectTime) < 120*time.Second {
			remainingTime := 120 - int(time.Since(device.DisconnectTime).Seconds())
			log.Printf("Skipping %s: still in cooldown period (%d seconds remaining)", device.Label, remainingTime)
			continue
		}

		// Check if device is available via inquiry
		normalizedMAC := blueutil.NormalizeMac(device.MAC)
		if _, exists := availableDevices[normalizedMAC]; exists {
			devicesToConnect = append(devicesToConnect, device)
			// } else {
			// 	log.Printf("Device %s not found in inquiry scan, skipping", device.Label)
		}
	}
	app.mu.Unlock()

	// Connect devices without holding the lock
	for _, device := range devicesToConnect {
		log.Printf("Auto-connecting %s (found in inquiry scan)...", device.Label)

		// Try to pair
		if err := app.bu.Pair(device.MAC); err != nil {
			log.Printf("Auto-connect pair failed for %s: %v", device.Label, err)
			continue
		}

		time.Sleep(2 * time.Second)

		// Try to connect
		if err := app.bu.Connect(device.MAC); err != nil {
			log.Printf("Auto-connect failed for %s: %v", device.Label, err)
			continue
		}

		// Verify connection with retries in case it doesn't have a 'connected' status right away
		var connected bool
		var checkErr error
		for j := 0; j < 3; j++ {
			time.Sleep(2 * time.Second)

			connected, checkErr = app.bu.IsConnected(device.MAC)
			if checkErr != nil {
				log.Printf("Auto-connect verification failed for %s (attempt %d): %v", device.Label, j+1, checkErr)
			}

			if connected {
				break
			}

			log.Printf("Auto-connect check %d/3 for %s: not yet connected", j+1, device.Label)
		}

		if connected {
			app.mu.Lock()
			device.Connected = true
			device.MenuItem.Check()
			app.mu.Unlock()
			log.Printf("Auto-connected %s", device.Label)
			beeep.Notify("blueMove", fmt.Sprintf("Auto-connected %s", device.Label), "")
		} else {
			log.Printf("Auto-connect to %s did not succeed after verification", device.Label)
		}
	}

	// Update toggle button
	app.mu.Lock()
	app.updateToggleButton()
	app.mu.Unlock()
}

func (app *App) updateToggleButton() {
	// This function is called with lock held
	allConnected := true
	for _, device := range app.devices {
		if !device.Connected {
			allConnected = false
			break
		}
	}

	if allConnected {
		app.toggleButton.SetTitle("Disconnect All")
	} else {
		app.toggleButton.SetTitle("Connect All")
	}
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// sendWakeOnLAN sends a Wake-on-LAN magic packet to the configured host
func (app *App) sendWakeOnLAN() {
	if app.cfg.WakeOnLAN == "" {
		log.Println("Wake-on-LAN: no target MAC configured")
		return
	}

	// Wake-on-LAN magic packet creation and sending
	// The magic packet consists of:
	// - 6 bytes of 0xFF
	// - 16 repetitions of the target MAC address (6 bytes each)
	// Total: 102 bytes
	// Send via UDP to broadcast address (255.255.255.255) on port 9

	mac, _ := net.ParseMAC(app.cfg.WakeOnLAN)
	packet := make([]byte, 102)
	copy(packet, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}) // preamble of a wake on lan packet
	for i := 0; i < 16; i++ {                                // copy the mac into the packet 16 times
		copy(packet[6+i*6:], mac)
	}

	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: 9,
	})

	if err != nil {
		log.Printf("Failed to dial UDP: %v", err)
		return
	}
	defer conn.Close()

	conn.Write(packet)

	beeep.Notify("blueMove", fmt.Sprintf("Wake-on-LAN packet sent to %s", app.cfg.WakeOnLAN), "")
}

// wakeAndDisconnectAll sends a Wake-on-LAN packet and then disconnects all Bluetooth devices
func (app *App) wakeAndDisconnectAll() {
	if app.cfg.WakeOnLAN == "" {
		log.Println("Wake & Disconnect All: no WoL target MAC configured")
		beeep.Alert("blueMove", "Wake-on-LAN not configured", "")
		return
	}

	// Send Wake-on-LAN packet first
	log.Printf("Wake & Disconnect All: Sending Wake-on-LAN packet to %s", app.cfg.WakeOnLAN)
	app.sendWakeOnLAN()

	// Wait a moment before disconnecting
	time.Sleep(1 * time.Second)

	// Disconnect all devices
	log.Println("Wake & Disconnect All: disconnecting all devices...")
	app.mu.Lock()
	app.disconnectAll()
	app.mu.Unlock()

	log.Println("Wake & Disconnect All: completed")
}

// disconnectAllAndSleep disconnects all Bluetooth devices and puts the Mac to sleep
func (app *App) disconnectAllAndSleep() {
	log.Println("Disconnect All & Sleep: disconnecting all devices...")
	
	// Disconnect all devices
	app.mu.Lock()
	app.disconnectAll()
	app.mu.Unlock()

	// Wait a moment for disconnections to complete
	time.Sleep(2 * time.Second)

	// Put the Mac to sleep using pmset
	log.Println("Disconnect All & Sleep: putting Mac to sleep...")
	cmd := exec.Command("pmset", "sleepnow")
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to put Mac to sleep: %v", err)
		beeep.Alert("blueMove", "Failed to put Mac to sleep", "")
		return
	}

	log.Println("Disconnect All & Sleep: completed")
}
