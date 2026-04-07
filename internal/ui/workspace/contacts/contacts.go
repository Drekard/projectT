// Package contacts contains the "Contacts" tab component
package contacts

import (
	"fmt"
	"image/color"
	"log"

	network "projectT/internal/services/p2p/ui"
	"projectT/internal/storage/database/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// UI represents the "Contacts" tab interface
type UI struct {
	content      *fyne.Container
	window       fyne.Window
	p2pUI        *network.UIP2P
	contactsUI   UIProvider
	contactsList *fyne.Container

	// Elements for adding a contact
	addressEntry  *widget.Entry
	usernameEntry *widget.Entry
}

// UIProvider interface for accessing UI functions
type UIProvider interface {
	OpenPeerChat(peerID, username string)
	OpenLocalChat()
	GetWindow() fyne.Window
}

// New creates a new "Contacts" tab UI
func New(contactsUI UIProvider) *UI {
	ui := &UI{
		contactsUI: contactsUI,
	}
	ui.content = ui.createContactsContent()
	return ui
}

// SetWindow sets the window
func (ui *UI) SetWindow(window fyne.Window) {
	ui.window = window
}

// SetP2PService sets the P2P service
func (ui *UI) SetP2PService(p2pUI *network.UIP2P) {
	ui.p2pUI = p2pUI
	ui.loadContactsList()
}

// GetContent returns the tab content
func (ui *UI) GetContent() fyne.CanvasObject {
	return ui.content
}

// Refresh refreshes the UI
func (ui *UI) Refresh() {
	ui.loadContactsList()
	if ui.content != nil {
		ui.content.Refresh()
	}
}

// createContactsContent creates the "Contacts" tab content
func (ui *UI) createContactsContent() *fyne.Container {
	// Contacts list
	ui.contactsList = container.NewVBox()

	// Divider for manual contact addition
	manualLabel := widget.NewLabel("Add contact by address")
	manualLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Address input field with limited width
	ui.addressEntry = widget.NewEntry()
	ui.addressEntry.SetPlaceHolder("projectt:peerid@/ip4/.../tcp/.../p2p/...")
	ui.addressEntry.MultiLine = false
	ui.addressEntry.Wrapping = fyne.TextWrapBreak

	// Limit address field width
	addressWrapper := container.NewGridWithColumns(2, ui.addressEntry)

	// Name input field with limited width
	ui.usernameEntry = widget.NewEntry()
	ui.usernameEntry.SetPlaceHolder("Contact name (optional)")
	ui.usernameEntry.MultiLine = false

	usernameEntryWrapper := canvas.NewRectangle(color.RGBA{R: 50, G: 50, B: 50, A: 255})
	usernameEntryWrapper.SetMinSize(fyne.NewSize(300, 30))

	username := container.NewStack(usernameEntryWrapper, ui.usernameEntry)

	// Add button (with limited width)
	addManualButton := widget.NewButtonWithIcon("Add Contact", theme.ContentAddIcon(), func() {
		ui.addContactByAddress()
	})

	// Limit name field width
	usernameWrapper := container.NewGridWithColumns(2, container.NewHBox(username, addManualButton))

	manualSection := container.NewVBox(
		manualLabel,
		addressWrapper,
		usernameWrapper,
	)

	content := container.NewVBox(
		widget.NewSeparator(),
		manualSection,
		widget.NewSeparator(),
		ui.contactsList,
	)

	// Background
	bg := canvas.NewRectangle(color.RGBA{R: 30, G: 30, B: 30, A: 0})

	return container.NewStack(bg, container.NewScroll(content))
}

// loadContactsList loads the contacts list from the database
func (ui *UI) loadContactsList() {
	if ui.contactsList == nil {
		return
	}

	ui.contactsList.Objects = nil

	if ui.p2pUI == nil {
		emptyLabel := widget.NewLabel("P2P service not initialized")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.contactsList.Add(emptyLabel)
		ui.contactsList.Refresh()
		return
	}

	// Get contacts from database
	contacts, err := ui.p2pUI.GetContacts()
	if err != nil {
		emptyLabel := widget.NewLabel(fmt.Sprintf("Error loading contacts: %v", err))
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.contactsList.Add(emptyLabel)
		ui.contactsList.Refresh()
		return
	}

	if len(contacts) == 0 {
		emptyLabel := widget.NewLabel("Contacts list is empty")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.contactsList.Add(emptyLabel)
	} else {
		for _, contact := range contacts {
			contactItem := ui.createContactItem(contact)
			ui.contactsList.Add(contactItem)
		}
	}

	ui.contactsList.Refresh()
}

// createContactItem creates a contact item
func (ui *UI) createContactItem(contact *models.Contact) *fyne.Container {
	// Status indicator (green if online)
	statusInd := canvas.NewCircle(color.RGBA{R: 128, G: 128, B: 128, A: 255})

	// Check if contact is connected
	if ui.p2pUI != nil && contact.PeerID != "" {
		connectedPeers := ui.p2pUI.GetConnectedPeers()
		for _, peer := range connectedPeers {
			if peer.PeerID == contact.PeerID {
				statusInd.FillColor = color.RGBA{R: 76, G: 175, B: 80, A: 255}
				break
			}
		}
	}

	// Contact name
	nameLabel := widget.NewLabel(contact.Username)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// PeerID (shortened)
	peerIDText := "no ID"
	if contact.PeerID != "" {
		peerIDText = contact.PeerID
		if len(peerIDText) > 16 {
			peerIDText = peerIDText[:8] + "..." + peerIDText[len(peerIDText)-8:]
		}
	}
	peerIDLabel := widget.NewLabel(peerIDText)
	peerIDLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Chat button
	chatBtn := widget.NewButtonWithIcon("", theme.MailComposeIcon(), func() {
		ui.openChatWithContact(contact)
	})

	// Connect button
	connectBtn := widget.NewButtonWithIcon("", theme.FolderIcon(), func() {
		ui.connectToContactByContact(contact)
	})

	// Delete button
	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		ui.deleteContact(contact)
	})

	content := container.NewBorder(
		nil,
		container.NewHBox(chatBtn, connectBtn, deleteBtn),
		container.NewHBox(statusInd, container.NewVBox(nameLabel, peerIDLabel)),
		nil,
		widget.NewSeparator(),
	)

	return content
}

// openChatWithContact opens a chat with a contact
func (ui *UI) openChatWithContact(contact *models.Contact) {
	// CHECK: if this is a local chat - open via OpenLocalChat()
	if contact.IsLocalChat() {
		log.Printf("[Contact] Local chat detected, opening via OpenLocalChat()")
		ui.contactsUI.OpenLocalChat()
		return
	}

	if contact.PeerID != "" {
		ui.contactsUI.OpenPeerChat(contact.PeerID, contact.Username)
	} else {
		ui.showErrorDialog("Error", "Contact has no PeerID")
	}
}

// connectToContactByContact connects to a contact
func (ui *UI) connectToContactByContact(contact *models.Contact) {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Error", "P2P service not initialized")
		return
	}

	if contact.PeerID == "" {
		ui.showErrorDialog("Error", "Contact has no PeerID")
		return
	}

	multiaddr := contact.Multiaddr
	if multiaddr == "" {
		ui.showErrorDialog("Error", "Contact has no address")
		return
	}

	addrStr := fmt.Sprintf("%s@%s", contact.PeerID, multiaddr)

	err := ui.p2pUI.ConnectToContact(addrStr)
	if err != nil {
		ui.showErrorDialog("Error", fmt.Sprintf("Failed to connect: %v", err))
		return
	}

	ui.showInfoDialog("Connection", "Attempting to connect to contact...")
}

// deleteContact deletes a contact
func (ui *UI) deleteContact(contact *models.Contact) {
	window := ui.contactsUI.GetWindow()
	if window == nil {
		return
	}

	dialog.ShowConfirm(
		"Delete Contact",
		fmt.Sprintf("Are you sure you want to delete the contact \"%s\"?", contact.Username),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			if ui.p2pUI == nil {
				ui.showErrorDialog("Error", "P2P service not initialized")
				return
			}

			err := ui.p2pUI.DeleteContact(contact.ID)
			if err != nil {
				ui.showErrorDialog("Error", fmt.Sprintf("Failed to delete contact: %v", err))
				return
			}

			ui.loadContactsList()
		},
		window,
	)
}

// addContactByAddress adds a contact by address
func (ui *UI) addContactByAddress() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Error", "P2P service not initialized")
		return
	}

	addrStr := ui.addressEntry.Text
	if addrStr == "" {
		ui.showErrorDialog("Error", "Enter contact address")
		return
	}

	username := ui.usernameEntry.Text

	err := ui.p2pUI.AddContactByAddress(addrStr, username)
	if err != nil {
		ui.showErrorDialog("Error", fmt.Sprintf("Failed to add contact: %v", err))
		return
	}

	ui.showInfoDialog("Success", "Contact added")
	ui.addressEntry.SetText("")
	ui.usernameEntry.SetText("")
	ui.loadContactsList()
}

// showErrorDialog shows an error dialog
func (ui *UI) showErrorDialog(title, message string) {
	window := ui.contactsUI.GetWindow()
	if window == nil {
		log.Printf("[%s] %s\n", title, message)
		return
	}
	dialog.ShowError(fmt.Errorf("%s", message), window)
}

// showInfoDialog shows an information dialog
func (ui *UI) showInfoDialog(title, message string) {
	window := ui.contactsUI.GetWindow()
	if window == nil {
		log.Printf("[%s] %s\n", title, message)
		return
	}
	dialog.ShowInformation(title, message, window)
}
