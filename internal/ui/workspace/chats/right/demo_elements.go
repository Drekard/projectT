// Package right contains right panel components (profile)
package right

import (
	"encoding/json"
	"fmt"
	"image/color"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/cards/concrete"
	"projectT/internal/ui/cards/hover_preview"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// DemoElementItem represents a showcase element element (demo element)
type DemoElementItem struct {
	ID          *int    `json:"id,omitempty"`           // Old format (numeric ID)
	ElementUUID *string `json:"element_uuid,omitempty"` // New format (UUID)
	Title       string  `json:"title"`
	Value       string  `json:"value"`
}

// GetElementUUID returns the ElementUUID of the element
func (dei *DemoElementItem) GetElementUUID() string {
	if dei.ElementUUID != nil && *dei.ElementUUID != "" {
		return *dei.ElementUUID
	}
	// If no UUID, try to find element by old ID
	if dei.ID != nil && *dei.ID > 0 {
		// Load element by ID and get its UUID
		item, err := queries.GetItemByID(*dei.ID)
		if err == nil && item != nil {
			return item.ElementUUID
		}
	}
	return ""
}

// parseDemoElements parses JSON string with pinned UUIDs
// Supports two formats:
// 1. New format (pinned_uuids): ["uuid1", "uuid2", "uuid3"]
// 2. Old format with ID: [{"id": 1, "title": "...", "value": "..."}]
// 3. Very old format: [24, 25, 26] (just ID array)
func parseDemoElements(jsonStr string) ([]DemoElementItem, error) {
	var result []DemoElementItem

	// First try to parse as string array (new pinned_uuids format)
	var uuids []string
	if err := json.Unmarshal([]byte(jsonStr), &uuids); err == nil {
		// This is UUID array
		for _, uuid := range uuids {
			result = append(result, DemoElementItem{
				ElementUUID: &uuid,
				Title:       "Element",
				Value:       "",
			})
		}
		return result, nil
	}

	// If that didn't work, try parsing as object array
	var objects []DemoElementItem
	if err := json.Unmarshal([]byte(jsonStr), &objects); err == nil {
		return objects, nil
	}

	// If that didn't work, try parsing as number array (very old format)
	var ids []int
	if err := json.Unmarshal([]byte(jsonStr), &ids); err == nil {
		// Convert IDs to DemoElementItem
		for _, id := range ids {
			result = append(result, DemoElementItem{
				ID:    &id,
				Title: fmt.Sprintf("Element #%d", id),
				Value: "",
			})
		}
		return result, nil
	}

	// If still no luck, try parsing as single number
	var singleID int
	if err := json.Unmarshal([]byte(jsonStr), &singleID); err == nil {
		result = append(result, DemoElementItem{
			ID:    &singleID,
			Title: fmt.Sprintf("Element #%d", singleID),
			Value: "",
		})
		return result, nil
	}

	// If nothing helped, return error
	return nil, fmt.Errorf("unknown JSON format")
}

// loadDemoElements loads and displays elements from demo_elements
func (p *Panel) loadDemoElements(jsonStr string) {
	if p.demoElementsContainer == nil {
		return
	}

	p.demoElementsContainer.Objects = nil

	// Use new parsing function with support for old formats
	demoElements, err := parseDemoElements(jsonStr)
	if err != nil {
		return
	}

	if len(demoElements) == 0 {
		emptyLabel := widget.NewLabel("No showcase elements")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		p.demoElementsContainer.Add(emptyLabel)
	} else {
		for _, elem := range demoElements {
			elementUUID := elem.GetElementUUID()
			if elementUUID != "" {
				elementCard := p.createDemoElementCard(elementUUID)
				p.demoElementsContainer.Add(elementCard)
			}
		}
	}

	p.demoElementsContainer.Refresh()
}

// createDemoElementCard creates a demo element card (similar to chat_panel.go)
func (p *Panel) createDemoElementCard(elementUUID string) fyne.CanvasObject {
	if elementUUID == "" {
		// If UUID is empty, show error
		return p.createDemoElementError("Invalid element format")
	}

	// Load element from database by element_uuid
	item, err := queries.GetItemByElementUUID(elementUUID)
	if err != nil {
		// If element not found, show error message
		return p.createDemoElementError("Element not found")
	}

	// Create full element card using concrete functionality
	// For profile use mode without buttons
	var cardRenderer fyne.CanvasObject
	switch item.Type {
	case "folder":
		cardRenderer = concrete.NewFolderCard(item, true).GetContainer()
	case "element":
		// For elements use composite card in no-button mode
		cardRenderer = concrete.NewCompositeCard(item, true).GetContainer()
	default:
		// For unknown types use composite card in no-button mode
		cardRenderer = concrete.NewCompositeCard(item, true).GetContainer()
	}

	// Wrap in clickable widget for right-click and preview handling
	clickableCard := hover_preview.NewClickableCard(cardRenderer, func() {
		// Show menu on right-click in no-button mode
		menuManager := hover_preview.NewMenuManager()
		menuManager.ShowSimpleMenu(item, cardRenderer, nil, true)
	})

	// Add element status indicator
	statusIndicator := p.createStatusIndicator(item)

	// Layout: card + status indicator
	cardWithStatus := container.NewVBox(
		clickableCard,
		statusIndicator,
	)

	return cardWithStatus
}

// createStatusIndicator creates an element status indicator
func (p *Panel) createStatusIndicator(item *models.Item) fyne.CanvasObject {
	// Don't show indicator for saved and preview elements
	if item.IsSaved() || item.IsPreview() {
		return container.NewHBox()
	}

	// Show indicator for archived elements
	var statusLabel *widget.Label

	if item.IsArchived() {
		statusLabel = widget.NewLabel("🗄️ Archive")
	} else {
		return container.NewHBox()
	}

	statusLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Indicator layout
	indicator := container.NewHBox(statusLabel)

	return indicator
}

// createDemoElementError creates an error card
func (p *Panel) createDemoElementError(errorMsg string) fyne.CanvasObject {
	msgLabel := widget.NewLabel(errorMsg)
	msgLabel.Wrapping = fyne.TextWrapBreak

	// Create container with background
	bgColor := color.RGBA{R: 200, G: 50, B: 50, A: 200} // Red for errors
	bg := canvas.NewRectangle(bgColor)
	bg.CornerRadius = 10
	bg.SetMinSize(fyne.NewSize(200, 50))

	messageContainer := container.NewStack(bg, container.NewPadded(msgLabel))

	return messageContainer
}
