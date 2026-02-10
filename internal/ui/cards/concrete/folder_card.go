package concrete

import (
	"fmt"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/cards"
	"projectT/internal/ui/cards/hover_preview"
	"projectT/internal/ui/cards/interfaces"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// FolderCard карточка для папок
type FolderCard struct {
	*cards.BaseCard
	titleLabel   *widget.RichText
	countLabel   *widget.Label
	countSegment *widget.TextSegment // Сегмент для счетчика элементов
	richText     *widget.RichText    // RichText для основного содержимого
	lastClick    time.Time           // Для обработки двойного клика
}

// FolderCardNavigationHandler интерфейс для обработки навигации по папкам
type FolderCardNavigationHandler interface {
	NavigateToFolder(folderID int) error
}

// GetItemCount возвращает количество элементов в папке
func GetItemCount(folderID int) (int, error) {
	items, err := queries.GetItemsByParent(folderID)
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

// NewFolderCard создает новую карточку для папки
func NewFolderCard(item *models.Item) interfaces.CardRenderer {
	return NewFolderCardWithNavigation(item, nil)
}

// NewFolderCardWithNavigation создает новую карточку для папки с обработчиком навигации
func NewFolderCardWithNavigation(item *models.Item, navigationHandler FolderCardNavigationHandler) interfaces.CardRenderer {
	folderCard := &FolderCard{
		BaseCard: cards.NewBaseCard(item),
	}

	// Создаем сегменты для RichText
	segments := []widget.RichTextSegment{}

	// Добавляем заголовок (с эмодзи и жирным стилем)
	formattedTitle := cards.GetFormattedTitle(item)
	titleSegment := &widget.TextSegment{
		Text: formattedTitle,
		Style: widget.RichTextStyle{
			TextStyle: fyne.TextStyle{Bold: true},
		},
	}
	segments = append(segments, titleSegment)

	// Добавляем перенос строки
	segments = append(segments, &widget.SeparatorSegment{})

	// Счетчик элементов - сначала устанавливаем "Загрузка..." пока получаем данные
	countSegment := &widget.TextSegment{
		Text: "Загрузка...",
		Style: widget.RichTextStyle{
			Inline: true,
		},
	}
	folderCard.countSegment = countSegment
	segments = append(segments, countSegment)

	// Создаем RichText с сегментами
	folderCard.richText = widget.NewRichText(segments...)
	folderCard.richText.Wrapping = fyne.TextWrapWord

	// Создаем вертикальный контейнер для содержимого
	content := container.NewVBox(folderCard.richText)

	// Оборачиваем в кликабельный виджет с поддержкой двойного клика
	clickableCard := hover_preview.NewClickableCardWithDoubleTap(content, func() {
		// Создаем менеджер меню
		menuManager := hover_preview.NewMenuManager()

		// Показываем меню действий
		menuManager.ShowSimpleMenu(folderCard.Item, folderCard.Container, nil)
	}, func() {
		// Обработчик двойного клика - переход в папку
		if navigationHandler != nil {
			err := navigationHandler.NavigateToFolder(folderCard.Item.ID)
			if err != nil {
				// Обработка ошибки перехода в папку
			}
		} else {
			// Обработка случая, когда обработчик навигации не установлен
		}
	})

	// Обновляем контейнер карточки
	folderCard.BaseCard.Container = container.NewStack(folderCard.Background, clickableCard)

	// Асинхронно обновляем счетчик элементов
	go func() {
		count, err := GetItemCount(item.ID)
		if err != nil {
			// Обновляем сегмент с ошибкой
			folderCard.countSegment.Text = "Ошибка загрузки"
			folderCard.richText.Refresh()
		} else {
			elementText := "элементов"
			if count == 1 {
				elementText = "элемент"
			} else if count > 1 && count < 5 {
				elementText = "элемента"
			}
			// Обновляем сегмент с количеством
			folderCard.countSegment.Text = fmt.Sprintf("%d %s", count, elementText)
			folderCard.richText.Refresh()
		}
	}()

	return folderCard
}

// Методы, необходимые для реализации интерфейса CardRenderer
func (fc *FolderCard) GetContainer() fyne.CanvasObject {
	return fc.Container
}

func (fc *FolderCard) GetWidget() fyne.CanvasObject {
	return fc.Container
}

func (fc *FolderCard) SetContainer(container fyne.CanvasObject) {
	fc.Container = container
}

func (fc *FolderCard) UpdateContent() {
	// Обновляем содержимое карточки
	// Пересоздаем карточку с обновленным элементом
	newCard := NewFolderCardWithNavigation(fc.Item, nil)

	// Копируем контейнер новой карточки в текущую
	fc.Container = newCard.GetContainer()
}
