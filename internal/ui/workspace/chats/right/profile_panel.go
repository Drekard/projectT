// Package right contains right panel components (profile)
package right

import (
	"image/color"
	"os"

	network "projectT/internal/services/p2p/ui"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Panel represents the right panel with profile
type Panel struct {
	container                *fyne.Container
	profileAvatar            *canvas.Image
	profileName              *widget.Label
	profileStatus            *widget.Label
	characteristicsContainer *fyne.Container
	demoElementsContainer    *fyne.Container
	profileMoreButton        *widget.Button
	chatsUI                  UIProvider
	currentContact           *models.Contact
}

// UIProvider interface for accessing chat UI functions
type UIProvider interface {
	GetP2PService() *network.UIP2P
	OpenRemoteProfile(peerID string)
}

// New creates a new right panel
func New(chatsUI UIProvider) *Panel {
	p := &Panel{
		chatsUI: chatsUI,
	}
	p.container = p.createProfileArea()
	return p
}

// Container returns the panel container
func (p *Panel) Container() *fyne.Container {
	return p.container
}

// Refresh updates the panel
func (p *Panel) Refresh() {
	if p.container != nil {
		p.container.Refresh()
	}
}

// UpdateProfile updates the conversation partner's profile
func (p *Panel) UpdateProfile(contact *models.Contact) {
	p.currentContact = contact

	// Check if this is a local chat
	if contact.IsLocalChat() {
		// For local chat show current user's profile
		if p.profileMoreButton != nil {
			p.profileMoreButton.Hide()
		}
		p.showUserProfile()
		return
	}

	// Show "⋯" button for remote contacts
	if p.profileMoreButton != nil && contact.PeerID != "" {
		p.profileMoreButton.Show()
	}

	// Update name
	if p.profileName != nil {
		p.profileName.SetText(contact.Username)
	}

	// Update status (text, from profile)
	if p.profileStatus != nil {
		p.profileStatus.SetText(contact.Title)
	}

	// Load avatar
	p.loadAvatar(contact.AvatarPath)

	// Load characteristics from peer profile
	if contact.PeerID != "" && p.characteristicsContainer != nil {
		// Load profile from profiles table by PeerID
		profile, err := queries.GetProfileByPeerID(contact.PeerID)
		if err == nil && profile != nil {
			if profile.ContentChar != "" {
				p.loadCharacteristics(profile.ContentChar)
			} else {
				// If no characteristics, clear container
				p.characteristicsContainer.Objects = nil
				p.characteristicsContainer.Refresh()
			}
		}
	}

	// Load demo elements from peer profile
	// NOTE: For remote profiles showcase is loaded separately after element sync
	// via RefreshDemoElementsAfterSync() to avoid race conditions
	if contact.PeerID != "" && p.demoElementsContainer != nil {
		// Clear showcase until sync completes
		p.demoElementsContainer.Objects = nil
		p.demoElementsContainer.Refresh()
	}

	// Update UI
	if p.container != nil {
		p.container.Refresh()
	}
}

// RefreshDemoElementsAfterSync updates the element showcase after sync completion
// This method should be called after successful pinned element loading via ItemSync
func (p *Panel) RefreshDemoElementsAfterSync(peerID string) {
	if p.demoElementsContainer == nil {
		return
	}

	// Load profile from profiles table by PeerID
	profile, err := queries.GetProfileByPeerID(peerID)
	if err != nil {
		return
	}

	if profile == nil {
		return
	}

	if profile.PinnedUUIDs != "" && profile.PinnedUUIDs != "[]" {
		p.loadDemoElements(profile.PinnedUUIDs)
	} else {
		p.demoElementsContainer.Objects = nil
		p.demoElementsContainer.Refresh()
	}

	// Update UI
	if p.container != nil {
		p.container.Refresh()
	}
}

// showUserProfile shows the current user's profile in the right panel
func (p *Panel) showUserProfile() {
	// Load local profile
	localProfile, err := queries.GetLocalProfile()
	if err != nil {
		return
	}

	// Create temporary contact with profile data
	tempContact := &models.Contact{
		Username:   localProfile.Username,
		Title:      localProfile.Title, // Title from profile
		AvatarPath: localProfile.AvatarPath,
		PeerID:     localProfile.PeerID,
	}

	// Update right panel with user profile
	p.UpdateProfile(tempContact)

	// Load characteristics from profile
	if localProfile.ContentChar != "" && p.characteristicsContainer != nil {
		p.loadCharacteristics(localProfile.ContentChar)
	}

	// Load pinned elements from pinned_uuids
	if localProfile.PinnedUUIDs != "" && p.demoElementsContainer != nil {
		p.loadDemoElements(localProfile.PinnedUUIDs)
	}
}

// createProfileArea creates the right panel with conversation partner's profile
func (p *Panel) createProfileArea() *fyne.Container {
	// Avatar - 100x100 image
	p.profileAvatar = canvas.NewImageFromResource(nil)
	p.profileAvatar.FillMode = canvas.ImageFillContain

	// Black background 100x100 for avatar
	avatarBg := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 255})
	avatarBg.SetMinSize(fyne.NewSize(100, 100))

	// Avatar on top of background via Stack
	avatarStack := container.NewStack(avatarBg, p.profileAvatar)

	// Name
	p.profileName = widget.NewLabel("")
	p.profileName.TextStyle = fyne.TextStyle{Bold: true}
	p.profileName.Alignment = fyne.TextAlignCenter

	// "⋯" button for opening full remote profile
	p.profileMoreButton = widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), func() {
		if p.currentContact != nil && p.currentContact.PeerID != "" && !p.currentContact.IsLocalChat() {
			p.chatsUI.OpenRemoteProfile(p.currentContact.PeerID)
		}
	})
	p.profileMoreButton.Importance = widget.LowImportance
	p.profileMoreButton.Hide()

	// User text status (set manually)
	p.profileStatus = widget.NewLabel("")
	p.profileStatus.TextStyle = fyne.TextStyle{Italic: true}
	p.profileStatus.Alignment = fyne.TextAlignCenter

	// Container for avatar and name
	headerContainer := container.NewVBox(
		container.NewCenter(avatarStack),
		container.NewCenter(
			container.NewHBox(
				p.profileName,
				p.profileMoreButton,
			),
		),
		p.profileStatus,
	)

	// Separator
	separator1 := canvas.NewRectangle(color.RGBA{R: 64, G: 64, B: 64, A: 255})
	separator1.SetMinSize(fyne.NewSize(200, 1))

	// Characteristics title
	characteristicsTitle := widget.NewLabel("Characteristics")
	characteristicsTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Container for characteristics
	p.characteristicsContainer = container.NewVBox()

	// Element showcase title
	demoElementsTitle := widget.NewLabel("Element Showcase")
	demoElementsTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Container for demo elements
	p.demoElementsContainer = container.NewVBox()

	// Main info (no internal scrolling)
	infoContainer := container.NewVBox(
		headerContainer,
		separator1,
		container.NewPadded(container.NewVBox(characteristicsTitle, p.characteristicsContainer)),
		separator1,
		container.NewPadded(container.NewVBox(demoElementsTitle, p.demoElementsContainer)),
	)

	// Wrap entire panel in scroll
	scrollContainer := container.NewScroll(infoContainer)

	// Background
	bg := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 255})
	bg.SetMinSize(fyne.NewSize(330, 1))

	p.container = container.NewStack(bg, scrollContainer)

	// Load current user's profile on initialization
	p.showUserProfile()

	return p.container
}

// loadAvatar loads avatar from local storage
func (p *Panel) loadAvatar(avatarPath string) {
	if p.profileAvatar == nil {
		return
	}

	if avatarPath == "" {
		// Empty avatar - hide image
		p.profileAvatar.Resource = nil
		p.profileAvatar.Refresh()
		return
	}

	// Check file existence
	if _, err := os.Stat(avatarPath); os.IsNotExist(err) {
		p.profileAvatar.Resource = nil
		p.profileAvatar.Refresh()
		return
	}

	// Load image
	avatarImg, err := fyne.LoadResourceFromPath(avatarPath)
	if err != nil {
		p.profileAvatar.Resource = nil
		p.profileAvatar.Refresh()
		return
	}

	// Set image
	p.profileAvatar.Resource = avatarImg
	p.profileAvatar.FillMode = canvas.ImageFillContain
	p.profileAvatar.Refresh()
}
