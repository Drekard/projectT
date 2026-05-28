package concrete

import (
	"image/color"
	"projectT/internal/storage/database/models"
	"projectT/internal/ui/cards"
	"projectT/internal/ui/cards/interfaces"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// TextCard карточка для текстовых элементов
type TextCard struct {
	*cards.BaseCard
	isContentInitialized bool // Флаг: контент уже инициализирован
	noButtonsMode        bool // Режим отображения без кнопок в меню
}

// NewTextCard создает новую карточку для текста
// Опциональный параметр noButtons управляет режимом отображения меню без кнопок
func NewTextCard(item *models.Item, noButtons ...bool) interfaces.CardRenderer {
	noButtonsMode := len(noButtons) > 0 && noButtons[0]
	return NewTextCardWithCallback(item, nil, noButtonsMode)
}

// NewTextCardWithCallback создает новую карточку для текста с пользовательским обработчиком клика
// Опциональный параметр noButtons управляет режимом отображения меню без кнопок
func NewTextCardWithCallback(item *models.Item, clickCallback func(), noButtons ...bool) interfaces.CardRenderer {
	noButtonsMode := len(noButtons) > 0 && noButtons[0]

	textCard := &TextCard{
		BaseCard:      cards.NewBaseCard(item),
		noButtonsMode: noButtonsMode,
	}

	// Создаем RichText для поддержки абзацев и переносов
	richText := widget.NewRichTextFromMarkdown(item.Description)
	richText.Wrapping = fyne.TextWrapWord

	// Динамический размер: ~34 символа на строку, ~15px высота строки
	const charsPerLine = 34
	const lineHeight = 15
	const maxVisibleLines = 5

	lineCount := 1
	if len(item.Description) > 0 {
		lines := 0
		currentLineLen := 0
		for _, ch := range item.Description {
			if ch == '\n' {
				lines++
				currentLineLen = 0
			} else {
				currentLineLen++
				if currentLineLen > charsPerLine {
					lines++
					currentLineLen = 0
				}
			}
		}
		if currentLineLen > 0 {
			lines++
		}
		lineCount = lines
	}

	var descriptionContainer fyne.CanvasObject
	if lineCount > maxVisibleLines {
		scrollContainer := container.NewScroll(richText)
		scrollContainer.SetMinSize(fyne.NewSize(200, 300))
		descriptionContainer = scrollContainer
	} else {
		bgContainer := canvas.NewRectangle(color.Transparent)
		bgContainer.SetMinSize(fyne.NewSize(100, float32(maxVisibleLines*lineHeight)))
		st := container.NewStack(bgContainer, richText)
		descriptionContainer = st
	}

	// Контейнер без фона, рамки и отступов, так как будет использоваться внутри другой карточки
	textCard.Container = descriptionContainer

	// Устанавливаем флаг, что контент инициализирован
	textCard.isContentInitialized = true

	return textCard
}

// Методы, необходимые для реализации интерфейса CardRenderer
func (tc *TextCard) GetContainer() fyne.CanvasObject {
	return tc.Container
}

func (tc *TextCard) GetWidget() fyne.CanvasObject {
	return tc.Container
}

func (tc *TextCard) SetContainer(container fyne.CanvasObject) {
	tc.Container = container
}

func (tc *TextCard) UpdateContent() {
	// Если контент уже инициализирован, просто обновляем контейнер
	// Не пересоздаём карточку заново!
	if tc.isContentInitialized {
		tc.Container.Refresh()
		return
	}

	// Первый вызов - пересоздаем карточку с обновленным элементом
	newCard := NewTextCardWithCallback(tc.Item, nil, tc.noButtonsMode)

	// Копируем контейнер новой карточки в текущую
	tc.Container = newCard.GetContainer()
}
