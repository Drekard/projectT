// Package p2p contains the connection tab component
package p2p

import (
	"fmt"
	"image/color"
	"os"
	"time"

	network "projectT/internal/services/p2p/ui"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createProfilesTab creates the tab with connections and profiles
func (ui *UI) createProfilesTab() fyne.CanvasObject {
	createConnectSection := ui.createConnectByAddressSection()

	// === Connected Peers ===
	connectedSection := ui.createConnectedPeersSection()

	// === Discovered Peers ===
	discoveredSection := ui.createDiscoveredPeersSection()

	// === Profiles (from profiles table) ===
	profilesSection := ui.createProfilesSection()

	content := container.NewVBox(
		widget.NewSeparator(),
		createConnectSection,
		widget.NewSeparator(),
		connectedSection,
		widget.NewSeparator(),
		discoveredSection,
		widget.NewSeparator(),
		profilesSection,
	)

	return container.NewScroll(content)
}

// createConnectedPeersSection creates the connected peers section
func (ui *UI) createConnectedPeersSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Connected Peers")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	ui.connectedCountLabel = widget.NewLabel("0/50")
	ui.connectedCountLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Refresh button
	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		ui.loadConnectedPeers()
	})

	// Connect to all button
	connectAllBtn := widget.NewButtonWithIcon("Connect to All", theme.FolderOpenIcon(), func() {
		if ui.p2pUI == nil {
			ui.showErrorDialog("Error", "P2P service not initialized")
			return
		}
		ui.p2pUI.ConnectToAll()
		ui.showInfoDialog("Connecting", "Attempting to connect to all known peers...")
		time.AfterFunc(5*time.Second, func() {
			ui.loadConnectedPeers()
		})
	})

	headerRow := container.NewHBox(sectionTitle, ui.connectedCountLabel, refreshBtn, connectAllBtn)

	ui.connectedPeersList = container.NewVBox()

	return container.NewVBox(headerRow, ui.connectedPeersList)
}

// createDiscoveredPeersSection creates the discovered peers section
func (ui *UI) createDiscoveredPeersSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Discovered Peers (DHT/mDNS)")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Refresh button
	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		ui.loadDiscoveredPeers()
	})

	headerRow := container.NewHBox(sectionTitle, refreshBtn)

	ui.discoveredPeersList = container.NewVBox()

	return container.NewVBox(headerRow, ui.discoveredPeersList)
}

// createProfilesSection creates the profiles section
func (ui *UI) createProfilesSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Peer Profiles")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	ui.profilesCountLabel = widget.NewLabel("0")
	ui.profilesCountLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Refresh button
	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		ui.loadProfiles()
	})

	// Exchange profile lists button
	exchangeBtn := widget.NewButtonWithIcon("Exchange Profile Lists", theme.MailComposeIcon(), func() {
		if ui.p2pUI == nil {
			ui.showErrorDialog("Error", "P2P service not initialized")
			return
		}
		ui.p2pUI.ExchangeProfileLists()
		ui.showInfoDialog("Exchanging", "Requesting profile exchange with all connected peers...")
		time.AfterFunc(5*time.Second, func() {
			ui.loadProfiles()
		})
	})

	headerRow := container.NewHBox(sectionTitle, ui.profilesCountLabel, refreshBtn, exchangeBtn)

	ui.profilesList = container.NewVBox()

	return container.NewVBox(headerRow, ui.profilesList)
}

// createConnectByAddressSection creates the connect by address section
func (ui *UI) createConnectByAddressSection() *fyne.Container {
	sectionTitle := widget.NewLabel("Connect by Address")
	sectionTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Address input field
	connectAddressEntry := widget.NewEntry()
	connectAddressEntry.SetPlaceHolder("projectt:peerid@/ip4/.../tcp/.../p2p/...")
	connectAddressEntry.MultiLine = false

	// Connect button
	connectButton := widget.NewButtonWithIcon("Connect", theme.FolderIcon(), func() {
		addrStr := connectAddressEntry.Text
		if addrStr == "" {
			ui.showErrorDialog("Error", "Enter peer address")
			return
		}

		if ui.p2pUI == nil {
			ui.showErrorDialog("Error", "P2P service not initialized")
			return
		}

		err := ui.p2pUI.ConnectToContact(addrStr)
		if err != nil {
			ui.showErrorDialog("Error", fmt.Sprintf("Failed to connect: %v", err))
			return
		}

		ui.showInfoDialog("Connection", "Attempting to connect to peer...")
		connectAddressEntry.SetText("")

		// Update connected peers list after a few seconds
		time.AfterFunc(3*time.Second, func() {
			ui.loadConnectedPeers()
		})
	})
	connectAddressWrapper := canvas.NewRectangle(color.RGBA{R: 50, G: 50, B: 50, A: 255})
	connectAddressWrapper.SetMinSize(fyne.NewSize(450, 30))
	connect := container.NewStack(connectAddressWrapper, connectAddressEntry)

	return container.NewVBox(
		sectionTitle,
		container.NewHBox(connect, connectButton),
	)
}

// loadConnectedPeers loads the list of connected peers
func (ui *UI) loadConnectedPeers() {
	if ui.connectedPeersList == nil {
		return
	}

	ui.connectedPeersList.Objects = nil

	if ui.p2pUI == nil {
		emptyLabel := widget.NewLabel("P2P service not initialized")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.connectedPeersList.Add(emptyLabel)
		ui.connectedPeersList.Refresh()
		return
	}

	peers := ui.p2pUI.GetConnectedPeers()
	count := ui.p2pUI.GetConnectedPeersCount()

	// Обновляем счётчик
	if ui.connectedCountLabel != nil {
		ui.connectedCountLabel.SetText(fmt.Sprintf("%d/50", count))
	}

	if len(peers) == 0 {
		emptyLabel := widget.NewLabel("No connected peers")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.connectedPeersList.Add(emptyLabel)
	} else {
		for _, peer := range peers {
			peerItem := ui.createConnectedPeerItem(peer)
			ui.connectedPeersList.Add(peerItem)
		}
	}

	ui.connectedPeersList.Refresh()
}

// createConnectedPeerItem creates a connected peer item
func (ui *UI) createConnectedPeerItem(peer *network.PeerInfo) *fyne.Container {
	// Avatar icon
	var avatarIcon *canvas.Image
	profile, err := queries.GetProfileByPeerID(peer.PeerID)
	if err == nil && profile != nil && profile.AvatarPath != "" {
		if _, statErr := os.Stat(profile.AvatarPath); statErr == nil {
			avatarIcon = canvas.NewImageFromFile(profile.AvatarPath)
			avatarIcon.FillMode = canvas.ImageFillContain
			avatarIcon.SetMinSize(fyne.NewSize(40, 40))
		}
	}
	if avatarIcon == nil {
		avatarIcon = canvas.NewImageFromResource(theme.AccountIcon())
		avatarIcon.FillMode = canvas.ImageFillContain
		avatarIcon.SetMinSize(fyne.NewSize(40, 40))
	}

	// Peer name
	nameLabel := widget.NewLabel(peer.Username)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// PeerID (full)
	peerIDLabel := widget.NewLabel(fmt.Sprintf("PeerID: %s", peer.PeerID))
	peerIDLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Address (full)
	addressLabel := widget.NewLabel(fmt.Sprintf("Address: %s", peer.Address))
	addressLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Latency
	latencyLabel := widget.NewLabel(fmt.Sprintf("Ping: %d ms", peer.LatencyMs))
	latencyLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Status indicator
	statusInd := canvas.NewCircle(color.RGBA{R: 0, G: 255, B: 0, A: 255})

	// Chat button
	chatBtn := widget.NewButtonWithIcon("Chat", theme.MailComposeIcon(), func() {
		ui.openPeerChat(peer.PeerID, peer.Username)
	})

	// Profile button
	profileBtn := widget.NewButtonWithIcon("Profile", theme.AccountIcon(), func() {
		ui.openRemoteProfile(peer.PeerID)
	})

	// Add to contacts button
	addContactBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		ui.addConnectedPeerToContacts(peer.PeerID)
	})

	// Disconnect button
	disconnectBtn := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
		ui.disconnectFromPeer(peer.PeerID)
	})

	leftContent := container.NewHBox(avatarIcon, container.NewVBox(nameLabel, peerIDLabel, addressLabel))

	content := container.NewBorder(
		nil, nil,
		container.NewHBox(statusInd, leftContent),
		container.NewHBox(latencyLabel, chatBtn, profileBtn, addContactBtn, disconnectBtn),
		widget.NewSeparator(),
	)

	return content
}

// loadDiscoveredPeers loads the list of discovered peers
func (ui *UI) loadDiscoveredPeers() {
	if ui.discoveredPeersList == nil {
		return
	}

	ui.discoveredPeersList.Objects = nil

	if ui.p2pUI == nil {
		emptyLabel := widget.NewLabel("P2P service not initialized")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.discoveredPeersList.Add(emptyLabel)
		ui.discoveredPeersList.Refresh()
		return
	}

	discovered := ui.p2pUI.GetDiscoveredPeers()

	if len(discovered) == 0 {
		emptyLabel := widget.NewLabel("No discovered peers")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.discoveredPeersList.Add(emptyLabel)
	} else {
		for peerID, lastSeen := range discovered {
			peerItem := ui.createDiscoveredPeerItem(peerID, lastSeen)
			ui.discoveredPeersList.Add(peerItem)
		}
	}

	ui.discoveredPeersList.Refresh()
}

// createDiscoveredPeerItem creates a discovered peer item
func (ui *UI) createDiscoveredPeerItem(peerID string, lastSeen time.Time) *fyne.Container {
	// Peer name (use full PeerID as name)
	nameLabel := widget.NewLabel(fmt.Sprintf("Peer: %s", peerID))
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}
	nameLabel.Wrapping = fyne.TextWrapWord

	// Full PeerID
	peerIDLabel := widget.NewLabel(fmt.Sprintf("PeerID: %s", peerID))
	peerIDLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Last seen time (date and time)
	lastSeenLabel := widget.NewLabel(fmt.Sprintf("Seen: %s", lastSeen.Format("2006-01-02 15:04:05")))
	lastSeenLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Status indicator (yellow - not connected)
	statusInd := canvas.NewCircle(color.RGBA{R: 255, G: 255, B: 0, A: 255})

	// Connect button
	connectBtn := widget.NewButtonWithIcon("Connect", theme.FolderIcon(), func() {
		ui.connectToDiscoveredPeer(peerID)
	})

	content := container.NewBorder(
		nil, nil,
		container.NewHBox(statusInd, container.NewVBox(nameLabel, peerIDLabel)),
		container.NewHBox(lastSeenLabel, connectBtn),
		widget.NewSeparator(),
	)

	return content
}

// loadProfiles loads the list of profiles
func (ui *UI) loadProfiles() {
	if ui.profilesList == nil {
		return
	}

	ui.profilesList.Objects = nil

	if ui.p2pUI == nil {
		emptyLabel := widget.NewLabel("P2P service not initialized")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.profilesList.Add(emptyLabel)
		ui.profilesList.Refresh()
		return
	}

	profiles, _ := ui.p2pUI.GetProfiles()

	// Обновляем счётчик
	if ui.profilesCountLabel != nil {
		ui.profilesCountLabel.SetText(fmt.Sprintf("%d", len(profiles)))
	}

	if len(profiles) == 0 {
		emptyLabel := widget.NewLabel("No profiles")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.profilesList.Add(emptyLabel)
	} else {
		for _, profile := range profiles {
			profileItem := ui.createProfileItem(profile)
			ui.profilesList.Add(profileItem)
		}
	}

	ui.profilesList.Refresh()
}

// createProfileItem creates a profile item
func (ui *UI) createProfileItem(profile *models.Profile) *fyne.Container {
	// Avatar icon
	var avatarIcon *canvas.Image
	if profile.AvatarPath != "" {
		if _, statErr := os.Stat(profile.AvatarPath); statErr == nil {
			avatarIcon = canvas.NewImageFromFile(profile.AvatarPath)
			avatarIcon.FillMode = canvas.ImageFillContain
			avatarIcon.SetMinSize(fyne.NewSize(40, 40))
		}
	}
	if avatarIcon == nil {
		avatarIcon = canvas.NewImageFromResource(theme.AccountIcon())
		avatarIcon.FillMode = canvas.ImageFillContain
		avatarIcon.SetMinSize(fyne.NewSize(40, 40))
	}

	// Username
	usernameLabel := widget.NewLabel(profile.Username)
	usernameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// PeerID (full)
	peerIDLabel := widget.NewLabel(fmt.Sprintf("PeerID: %s", profile.PeerID))
	peerIDLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Updated time (date and time)
	cachedAtLabel := widget.NewLabel("")
	if profile.UpdatedAt.IsZero() {
		cachedAtLabel.SetText("Updated: never")
	} else {
		cachedAtLabel.SetText(fmt.Sprintf("Updated: %s", profile.UpdatedAt.Format("2006-01-02 15:04:05")))
	}
	cachedAtLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Status indicator (blue - cached profile)
	statusInd := canvas.NewCircle(color.RGBA{R: 0, G: 150, B: 255, A: 255})

	// Chat button
	chatBtn := widget.NewButtonWithIcon("Chat", theme.MailComposeIcon(), func() {
		ui.openPeerChat(profile.PeerID, profile.Username)
	})

	// Profile button
	profileBtn := widget.NewButtonWithIcon("Profile", theme.AccountIcon(), func() {
		ui.openRemoteProfile(profile.PeerID)
	})

	// Delete profile button
	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		ui.deleteProfile(profile.PeerID, profile.Username)
	})

	leftContent := container.NewHBox(avatarIcon, container.NewVBox(usernameLabel, peerIDLabel))

	content := container.NewBorder(
		nil, nil,
		container.NewHBox(statusInd, leftContent),
		container.NewHBox(cachedAtLabel, chatBtn, profileBtn, deleteBtn),
		widget.NewSeparator(),
	)

	return content
}

// openPeerChat opens a chat with a peer
func (ui *UI) openPeerChat(peerID, username string) {
	// First switch to the "Chats" tab
	if ui.onNavigate != nil {
		ui.onNavigate("chats")
	}

	// Then open chat with peer
	if ui.p2pUIProvider != nil {
		ui.p2pUIProvider.OpenPeerChat(peerID, username)
	}
}

// openRemoteProfile opens the remote profile of a peer
func (ui *UI) openRemoteProfile(peerID string) {
	if ui.p2pUIProvider != nil {
		ui.p2pUIProvider.OpenRemoteProfile(peerID)
	}
}

// addConnectedPeerToContacts adds a connected peer to contacts
func (ui *UI) addConnectedPeerToContacts(peerID string) {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Error", "P2P service not initialized")
		return
	}

	// Get peer address
	addrs := ui.p2pUI.GetPeerAddresses(peerID)
	if len(addrs) == 0 {
		ui.showErrorDialog("Error", "Peer address not found")
		return
	}

	// Add to contacts
	err := ui.p2pUI.AddContactByAddress(addrs[0], "")
	if err != nil {
		ui.showErrorDialog("Error", fmt.Sprintf("Failed to add to contacts: %v", err))
		return
	}

	ui.showInfoDialog("Success", "Peer added to contacts")
}

// disconnectFromPeer disconnects from a peer
func (ui *UI) disconnectFromPeer(peerID string) {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Error", "P2P service not initialized")
		return
	}

	err := ui.p2pUI.DisconnectPeer(peerID)
	if err != nil {
		ui.showErrorDialog("Error", fmt.Sprintf("Failed to disconnect: %v", err))
		return
	}

	// Update list after a few seconds
	time.AfterFunc(2*time.Second, func() {
		ui.loadConnectedPeers()
	})
}

// connectToDiscoveredPeer connects to a discovered peer
func (ui *UI) connectToDiscoveredPeer(peerID string) {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Error", "P2P service not initialized")
		return
	}

	// Get peer address from discovered
	// TODO: Get address from peerstore
	ui.showInfoDialog("Information", "Connecting to discovered peer requires implementation")
}

// deleteProfile deletes a peer profile from the database
func (ui *UI) deleteProfile(peerID, username string) {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Error", "P2P service not initialized")
		return
	}

	// Show confirmation dialog
	confirmDialog := dialog.NewConfirm(
		"Confirm Deletion",
		fmt.Sprintf("Are you sure you want to delete the profile '%s' from the database?", username),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			err := ui.p2pUI.DeleteProfile(peerID)
			if err != nil {
				ui.showErrorDialog("Error", fmt.Sprintf("Failed to delete profile: %v", err))
				return
			}

			ui.showInfoDialog("Success", "Profile deleted from database")

			// Update profiles list
			ui.loadProfiles()
		},
		ui.window,
	)
	confirmDialog.SetConfirmImportance(widget.DangerImportance)
	confirmDialog.Show()
}
