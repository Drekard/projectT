// Package right provides right panel components for chats
package right

import (
	"fmt"
	"log"

	"projectT/internal/services"
	"projectT/internal/storage/database/models"
	"projectT/internal/ui/cards/concrete"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// PreviewPanel represents a panel for viewing elements with 'preview' status
type PreviewPanel struct {
	container *fyne.Container
	window    fyne.Window
	itemsSvc  *services.ItemsService
}

// NewPreviewPanel creates a new preview elements panel
func NewPreviewPanel(window fyne.Window) *PreviewPanel {
	p := &PreviewPanel{
		window:   window,
		itemsSvc: services.NewItemsService(),
	}
	p.container = p.createContainer()
	return p
}

// Container returns the panel container
func (p *PreviewPanel) Container() fyne.CanvasObject {
	return p.container
}

// createContainer creates the panel container
func (p *PreviewPanel) createContainer() *fyne.Container {
	// Title
	title := widget.NewLabel("Element Preview")
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Subtitle
	subtitle := widget.NewLabel("Elements loaded from chats for viewing")
	subtitle.TextStyle = fyne.TextStyle{Italic: true}
	subtitle.Alignment = fyne.TextAlignCenter

	// Container for preview elements
	previewContainer := container.NewVBox()

	// Load preview elements
	previewItems, err := p.itemsSvc.GetPreviewItemsWithoutParentFilter()
	if err != nil {
		log.Printf("Error loading preview elements: %v", err)
		errorLabel := widget.NewLabel("Error loading elements")
		errorLabel.Importance = widget.DangerImportance
		previewContainer.Add(errorLabel)
	} else if len(previewItems) == 0 {
		emptyLabel := widget.NewLabel("No elements to view")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		emptyLabel.Alignment = fyne.TextAlignCenter
		previewContainer.Add(emptyLabel)
	} else {
		// Create cards for each preview element
		for _, item := range previewItems {
			card := p.createPreviewCard(item)
			previewContainer.Add(card)
		}
	}

	// Refresh button
	refreshButton := widget.NewButton("Refresh", func() {
		p.Refresh()
	})

	// Separator
	separator := widget.NewSeparator()

	// Main layout
	content := container.NewVBox(
		title,
		subtitle,
		separator,
		container.NewScroll(previewContainer),
		refreshButton,
	)

	return container.NewPadded(content)
}

// createPreviewCard creates a preview element card with action buttons
func (p *PreviewPanel) createPreviewCard(item *models.Item) fyne.CanvasObject {
	// Element title
	titleLabel := widget.NewLabel(item.Title)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Element description
	descLabel := widget.NewLabel(item.Description)
	descLabel.Wrapping = fyne.TextWrapWord

	// Element type
	typeLabel := widget.NewLabel(fmt.Sprintf("Type: %s", item.Type))
	typeLabel.TextStyle = fyne.TextStyle{Italic: true}

	// "Save" button
	saveButton := widget.NewButton("Save to collection", func() {
		p.saveItem(item)
	})
	saveButton.Importance = widget.HighImportance

	// "Delete" button
	deleteButton := widget.NewButton("Delete", func() {
		p.deleteItem(item)
	})
	deleteButton.Importance = widget.DangerImportance

	// Action buttons
	buttons := container.NewHBox(saveButton, deleteButton)

	// Create element card using concrete
	var cardRenderer fyne.CanvasObject
	switch item.Type {
	case models.ItemTypeFolder:
		cardRenderer = concrete.NewFolderCard(item, true).GetContainer()
	case models.ItemTypeElement:
		cardRenderer = concrete.NewCompositeCard(item, true).GetContainer()
	default:
		cardRenderer = concrete.NewCompositeCard(item, true).GetContainer()
	}

	// Card layout
	card := container.NewVBox(
		cardRenderer,
		titleLabel,
		descLabel,
		typeLabel,
		buttons,
		widget.NewSeparator(),
	)

	return card
}

// saveItem saves element to collection (changes status from 'preview' to 'saved')
func (p *PreviewPanel) saveItem(item *models.Item) {
	log.Printf("[Preview] Saving element: ID=%d, title=%s", item.ID, item.Title)

	err := p.itemsSvc.SavePreviewItem(item.ID)
	if err != nil {
		log.Printf("Error saving element: %v", err)
		dialog.ShowError(fmt.Errorf("error saving element: %w", err), p.window)
		return
	}

	log.Printf("[Preview] ✅ Element saved: ID=%d", item.ID)

	// Show notification
	dialog.ShowInformation("Success", fmt.Sprintf("Element '%s' saved to collection", item.Title), p.window)

	// Refresh panel
	p.Refresh()
}

// deleteItem deletes preview element
func (p *PreviewPanel) deleteItem(item *models.Item) {
	// Delete confirmation
	confirmDialog := dialog.NewConfirm(
		"Delete element",
		fmt.Sprintf("Are you sure you want to delete element '%s'?", item.Title),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			log.Printf("[Preview] Deleting element: ID=%d, title=%s", item.ID, item.Title)

			// TODO: Call element deletion function
			// For now just log
			log.Printf("[Preview] ⚠️ Element deletion not yet implemented")

			// Refresh panel
			p.Refresh()
		},
		p.window,
	)
	confirmDialog.SetConfirmImportance(widget.DangerImportance)
	confirmDialog.Show()
}

// Refresh updates the panel
func (p *PreviewPanel) Refresh() {
	if p.container != nil {
		newContainer := p.createContainer()
		p.container.Objects = newContainer.Objects
		p.container.Refresh()
	}
}
