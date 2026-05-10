package center

import (
	"image/color"

	"projectT/internal/storage/database/models"
	"projectT/internal/ui/cards/hover_preview"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// MessageBubble пузырёк сообщения
type MessageBubble struct {
	container fyne.CanvasObject
}

// NewMessageBubble создаёт новый пузырёк сообщения
func NewMessageBubble(message *models.ChatMessage, isOutgoing bool, onRightClick func(), onOpenFolder func(folderUUID, peerID string)) *MessageBubble {
	mb := &MessageBubble{}

	switch message.ContentType {
	case "element":
		mb.container = mb.createBubbleForElement(message, isOutgoing, onRightClick)
	case "folder_batch":
		mb.container = mb.createBubbleForFolderBatch(message, isOutgoing, onRightClick, onOpenFolder)
	default:
		mb.container = mb.createBubble(message, isOutgoing, onRightClick)
	}

	return mb
}

// createBubble создаёт пузырёк сообщения
func (mb *MessageBubble) createBubble(message *models.ChatMessage, isOutgoing bool, onRightClick func()) fyne.CanvasObject {
	msgLabel := widget.NewLabel(message.Content)
	msgLabel.Wrapping = fyne.TextWrapBreak

	if isOutgoing {
		msgLabel.Alignment = fyne.TextAlignTrailing
	}

	timeStr := message.SentAt.Format("15:04")
	timeLabel := widget.NewLabel(timeStr)
	timeLabel.TextStyle = fyne.TextStyle{Italic: true}

	if isOutgoing {
		timeLabel.Alignment = fyne.TextAlignTrailing
	}

	content := container.NewVBox(msgLabel, timeLabel)

	bgColor := color.RGBA{R: 144, G: 55, B: 255, A: 200}
	if !isOutgoing {
		bgColor = color.RGBA{R: 80, G: 80, B: 80, A: 200}
	}

	const (
		maxWidth     = 300
		minWidth     = 100
		charsPerUnit = 10
	)
	calculatedWidth := float32(minWidth) + float32(len(message.Content))/charsPerUnit*float32(maxWidth-minWidth)/10
	if calculatedWidth > maxWidth {
		calculatedWidth = maxWidth
	}

	bg := canvas.NewRectangle(bgColor)
	bg.CornerRadius = 10
	bg.SetMinSize(fyne.NewSize(calculatedWidth, 20))

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

// createErrorBubble создаёт пузырёк с сообщением об ошибке
func (mb *MessageBubble) createErrorBubble(errorMsg string) fyne.CanvasObject {
	msgLabel := widget.NewLabel(errorMsg)
	msgLabel.Wrapping = fyne.TextWrapBreak

	timeLabel := widget.NewRichTextFromMarkdown("*ошибка*")
	if len(timeLabel.Segments) > 0 {
		timeLabel.Segments[0].(*widget.TextSegment).Style = widget.RichTextStyleInline
		timeLabel.Segments[0].(*widget.TextSegment).Style.ColorName = theme.ColorNameError
	}

	content := container.NewVBox(msgLabel, timeLabel)

	bgColor := color.RGBA{R: 200, G: 50, B: 50, A: 200}
	bg := canvas.NewRectangle(bgColor)
	bg.CornerRadius = 10
	bg.SetMinSize(fyne.NewSize(200, 50))

	messageContainer := container.NewStack(bg, container.NewPadded(content))
	bubbleContent := container.NewHBox(messageContainer, layout.NewSpacer())

	clickableBubble := hover_preview.NewClickableCard(bubbleContent, func() {})

	return clickableBubble
}

// Container возвращает контейнер пузырька
func (mb *MessageBubble) Container() fyne.CanvasObject {
	return mb.container
}
