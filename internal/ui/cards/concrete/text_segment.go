package concrete

import (
	"projectT/internal/storage/database/models"
	"projectT/internal/ui/cards"
	"projectT/internal/ui/cards/interfaces"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// TextCard карточка для текстовых элементов
type TextCard struct {
	*cards.BaseCard
	descLabel *widget.RichText
}

// NewTextCard создает новую карточку для текста
func NewTextCard(item *models.Item) interfaces.CardRenderer {
	return NewTextCardWithCallback(item, nil)
}

// NewTextCardWithCallback создает новую карточку для текста с пользовательским обработчиком клика
func NewTextCardWithCallback(item *models.Item, clickCallback func()) interfaces.CardRenderer {
	textCard := &TextCard{
		BaseCard: cards.NewBaseCard(item),
	}

	// Создаем метку для описания (меньший шрифт, с отступами)
	descriptionLabel := widget.NewRichTextFromMarkdown(item.Description)
	descriptionLabel.Wrapping = fyne.TextWrapWord

	// Если длина описания больше 500 символов, оборачиваем в прокручиваемый контейнер
	var descriptionContainer fyne.CanvasObject
	if len(item.Description) > 1000 {
		// Оборачиваем в прокручиваемый контейнер
		scrollContainer := container.NewScroll(descriptionLabel)
		scrollContainer.SetMinSize(fyne.NewSize(200, 400))
		descriptionContainer = scrollContainer
	} else {
		descriptionContainer = descriptionLabel
	}

	// Контейнер без фона, рамки и отступов, так как будет использоваться внутри другой карточки
	textCard.Container = descriptionContainer

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
	// Обновляем содержимое карточки
	// Пересоздаем карточку с обновленным элементом
	newCard := NewTextCardWithCallback(tc.Item, nil)

	// Копируем контейнер новой карточки в текущую
	tc.Container = newCard.GetContainer()
}
