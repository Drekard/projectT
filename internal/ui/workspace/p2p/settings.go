// Package p2p contains the connection tab component
package p2p

import (
	"fmt"
	"image/color"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createSettingsTab creates the settings tab
func (ui *UI) createSettingsTab() fyne.CanvasObject {
	// === Your Address ===
	addressSection := ui.createAddressSection()

	// === Connection Settings ===
	settingsSection := ui.createP2PSettingsSection()

	content := container.NewVBox(
		widget.NewSeparator(),
		addressSection,
		widget.NewSeparator(),
		settingsSection,
	)

	return container.NewScroll(content)
}

// createAddressSection creates the address section
func (ui *UI) createAddressSection() *fyne.Container {
	label := widget.NewLabel("Your address: ")
	copyButton := widget.NewButtonWithIcon("Copy Address", theme.ContentCopyIcon(), func() {
		ui.copyMyAddress()
	})

	checkPortButton := widget.NewButton("Check Port", func() {
		ui.checkPortAccessibility()
	})

	// Button to show local addresses
	showLocalButton := widget.NewButton("Local Addresses", func() {
		ui.showLocalAddresses()
	})

	addressRow := container.NewHBox(label, copyButton)
	buttonsRow := container.NewHBox(checkPortButton, showLocalButton)

	return container.NewVBox(addressRow, buttonsRow)
}

// createP2PSettingsSection creates the P2P settings section
func (ui *UI) createP2PSettingsSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Connection Settings")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Listen port with background
	portLabel := widget.NewLabel("Listen Port:")
	portBg := canvas.NewRectangle(color.RGBA{R: 50, G: 50, B: 50, A: 255})
	portBg.SetMinSize(fyne.NewSize(100, 30))
	ui.portEntry = widget.NewEntry()
	portWrapper := container.NewStack(portBg, ui.portEntry)
	portRow := container.NewHBox(portLabel, portWrapper)

	// Setting checkboxes
	ui.natPortMapCheck = widget.NewCheck("NAT Port Mapping (UPnP/NAT-PMP)", nil)
	ui.relayCheck = widget.NewCheck("Relay (NAT traversal)", nil)
	ui.autoRelayCheck = widget.NewCheck("Auto-detect Relay", nil)
	ui.dhtCheck = widget.NewCheck("DHT (global discovery)", nil)
	ui.mdnsCheck = widget.NewCheck("mDNS (local network)", nil)
	ui.helperModeCheck = widget.NewCheck("Helper Mode", nil)

	// STUN server with background
	stunLabel := widget.NewLabel("STUN Server:")
	stunBg := canvas.NewRectangle(color.RGBA{R: 50, G: 50, B: 50, A: 255})
	stunBg.SetMinSize(fyne.NewSize(200, 30))
	ui.stunServerEntry = widget.NewEntry()
	stunWrapper := container.NewStack(stunBg, ui.stunServerEntry)
	stunRow := container.NewHBox(stunLabel, stunWrapper)

	// Save and reset buttons
	saveBtn := widget.NewButtonWithIcon("Save", theme.ConfirmIcon(), func() {
		ui.saveP2PSettings()
	})
	saveBtn.Importance = widget.HighImportance

	resetBtn := widget.NewButtonWithIcon("Reset", theme.CancelIcon(), func() {
		ui.resetP2PSettings()
	})

	buttonsRow := container.NewHBox(saveBtn, resetBtn)

	return container.NewVBox(
		sectionTitle,
		portRow,
		ui.natPortMapCheck,
		ui.relayCheck,
		ui.autoRelayCheck,
		ui.dhtCheck,
		ui.mdnsCheck,
		stunRow,
		ui.helperModeCheck,
		widget.NewSeparator(),
		buttonsRow,
	)
}

// saveP2PSettings saves P2P settings
func (ui *UI) saveP2PSettings() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Error", "P2P service not initialized")
		return
	}

	// Get current settings
	settings := ui.p2pUI.GetSettings()

	// Update from UI
	if ui.portEntry.Text != "" {
		if _, err := fmt.Sscanf(ui.portEntry.Text, "%d", &settings.ListenPort); err != nil {
			log.Printf("Error parsing port: %v", err)
		}
	}
	settings.EnableNATPortMap = ui.natPortMapCheck.Checked
	settings.EnableRelay = ui.relayCheck.Checked
	settings.EnableAutoRelay = ui.autoRelayCheck.Checked
	settings.EnableDHT = ui.dhtCheck.Checked
	settings.EnableMDNS = ui.mdnsCheck.Checked
	settings.STUNServer = ui.stunServerEntry.Text
	settings.EnableHelperMode = ui.helperModeCheck.Checked

	// Save settings via UpdateSettings
	if err := ui.p2pUI.UpdateSettings(settings); err != nil {
		ui.showErrorDialog("Error", "Failed to save settings: "+err.Error())
		return
	}

	ui.showInfoDialog("Success", "Settings saved. Restart P2P to apply.")
}

// resetP2PSettings resets settings to default values
func (ui *UI) resetP2PSettings() {
	if ui.p2pUI == nil {
		return
	}

	settings := ui.p2pUI.GetSettings()
	ui.portEntry.SetText(fmt.Sprintf("%d", settings.ListenPort))
	ui.natPortMapCheck.SetChecked(settings.EnableNATPortMap)
	ui.relayCheck.SetChecked(settings.EnableRelay)
	ui.autoRelayCheck.SetChecked(settings.EnableAutoRelay)
	ui.dhtCheck.SetChecked(settings.EnableDHT)
	ui.mdnsCheck.SetChecked(settings.EnableMDNS)
	ui.stunServerEntry.SetText(settings.STUNServer)
	ui.helperModeCheck.SetChecked(settings.EnableHelperMode)
}

// loadP2PSettings loads P2P settings
func (ui *UI) loadP2PSettings() {
	if ui.p2pUI == nil {
		log.Printf("[loadP2PSettings] p2pUI == nil")
		return
	}

	log.Printf("[loadP2PSettings] Loading settings...")
	settings := ui.p2pUI.GetSettings()
	log.Printf("[loadP2PSettings] Got settings: Port=%d, NAT=%v, Relay=%v",
		settings.ListenPort, settings.EnableNATPortMap, settings.EnableRelay)

	// Set port (if 0 or not set - use 8080)
	port := settings.ListenPort
	if port <= 0 {
		port = 8080
	}
	ui.portEntry.SetText(fmt.Sprintf("%d", port))

	// Set checkboxes
	ui.natPortMapCheck.SetChecked(settings.EnableNATPortMap)
	ui.relayCheck.SetChecked(settings.EnableRelay)
	ui.autoRelayCheck.SetChecked(settings.EnableAutoRelay)
	ui.dhtCheck.SetChecked(settings.EnableDHT)
	ui.mdnsCheck.SetChecked(settings.EnableMDNS)
	ui.stunServerEntry.SetText(settings.STUNServer)
	ui.helperModeCheck.SetChecked(settings.EnableHelperMode)

	log.Printf("[loadP2PSettings] Settings loaded")
}

// copyMyAddress copies my address to clipboard
func (ui *UI) copyMyAddress() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Error", "P2P service not initialized")
		return
	}

	addr, err := ui.p2pUI.GetPeerAddress()
	if err != nil {
		ui.showErrorDialog("Error", fmt.Sprintf("Failed to get address: %v", err))
		return
	}

	// Copy to clipboard
	clipboard := ui.window.Clipboard()
	clipboard.SetContent(addr)

	ui.showInfoDialog("Success", "Address copied to clipboard")
}

// checkPortAccessibility checks port accessibility
func (ui *UI) checkPortAccessibility() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Error", "P2P service not initialized")
		return
	}

	ui.showInfoDialog("Check", "Checking port... (requires external service)")
	// TODO: Implement port check via STUN or external service
}

// showLocalAddresses shows local addresses
func (ui *UI) showLocalAddresses() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Error", "P2P service not initialized")
		return
	}

	addrs := ui.p2pUI.GetLocalAddresses()
	if len(addrs) == 0 {
		ui.showInfoDialog("Local Addresses", "No local addresses found")
		return
	}

	// Format addresses for display
	addrsText := ""
	for i, addr := range addrs {
		addrsText += fmt.Sprintf("%d. %s\n", i+1, addr)
	}

	// Format clean addresses for copying (without numbering)
	addrsTextClean := ""
	for _, addr := range addrs {
		addrsTextClean += addr + "\n"
	}

	// Create label with addresses
	addrLabel := widget.NewLabel(addrsText)
	addrLabel.Wrapping = fyne.TextWrapBreak

	// Copy button
	copyButton := widget.NewButtonWithIcon("Copy", theme.ContentCopyIcon(), func() {
		// Copy all addresses to clipboard (without numbering)
		clipboard := ui.window.Clipboard()
		clipboard.SetContent(addrsTextClean)
		ui.showInfoDialog("Success", "Local addresses copied to clipboard")
	})

	// Create dialog with buttons
	content := container.NewVBox(addrLabel, copyButton)

	customDialog := dialog.NewCustom("Local Addresses", "Close", content, ui.window)
	customDialog.Show()
}

// showErrorDialog shows an error dialog
func (ui *UI) showErrorDialog(title, message string) {
	dialog.ShowError(fmt.Errorf("%s", message), ui.window)
}

// showInfoDialog shows an information dialog
func (ui *UI) showInfoDialog(title, message string) {
	dialog.ShowInformation(title, message, ui.window)
}
