package concrete

import (
	"encoding/json"
	"projectT/internal/storage/database/models"
	"projectT/internal/ui/cards"
	"projectT/internal/ui/cards/hover_preview"
	"projectT/internal/ui/cards/interfaces"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// LinkCard карточка для ссылок
type LinkCard struct {
	*cards.BaseCard
	richText             *widget.RichText
	isContentInitialized bool // Флаг: контент уже инициализирован
	noButtonsMode        bool // Режим отображения без кнопок в меню
}

// NewLinkCard создает новую карточку для ссылки
// Опциональный параметр noButtons управляет режимом отображения меню без кнопок
func NewLinkCard(item *models.Item, noButtons ...bool) interfaces.CardRenderer {
	noButtonsMode := len(noButtons) > 0 && noButtons[0]
	return NewLinkCardWithCallback(item, nil, noButtonsMode)
}

// NewLinkCardWithCallback создает новую карточку для ссылки с пользовательским обработчиком клика
// Опциональный параметр noButtons управляет режимом отображения меню без кнопок
func NewLinkCardWithCallback(item *models.Item, clickCallback func(), noButtons ...bool) interfaces.CardRenderer {
	noButtonsMode := len(noButtons) > 0 && noButtons[0]

	linkCard := &LinkCard{
		BaseCard:      cards.NewBaseCard(item),
		noButtonsMode: noButtonsMode,
	}

	// Извлекаем все ссылки
	allLinks := linkCard.extractAllLinks(item.ContentMeta)

	// Создаем сегменты для RichText
	segments := []widget.RichTextSegment{}

	// Добавляем все ссылки как гиперссылки, разделенные разрывами строк
	for i, link := range allLinks {
		text := link
		if len(text) > 40 {
			text = text[:37] + "..."
		}
		linkSegment := &widget.HyperlinkSegment{
			Text: text,
		}
		segments = append(segments, linkSegment)

		linkSegment.OnTapped = func(index int) func() {
			return func() {
				// Открываем URL
				_ = fyne.CurrentApp().OpenURL(hover_preview.ParseURL(allLinks[index]))
			}
		}(i)

		// Добавляем разрыв строки после каждой ссылки, кроме последней
		if i < len(allLinks)-1 {
			segments = append(segments, &widget.TextSegment{
				Text: "\n",
			})
		}
	}
	// Создаем RichText с сегментами
	linkCard.richText = widget.NewRichText(segments...)

	// Контейнер без фона, рамки и отступов, так как будет использоваться внутри другой карточки
	linkCard.Container = linkCard.richText

	// Устанавливаем флаг, что контент инициализирован
	linkCard.isContentInitialized = true

	return linkCard
}

// Методы, необходимые для реализации интерфейса CardRenderer
func (lc *LinkCard) GetContainer() fyne.CanvasObject {
	return lc.Container
}

func (lc *LinkCard) GetWidget() fyne.CanvasObject {
	return lc.Container
}

func (lc *LinkCard) SetContainer(container fyne.CanvasObject) {
	lc.Container = container
}

func (lc *LinkCard) UpdateContent() {
	// Если контент уже инициализирован, просто обновляем контейнер
	// Не пересоздаём карточку заново!
	if lc.isContentInitialized {
		lc.Container.Refresh()
		return
	}

	// Первый вызов - пересоздаем карточку с обновленным элементом
	newCard := NewLinkCardWithCallback(lc.Item, nil, lc.noButtonsMode)

	// Копируем контейнер новой карточки в текущую
	lc.Container = newCard.GetContainer()
}

func (lc *LinkCard) extractAllLinks(contentMeta string) []string {
	var links []string
	if contentMeta == "" {
		return links
	}

	type Block struct {
		Type    string `json:"type"`
		Content string `json:"content,omitempty"`
	}

	var blocks []Block
	if err := json.Unmarshal([]byte(contentMeta), &blocks); err == nil {
		for _, block := range blocks {
			if block.Type == "link" && block.Content != "" {
				links = append(links, block.Content)
			}
		}
	}

	return links
}
