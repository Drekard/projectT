package profile

import (
	"image/color"
	"projectT/internal/config"
	"projectT/internal/services/pinned"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/workspace/saved"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
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

type LayoutChangeHandler func()

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
	layoutToggleButton       *widget.Button
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
	layoutMode               string // "horizontal" or "vertical"
	onLayoutChange           LayoutChangeHandler
	config                   *config.Config
	onSave                   func()
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

	// Default layout mode (will be overridden by config if available)
	ui.layoutMode = "horizontal"

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

	// Build content based on layout mode
	p.buildContent()
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

	p.layoutToggleButton = widget.NewButtonWithIcon("", theme.MediaReplayIcon(), func() {
		p.toggleLayoutMode()
	})

	// Characteristics container (created once, reused across layout switches)
	p.characteristicsContainer = container.NewVBox()
	p.characteristicsScroll = container.NewScroll(p.characteristicsContainer)
	p.characteristicsScroll.SetMinSize(fyne.NewSize(0, 200))

	p.addCharacteristicButton = widget.NewButton("+ Add characteristic", func() {
		p.AddCharacteristic()
	})
	p.addCharacteristicButton.Importance = widget.LowImportance
}

func (p *UI) buildContent() {
	if p.layoutMode == "horizontal" {
		leftPanel := p.createLeftPanel()
		rightPanel := p.createRightPanel()
		hsplit := container.NewHSplit(leftPanel, rightPanel)
		hsplit.SetOffset(0.35)
		p.content = hsplit
	} else {
		topPanel := p.createTopPanel()
		bottomPanel := p.createRightPanel()
		vsplit := container.NewVSplit(topPanel, bottomPanel)
		vsplit.SetOffset(0.5)
		p.content = vsplit
	}
}

func (p *UI) createAvatarSection() fyne.CanvasObject {
	nameBg := canvas.NewRectangle(color.Transparent)
	nameBg.SetMinSize(fyne.NewSize(250, 40))
	titleBg := canvas.NewRectangle(color.Transparent)
	titleBg.SetMinSize(fyne.NewSize(250, 40))

	return container.NewVBox(
		container.NewCenter(p.avatarContainer),
		container.NewStack(nameBg, p.userNameEntry),
		container.NewStack(titleBg, p.userTitleEntry),
		container.NewCenter(container.NewHBox(p.backgroundButton, p.avatarButton, p.themeButton, p.layoutToggleButton)),
	)
}

func (p *UI) createCharacteristicsSection() fyne.CanvasObject {
	return container.NewVBox(
		p.characteristicsScroll,
		p.addCharacteristicButton,
	)
}

// createLeftPanel creates the left profile panel (avatar, name, title, buttons, characteristics)
func (p *UI) createLeftPanel() fyne.CanvasObject {
	avatarSection := p.createAvatarSection()

	// Horizontal separator
	separator := canvas.NewRectangle(color.Gray{Y: 128})
	separator.SetMinSize(fyne.NewSize(0, 1))

	characteristicsSection := p.createCharacteristicsSection()

	content := container.NewVBox(
		avatarSection,
		separator,
		characteristicsSection,
	)

	return container.NewScroll(content)
}

// createTopPanel creates the top panel for vertical mode (left: avatar/info/buttons, right: characteristics)
// Characteristics are anchored to the bottom edge so the "+ Add characteristic" button is always visible
func (p *UI) createTopPanel() fyne.CanvasObject {
	avatarSection := p.createAvatarSection()

	charScroll := container.NewScroll(p.characteristicsContainer)
	charScroll.SetMinSize(fyne.NewSize(0, 100))

	characteristicsSection := container.NewBorder(
		nil,                       // Top
		p.addCharacteristicButton, // Bottom (anchored to bottom edge)
		nil, nil,                  // Left, Right
		charScroll, // Center
	)

	hsplit := container.NewHSplit(avatarSection, characteristicsSection)
	hsplit.SetOffset(0.4)

	return hsplit
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
				fyne.CurrentApp().Driver().DoFromGoroutine(func() {
					p.updatePinnedItems(pinnedGridManager)
				}, false)
			}
		}
	}()

	// Use Border layout to fill available space without horizontal scrolling
	// Title at top, grid takes all remaining space
	content := container.NewBorder(
		nil,                 // Top
		nil,                 // Bottom
		nil,                 // Left
		nil,                 // Right
		pinnedGridContainer, // Content
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

// toggleLayoutMode switches between horizontal and vertical layout modes
func (p *UI) toggleLayoutMode() {
	if p.layoutMode == "horizontal" {
		p.layoutMode = "vertical"
	} else {
		p.layoutMode = "horizontal"
	}

	// Сохраняем layout mode в конфиг
	if p.config != nil {
		p.config.UISettings.LayoutMode = p.layoutMode
		if p.onSave != nil {
			p.onSave()
		}
	}

	p.buildContent()

	if p.onLayoutChange != nil {
		p.onLayoutChange()
	}
}

// SetLayoutChangeHandler sets the callback for layout mode changes
func (p *UI) SetLayoutChangeHandler(handler LayoutChangeHandler) {
	p.onLayoutChange = handler
}

// SetConfig sets the config for saving UI settings
func (p *UI) SetConfig(cfg *config.Config) {
	p.config = cfg
}

// SetOnSave sets the callback for saving config
func (p *UI) SetOnSave(onSave func()) {
	p.onSave = onSave
}

// RestoreLayoutMode restores layout mode from config
func (p *UI) RestoreLayoutMode() {
	if p.config != nil && p.config.UISettings.LayoutMode != "" {
		p.layoutMode = p.config.UISettings.LayoutMode
		p.buildContent()
	}
}
