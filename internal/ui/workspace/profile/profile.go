package profile

import (
	"image/color"
	"projectT/internal/services/pinned"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/workspace/saved"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ContentCharacteristicItem represents a single characteristic item with title and value
type ContentCharacteristicItem struct {
	ElementUUID string `json:"element_uuid"` // Global element UUID for P2P
	Title       string `json:"title"`
	Value       string `json:"value"`
}

// fieldRow represents a row with a custom field
type fieldRow struct {
	elementUUID  string // ElementUUID of the element (for P2P)
	titleLabel   *widget.Label
	titleEntry   *widget.Entry
	valueEntry   *widget.Entry
	removeButton *widget.Button
	container    *fyne.Container
	timer        *time.Timer
}

type BackgroundUpdater interface {
	SetBackgroundColor(c color.Color)
}

type UI struct {
	content                  fyne.CanvasObject
	userNameEntry            *widget.Entry
	userTitleEntry           *widget.Entry
	avatarImage              *canvas.Image
	avatarContainer          *fyne.Container
	customFields             []*fieldRow
	characteristicsContainer *fyne.Container
	characteristicsScroll    *container.Scroll
	backgroundButton         *widget.Button
	avatarButton             *widget.Button
	themeButton              *widget.Button
	addCharacteristicButton  *widget.Button
	loadCharacteristicsJSON  string
	nextID                   int
	avatarPath               string
	backgroundPath           string
	window                   fyne.Window
	gridManager              *saved.GridManager
	userNameTimer            *time.Timer
	userTitleTimer           *time.Timer
	backgroundUpdater        BackgroundUpdater
}

func New() *UI {
	ui := &UI{}

	// Load profile from database
	profile, err := queries.GetLocalProfile()
	if err == nil {
		// Set paths from database
		ui.avatarPath = profile.AvatarPath
		ui.backgroundPath = profile.BackgroundPath

		// Save characteristics JSON for later loading
		ui.loadCharacteristicsJSON = profile.ContentChar
	} else {
		_ = err //nolint:staticcheck // Ignore profile loading error
	}

	// Initialize gridManager before creating the view
	ui.gridManager = saved.NewGridManager()

	ui.createView()

	// Load characteristics after components are created
	ui.LoadCharacteristicsFromJSON(ui.loadCharacteristicsJSON)

	// nextID is no longer needed since we use ElementUUID instead of ID
	// Kept for backward compatibility
	ui.nextID = 1

	return ui
}

func (p *UI) createView() {
	// Create main components
	p.createComponents()

	// Create left panel (avatar, name, title, buttons, characteristics)
	leftPanel := p.createLeftPanel()

	// Create right panel (pinned items)
	rightPanel := p.createRightPanel()

	// Split into left and right parts using SplitContainer
	split := container.NewHSplit(leftPanel, rightPanel)
	split.SetOffset(0.35) // Left panel takes 35% of width

	p.content = split
}

func (p *UI) createComponents() {
	// Create components for profile

	// Avatar
	var avatarImagePath string
	if p.avatarPath != "" {
		avatarImagePath = p.avatarPath
	} else {
		avatarImagePath = "storage/files/avatars/local/ProjctT_true.png"
	}

	p.avatarImage = canvas.NewImageFromFile(avatarImagePath)
	p.avatarImage.FillMode = canvas.ImageFillContain
	p.avatarImage.SetMinSize(fyne.NewSize(100, 100))

	// Wrap image in clickable widget
	avatarClickable := NewAvatarClickableImage(p.avatarImage, nil)

	// Avatar container
	p.avatarContainer = container.NewCenter(avatarClickable)

	p.userNameEntry = widget.NewEntry()
	p.userNameEntry.SetPlaceHolder("Login")
	p.userNameEntry.OnChanged = func(text string) {
		p.scheduleProfileAutoSave()
	}

	p.userTitleEntry = widget.NewEntry()
	p.userTitleEntry.SetPlaceHolder("Title")
	p.userTitleEntry.OnChanged = func(text string) {
		p.scheduleProfileAutoSave()
	}

	// Load data from profile
	profile, err := queries.GetLocalProfile()
	if err == nil {
		p.userNameEntry.SetText(profile.Username)
		p.userTitleEntry.SetText(profile.Title)
	}

	p.backgroundButton = widget.NewButton("Background", func() {
		p.showBackgroundDialog()
	})

	p.avatarButton = widget.NewButton("Avatar", func() {
		p.showAvatarDialog()
	})

	p.themeButton = widget.NewButton("Theme", func() {
		p.showThemeDialog()
	})
}

// createLeftPanel creates the left profile panel (avatar, name, title, buttons, characteristics)
func (p *UI) createLeftPanel() fyne.CanvasObject {
	// Transparent rectangles for input field backgrounds (400px width)
	nameBg := canvas.NewRectangle(color.Transparent)
	nameBg.SetMinSize(fyne.NewSize(250, 40))
	titleBg := canvas.NewRectangle(color.Transparent)
	titleBg.SetMinSize(fyne.NewSize(250, 40))

	// Avatar, name, title, buttons
	avatarSection := container.NewVBox(
		container.NewCenter(p.avatarContainer),
		container.NewStack(nameBg, p.userNameEntry),
		container.NewStack(titleBg, p.userTitleEntry),
		container.NewCenter(container.NewHBox(p.backgroundButton, p.avatarButton, p.themeButton)),
	)

	// Characteristics
	p.characteristicsContainer = container.NewVBox()
	p.characteristicsScroll = container.NewScroll(p.characteristicsContainer)
	p.characteristicsScroll.SetMinSize(fyne.NewSize(0, 200))

	p.addCharacteristicButton = widget.NewButton("+ Add characteristic", func() {
		p.AddCharacteristic()
	})
	p.addCharacteristicButton.Importance = widget.LowImportance

	characteristicsSection := container.NewVBox(
		widget.NewLabel("Characteristics"),
		p.characteristicsScroll,
		p.addCharacteristicButton,
	)

	// Horizontal separator
	separator := canvas.NewRectangle(color.Gray{Y: 128})
	separator.SetMinSize(fyne.NewSize(0, 1))

	content := container.NewVBox(
		avatarSection,
		separator,
		characteristicsSection,
	)

	return container.NewScroll(content)
}

// createRightPanel creates the right profile panel (pinned items)
func (p *UI) createRightPanel() fyne.CanvasObject {
	// Use GridManager to display pinned items
	pinnedGridManager := saved.NewGridManager()

	// Set 2 columns for profile tab
	pinnedGridManager.SetColumnCount(2)

	// Load pinned items
	p.updatePinnedItems(pinnedGridManager)

	pinnedGridContainer := pinnedGridManager.GetContainer()

	// Subscribe to pinned items change events
	eventChan := pinned.GetEventManager().Subscribe()
	go func() {
		for eventType := range eventChan {
			if eventType == "pinned_items_changed" {
				// Update pinned items
				p.updatePinnedItems(pinnedGridManager)
			}
		}
	}()

	// Use Border layout to fill available space without horizontal scrolling
	// Title at top, grid takes all remaining space
	content := container.NewBorder(
		widget.NewLabel("Showcase"), // Top
		nil,                         // Bottom
		nil,                         // Left
		nil,                         // Right
		pinnedGridContainer,         // Content
	)

	return content
}

// updatePinnedItems updates the display of pinned items
func (p *UI) updatePinnedItems(gridManager *saved.GridManager) {
	// Load pinned items
	pinnedItems, err := queries.GetPinnedItems()
	if err != nil {
		pinnedItems = []*models.Item{} // Initialize with empty list on error
	}

	// Update items in GridManager
	gridManager.LoadItemsWithoutCreateElement(pinnedItems)
}
