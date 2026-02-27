package cards

import (
	"image/color"

	"projectT/internal/storage/database/models"
	"projectT/internal/ui/cards/hover_preview"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// BaseCard базовая реализация карточки
type BaseCard struct {
	Item       *models.Item
	Container  fyne.CanvasObject
	Background *canvas.Rectangle
	TitleLabel *widget.RichText
	Position   fyne.Position
	Size       fyne.Size
}

// NewBaseCard создает новую базовую карточку
func NewBaseCard(item *models.Item) *BaseCard {
	baseCard := &BaseCard{
		Item:     item,
		Position: fyne.NewPos(0, 0),
		Size:     fyne.NewSize(120, 120),
	}

	// Создаем фон карточки
	baseCard.Background = canvas.NewRectangle(color.RGBA{0, 0, 0, 255})
	baseCard.Background.CornerRadius = 20
	// Устанавливаем серую границу
	baseCard.Background.StrokeColor = color.RGBA{80, 80, 80, 255}
	baseCard.Background.StrokeWidth = 1

	mainContainer := baseCard.Background

	// Оборачиваем в кликабельный виджет
	baseCard.Container = hover_preview.NewClickableCard(mainContainer, func() {
		// Создаем менеджер меню
		menuManager := hover_preview.NewMenuManager()

		// Показываем меню действий
		menuManager.ShowSimpleMenu(baseCard.Item, baseCard.Container, nil)
	})

	return baseCard
}

// GetFormattedTitle возвращает форматированный заголовок с эмодзи, жирным выделением и ограничением длины
func GetFormattedTitle(item *models.Item) string {
	if item.Title == "" {
		return ""
	}

	emoji := GetEmojiForItemType(string(item.Type))
	title := emoji + " " + item.Title

	// Ограничиваем длину заголовка 30 символами, иначе добавляем "..."
	if len(title) > 55 {
		title = title[:52] + "..."
	}

	// Добавляем жирное выделение
	return title
}

// GetTitleWidget возвращает виджет заголовка с эмодзи, жирным выделением и ограничением длины
func (bc *BaseCard) GetTitleWidget() fyne.CanvasObject {
	formattedTitle := GetFormattedTitle(bc.Item)
	if formattedTitle != "" {
		titleLabel := widget.NewRichTextFromMarkdown(formattedTitle)
		titleLabel.Wrapping = fyne.TextWrapWord
		return titleLabel
	} else {
		// Если заголовок пустой, возвращаем пустой контейнер
		return container.NewPadded()
	}
}

// GetWidget возвращает виджет карточки
func (bc *BaseCard) GetWidget() fyne.CanvasObject {
	return bc.Container
}

// GetItem возвращает элемент карточки
func (bc *BaseCard) GetItem() *models.Item {
	return bc.Item
}

// SetPosition устанавливает позицию карточки
func (bc *BaseCard) SetPosition(x, y float32) {
	bc.Position = fyne.NewPos(x, y)
	bc.Container.Move(bc.Position)
}

// SetSize устанавливает размер карточки
func (bc *BaseCard) SetSize(width, height float32) {
	bc.Size = fyne.NewSize(width, height)
	bc.Container.Resize(bc.Size)

	// Обновляем размер фона
	if bc.Background != nil {
		bc.Background.Resize(bc.Size)
	}
}

// GetEmojiForItemType возвращает эмоджи для типа элемента
func GetEmojiForItemType(itemType string) string {
	switch itemType {
	case "text":
		return "📝"
	case "image":
		return "🖼️"
	case "file":
		return "📄"
	case "link":
		return "🔗"
	case "folder":
		return "📁"
	case "audio":
		return "🎵"
	case "video":
		return "🎬"
	default:
		return ""
	}
}
