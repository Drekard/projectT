package center

import (
	"encoding/json"
	"image/color"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/cards/concrete"
	"projectT/internal/ui/cards/hover_preview"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
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

	// Загружаем папку из БД
	item, err := queries.GetItemByElementUUID(folderUUID)
	if err != nil || item == nil {
		// Если папки нет в БД — создаём заглушку из метаданных
		var folderTitle string
		if message.Metadata != "" {
			var meta map[string]interface{}
			if err := json.Unmarshal([]byte(message.Metadata), &meta); err == nil {
				if t, ok := meta["folder_title"].(string); ok {
					folderTitle = t
				}
			}
		}
		if folderTitle == "" {
			folderTitle = "Folder"
		}
		item = &models.Item{
			Type:         "folder",
			Title:        folderTitle,
			ElementUUID:  folderUUID,
			OwnerType:    "remote",
			SourcePeerID: &message.FromPeerID,
		}
	}

	// Используем FolderCard из concrete для единообразного отображения
	cardRenderer := concrete.NewFolderCard(item, true)

	bgColor := color.RGBA{R: 144, G: 55, B: 255, A: 200}
	if !isOutgoing {
		bgColor = color.RGBA{R: 80, G: 80, B: 80, A: 200}
	}

	bg := canvas.NewRectangle(bgColor)
	bg.CornerRadius = 10
	bg.SetMinSize(fyne.NewSize(200, 100))

	messageContainer := container.NewStack(bg, container.NewPadded(cardRenderer.GetContainer()))

	var bubbleContent fyne.CanvasObject
	if isOutgoing {
		bubbleContent = container.NewHBox(layout.NewSpacer(), messageContainer)
	} else {
		bubbleContent = container.NewHBox(messageContainer, layout.NewSpacer())
	}

	// Для входящих — клик открывает содержимое папки
	if !isOutgoing && onOpenFolder != nil {
		peerID := message.FromPeerID
		clickableBubble := hover_preview.NewClickableCardWithDoubleTap(bubbleContent, onRightClick, func() {
			onOpenFolder(folderUUID, peerID)
		})
		return clickableBubble
	}

	clickableBubble := hover_preview.NewClickableCard(bubbleContent, onRightClick)
	return clickableBubble
}
