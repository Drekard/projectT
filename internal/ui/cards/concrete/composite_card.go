package concrete

import (
	"projectT/internal/storage/database/models"
	"projectT/internal/ui/cards"
	"projectT/internal/ui/cards/hover_preview"
	"projectT/internal/ui/cards/interfaces"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// CompositeCard карточка для составных элементов
type CompositeCard struct {
	*cards.BaseCard
}

// NewCompositeCard создает новую карточку для составного элемента
func NewCompositeCard(item *models.Item) interfaces.CardRenderer {
	compositeCard := &CompositeCard{
		BaseCard: cards.NewBaseCard(item),
	}

	// Парсим ContentMeta для получения всех блоков
	blocks, err := cards.ParseBlocks(item.ContentMeta)

	if err != nil && item.Description == "" {
		// Если ошибка парсинга, проверяем наличие описания
		// Если нет описания, создаем карточку с сообщением об ошибке
		errorLabel := widget.NewLabel("Ошибка парсинга содержимого")
		errorLabel.Alignment = fyne.TextAlignCenter

		contentContainer := container.NewPadded(container.NewCenter(errorLabel))
		compositeCard.BaseCard.Container = container.NewStack(compositeCard.Background, contentContainer)

		return compositeCard
	}

	// Разделяем блоки по типам
	var textBlocks, imageBlocks, fileBlocks, linkBlocks []cards.Block

	for _, block := range blocks {
		switch block.Type {
		case "text":
			textBlocks = append(textBlocks, block)
		case "image":
			imageBlocks = append(imageBlocks, block)
		case "file":
			fileBlocks = append(fileBlocks, block)
		case "link":
			linkBlocks = append(linkBlocks, block)
		}
	}

	// Создаем секции карточки
	sections := []fyne.CanvasObject{}

	// 1. Секция названия (только если заголовок не пустой)
	if item.Title != "" {
		formattedTitle := cards.GetFormattedTitle(item)
		titleLabel := widget.NewRichTextFromMarkdown(formattedTitle)
		titleLabel.Wrapping = fyne.TextWrapWord
		// Устанавливаем жирное выделение для всего текста
		for _, seg := range titleLabel.Segments {
			if textSeg, ok := seg.(*widget.TextSegment); ok {
				textSeg.Style.TextStyle.Bold = true
			}
		}
		sections = append(sections, titleLabel)
	}

	// 2. Секция текста (если есть текст и отсутствуют другие элементы)
	if item.Description != "" && len(imageBlocks) == 0 && len(fileBlocks) == 0 && len(linkBlocks) == 0 {
		// Создаем временную модель данных для текста
		tempItem := *item
		// Используем Description для текстовой карточки
		tempItem.ContentMeta = ""

		// Создаем карточку текста и получаем ее контейнер
		textCard := NewTextCardWithCallback(&tempItem, nil)
		if textCard != nil && textCard.GetContainer() != nil {
			sections = append(sections, textCard.GetContainer())
		}
	}

	// 3. Секция изображений (если есть)
	if len(imageBlocks) > 0 {
		// Создаем временную модель данных для изображений
		tempItem := *item
		tempItem.ContentMeta = item.ContentMeta // используем оригинальные данные

		// Создаем карточку изображения и получаем ее контейнер
		imageCard := NewImageCardWithCallback(&tempItem, nil)
		if imageCard != nil && imageCard.GetContainer() != nil {
			sections = append(sections, imageCard.GetContainer())
		}
	}

	// 4. Секция файлов (если есть)
	if len(fileBlocks) > 0 {
		// Создаем временную модель данных для файлов
		tempItem := *item
		tempItem.ContentMeta = item.ContentMeta // используем оригинальные данные

		// Создаем карточку файла и получаем ее контейнер
		fileCard := NewFileCardWithCallback(&tempItem, nil)
		if fileCard != nil && fileCard.GetContainer() != nil {
			sections = append(sections, fileCard.GetContainer())
		}
	}

	// 5. Секция ссылок (если есть)
	if len(linkBlocks) > 0 {
		// Создаем временную модель данных для ссылок
		tempItem := *item
		tempItem.ContentMeta = item.ContentMeta // используем оригинальные данные

		// Создаем карточку ссылки и получаем ее контейнер
		linkCard := NewLinkCardWithCallback(&tempItem, nil)
		if linkCard != nil && linkCard.GetContainer() != nil {
			sections = append(sections, linkCard.GetContainer())
		}
	}

	// Собираем все секции в вертикальный контейнер
	mainContent := container.NewVBox(sections...)

	// Оборачиваем в контейнер с фоном
	contentWithBackground := container.NewStack(
		compositeCard.Background,
		container.NewPadded(mainContent),
	)

	// Оборачиваем в кликабельный виджет с поддержкой двойного клика
	compositeCard.Container = hover_preview.NewClickableCardWithDoubleTap(
		contentWithBackground,
		func() {
			// Обычный клик - показываем меню
			menuManager := hover_preview.NewMenuManager()
			menuManager.ShowSimpleMenu(compositeCard.Item, compositeCard.Container, nil)
		},
		func() {
			// Двойной клик - открываем первый файл или изображение
			// Реализация открытия первого элемента будет зависеть от типа карточки
			// Временно оставляем пустой, так как основная логика перенесена в специализированные карточки
		},
	)

	return compositeCard
}

// Методы, необходимые для реализации интерфейса CardRenderer
func (cc *CompositeCard) GetContainer() fyne.CanvasObject {
	return cc.Container
}

func (cc *CompositeCard) GetWidget() fyne.CanvasObject {
	return cc.Container
}

func (cc *CompositeCard) SetContainer(container fyne.CanvasObject) {
	cc.Container = container
}

func (cc *CompositeCard) UpdateContent() {
	// Пересоздаем карточку с обновленным элементом
	newCard := NewCompositeCard(cc.Item)

	// Копируем контейнер новой карточки в текущую
	cc.Container = newCard.GetContainer()
}
