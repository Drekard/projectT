package center

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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createBubbleForElement создаёт пузырёк для сообщения типа element
func (mb *MessageBubble) createBubbleForElement(message *models.ChatMessage, isOutgoing bool, onRightClick func()) fyne.CanvasObject {
	elementUUID := message.Content
	if elementUUID == "" {
		return mb.createErrorBubble("Неверный формат элемента")
	}

	item, err := queries.GetItemByElementUUID(elementUUID)
	if err != nil {
		return mb.createErrorBubble("Элемент не найден")
	}

	var cardRenderer fyne.CanvasObject
	switch item.Type {
	case "folder":
		cardRenderer = concrete.NewFolderCard(item, true).GetContainer()
	case "element":
		cardRenderer = concrete.NewCompositeCard(item, true).GetContainer()
	default:
		cardRenderer = concrete.NewCompositeCard(item, true).GetContainer()
	}

	content := container.NewVBox(cardRenderer)

	bgColor := color.RGBA{R: 144, G: 55, B: 255, A: 200}
	if !isOutgoing {
		bgColor = color.RGBA{R: 80, G: 80, B: 80, A: 200}
	}

	bg := canvas.NewRectangle(bgColor)
	bg.CornerRadius = 10
	bg.SetMinSize(fyne.NewSize(200, 150))

	messageContainer := container.NewStack(bg, container.NewPadded(content))

	var bubbleContent fyne.CanvasObject
	if isOutgoing {
		bubbleContent = container.NewHBox(layout.NewSpacer(), messageContainer)
	} else {
		bubbleContent = container.NewHBox(messageContainer, layout.NewSpacer())
	}

	clickableBubble := hover_preview.NewClickableCard(bubbleContent, onRightClick)

	return clickableBubble
}

// createBubbleForFolderBatch создаёт пузырёк для сообщения типа folder_batch
func (mb *MessageBubble) createBubbleForFolderBatch(message *models.ChatMessage, isOutgoing bool, onRightClick func(), onOpenFolder func(folderUUID, peerID string)) fyne.CanvasObject {
	folderUUID := message.Content
	if folderUUID == "" {
		return mb.createErrorBubble("Неверный формат папки")
	}

	var folderTitle string
	var itemCount int
	if message.Metadata != "" {
		var meta map[string]interface{}
		if err := json.Unmarshal([]byte(message.Metadata), &meta); err == nil {
			if t, ok := meta["folder_title"].(string); ok {
				folderTitle = t
			}
			if c, ok := meta["item_count"].(float64); ok {
				itemCount = int(c)
			}
		}
	}

	if folderTitle == "" {
		item, err := queries.GetItemByElementUUID(folderUUID)
		if err == nil && item != nil {
			folderTitle = item.Title
		} else {
			folderTitle = "Folder"
		}
	}

	folderIcon := widget.NewIcon(theme.FolderIcon())

	titleLabel := widget.NewLabel(folderTitle)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	countLabel := widget.NewLabel(fmt.Sprintf("%d elements", itemCount))
	countLabel.TextStyle = fyne.TextStyle{Italic: true}

	timeStr := message.SentAt.Format("15:04")
	timeLabel := widget.NewLabel(timeStr)
	timeLabel.TextStyle = fyne.TextStyle{Italic: true}

	cardContent := container.NewVBox(
		container.NewHBox(folderIcon, titleLabel),
		countLabel,
		timeLabel,
	)

	bgColor := color.RGBA{R: 144, G: 55, B: 255, A: 200}
	if !isOutgoing {
		bgColor = color.RGBA{R: 80, G: 80, B: 80, A: 200}
	}

	bg := canvas.NewRectangle(bgColor)
	bg.CornerRadius = 10
	bg.SetMinSize(fyne.NewSize(200, 80))

	messageContainer := container.NewStack(bg, container.NewPadded(cardContent))

	var bubbleContent fyne.CanvasObject
	if isOutgoing {
		bubbleContent = container.NewHBox(layout.NewSpacer(), messageContainer)
	} else {
		bubbleContent = container.NewHBox(messageContainer, layout.NewSpacer())
	}

	if !isOutgoing && onOpenFolder != nil {
		peerID := message.FromPeerID
		clickableBubble := hover_preview.NewClickableCard(bubbleContent, func() {
			onOpenFolder(folderUUID, peerID)
		})
		return clickableBubble
	}

	clickableBubble := hover_preview.NewClickableCard(bubbleContent, onRightClick)
	return clickableBubble
}
