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
	copyButton := widget.NewButtonWithIcon("Copy Your Address", theme.ContentCopyIcon(), func() {
		ui.copyMyAddress()
	})

	checkPortButton := widget.NewButton("Check Port", func() {
		ui.checkPortAccessibility()
	})

	buttonsRow := container.NewHBox(copyButton, checkPortButton)

	return container.NewVBox(buttonsRow)
}

// createP2PSettingsSection creates the P2P settings section
func (ui *UI) createP2PSettingsSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Connection Settings")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// P2P enable/disable checkbox
	ui.p2pEnabledCheck = widget.NewCheck("Enable P2P", func(checked bool) {
		ui.toggleP2P(checked)
	})

	// Listen port with background
	portLabel := widget.NewLabel("Listen Port:")
	portBg := canvas.NewRectangle(color.RGBA{R: 50, G: 50, B: 50, A: 255})
	portBg.SetMinSize(fyne.NewSize(100, 30))
	ui.portEntry = widget.NewEntry()
	portWrapper := container.NewStack(portBg, ui.portEntry)
	portRow := container.NewHBox(portLabel, portWrapper)

	// Checkboxes
	ui.autoConnectCheck = widget.NewCheck("Auto-connect to all known peers on startup", nil)
	ui.autoProfileExCheck = widget.NewCheck("Auto-exchange profile lists with connected peers", nil)

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
		ui.p2pEnabledCheck,
		portRow,
		ui.autoConnectCheck,
		ui.autoProfileExCheck,
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
	settings.EnableAutoConnect = ui.autoConnectCheck.Checked
	settings.EnableAutoProfileEx = ui.autoProfileExCheck.Checked

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
	ui.autoConnectCheck.SetChecked(settings.EnableAutoConnect)
	ui.autoProfileExCheck.SetChecked(settings.EnableAutoProfileEx)
}

// loadP2PSettings loads P2P settings
func (ui *UI) loadP2PSettings() {
	if ui.p2pUI == nil {
		return
	}

	settings := ui.p2pUI.GetSettings()

	port := settings.ListenPort
	if port <= 0 {
		port = 8080
	}

	// Обновляем UI в главном потоке
	fyne.CurrentApp().Driver().DoFromGoroutine(func() {
		ui.portEntry.SetText(fmt.Sprintf("%d", port))
	}, false)

	status := ui.p2pUI.GetStatus()
	fyne.CurrentApp().Driver().DoFromGoroutine(func() {
		ui.p2pEnabledCheck.SetChecked(status.IsRunning)
	}, false)

	ui.autoConnectCheck.SetChecked(settings.EnableAutoConnect)
	ui.autoProfileExCheck.SetChecked(settings.EnableAutoProfileEx)

	ui.updateAddressDisplay()
}

// toggleP2P enables or disables P2P functionality
func (ui *UI) toggleP2P(enabled bool) {
	if ui.p2pUI == nil {
		return
	}

	if enabled {
		go func() {
			if err := ui.p2pUI.Start(); err != nil {
				log.Printf("[P2P] Error starting P2P: %v", err)
				fyne.CurrentApp().Driver().DoFromGoroutine(func() {
					ui.p2pEnabledCheck.SetChecked(false)
					ui.showErrorDialog("Error", "Failed to start P2P: "+err.Error())
				}, false)
			} else {
				log.Printf("[P2P] P2P started")
				ui.updateAddressDisplay()
			}
		}()
	} else {
		go func() {
			if err := ui.p2pUI.Stop(); err != nil {
				log.Printf("[P2P] Error stopping P2P: %v", err)
				fyne.CurrentApp().Driver().DoFromGoroutine(func() {
					ui.p2pEnabledCheck.SetChecked(true)
					ui.showErrorDialog("Error", "Failed to stop P2P: "+err.Error())
				}, false)
			} else {
				log.Printf("[P2P] P2P stopped")
				ui.updateAddressDisplay()
			}
		}()
	}
}

// updateAddressDisplay обновляет отображение адреса
func (ui *UI) updateAddressDisplay() {
	if ui.p2pUI == nil {
		return
	}

	// Загружаем публичный адрес асинхронно
	go func() {
		_ = ui.p2pUI.RefreshPublicAddress()
	}()
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
	clipboard := fyne.CurrentApp().Clipboard()
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

// showErrorDialog shows an error dialog
func (ui *UI) showErrorDialog(title, message string) {
	dialog.ShowError(fmt.Errorf("%s", message), ui.window)
}

// showInfoDialog shows an information dialog
func (ui *UI) showInfoDialog(title, message string) {
	dialog.ShowInformation(title, message, ui.window)
}
