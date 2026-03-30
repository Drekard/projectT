// Package right предоставляет компоненты правой панели чатов
package right

import (
	"fmt"
	"log"

	"projectT/internal/services"
	"projectT/internal/storage/database/models"
	"projectT/internal/ui/cards/concrete"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// PreviewPanel представляет панель для просмотра элементов со статусом 'preview'
type PreviewPanel struct {
	container *fyne.Container
	window    fyne.Window
	itemsSvc  *services.ItemsService
}

// NewPreviewPanel создаёт новую панель preview элементов
func NewPreviewPanel(window fyne.Window) *PreviewPanel {
	p := &PreviewPanel{
		window:   window,
		itemsSvc: services.NewItemsService(),
	}
	p.container = p.createContainer()
	return p
}

// Container возвращает контейнер панели
func (p *PreviewPanel) Container() fyne.CanvasObject {
	return p.container
}

// createContainer создаёт контейнер панели
func (p *PreviewPanel) createContainer() *fyne.Container {
	// Заголовок
	title := widget.NewLabel("Просмотр элементов")
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Подзаголовок
	subtitle := widget.NewLabel("Элементы, загруженные из чатов для просмотра")
	subtitle.TextStyle = fyne.TextStyle{Italic: true}
	subtitle.Alignment = fyne.TextAlignCenter

	// Контейнер для preview элементов
	previewContainer := container.NewVBox()

	// Загружаем preview элементы
	previewItems, err := p.itemsSvc.GetPreviewItemsWithoutParentFilter()
	if err != nil {
		log.Printf("Ошибка загрузки preview элементов: %v", err)
		errorLabel := widget.NewLabel("Ошибка загрузки элементов")
		errorLabel.Importance = widget.DangerImportance
		previewContainer.Add(errorLabel)
	} else if len(previewItems) == 0 {
		emptyLabel := widget.NewLabel("Нет элементов для просмотра")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		emptyLabel.Alignment = fyne.TextAlignCenter
		previewContainer.Add(emptyLabel)
	} else {
		// Создаём карточки для каждого preview элемента
		for _, item := range previewItems {
			card := p.createPreviewCard(item)
			previewContainer.Add(card)
		}
	}

	// Кнопка обновления
	refreshButton := widget.NewButton("Обновить", func() {
		p.Refresh()
	})

	// Разделитель
	separator := widget.NewSeparator()

	// Основная компоновка
	content := container.NewVBox(
		title,
		subtitle,
		separator,
		container.NewScroll(previewContainer),
		refreshButton,
	)

	return container.NewPadded(content)
}

// createPreviewCard создаёт карточку preview элемента с кнопками действий
func (p *PreviewPanel) createPreviewCard(item *models.Item) fyne.CanvasObject {
	// Заголовок элемента
	titleLabel := widget.NewLabel(item.Title)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Описание элемента
	descLabel := widget.NewLabel(item.Description)
	descLabel.Wrapping = fyne.TextWrapWord

	// Тип элемента
	typeLabel := widget.NewLabel(fmt.Sprintf("Тип: %s", item.Type))
	typeLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Кнопка "Сохранить"
	saveButton := widget.NewButton("Сохранить в коллекцию", func() {
		p.saveItem(item)
	})
	saveButton.Importance = widget.HighImportance

	// Кнопка "Удалить"
	deleteButton := widget.NewButton("Удалить", func() {
		p.deleteItem(item)
	})
	deleteButton.Importance = widget.DangerImportance

	// Кнопки действий
	buttons := container.NewHBox(saveButton, deleteButton)

	// Создаём карточку элемента используя concrete
	var cardRenderer fyne.CanvasObject
	switch item.Type {
	case models.ItemTypeFolder:
		cardRenderer = concrete.NewFolderCard(item, true).GetContainer()
	case models.ItemTypeElement:
		cardRenderer = concrete.NewCompositeCard(item, true).GetContainer()
	default:
		cardRenderer = concrete.NewCompositeCard(item, true).GetContainer()
	}

	// Компоновка карточки
	card := container.NewVBox(
		cardRenderer,
		titleLabel,
		descLabel,
		typeLabel,
		buttons,
		widget.NewSeparator(),
	)

	return card
}

// saveItem сохраняет элемент в коллекцию (меняет статус с 'preview' на 'saved')
func (p *PreviewPanel) saveItem(item *models.Item) {
	log.Printf("[Preview] Сохранение элемента: ID=%d, title=%s", item.ID, item.Title)

	err := p.itemsSvc.SavePreviewItem(item.ID)
	if err != nil {
		log.Printf("Ошибка сохранения элемента: %v", err)
		dialog.ShowError(fmt.Errorf("ошибка сохранения элемента: %w", err), p.window)
		return
	}

	log.Printf("[Preview] ✅ Элемент сохранён: ID=%d", item.ID)

	// Показываем уведомление
	dialog.ShowInformation("Успех", fmt.Sprintf("Элемент '%s' сохранён в коллекцию", item.Title), p.window)

	// Обновляем панель
	p.Refresh()
}

// deleteItem удаляет preview элемент
func (p *PreviewPanel) deleteItem(item *models.Item) {
	// Подтверждение удаления
	confirmDialog := dialog.NewConfirm(
		"Удаление элемента",
		fmt.Sprintf("Вы уверены, что хотите удалить элемент '%s'?", item.Title),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			log.Printf("[Preview] Удаление элемента: ID=%d, title=%s", item.ID, item.Title)

			// TODO: Вызвать функцию удаления элемента
			// Пока просто логируем
			log.Printf("[Preview] ⚠️ Удаление элементов ещё не реализовано")

			// Обновляем панель
			p.Refresh()
		},
		p.window,
	)
	confirmDialog.SetConfirmImportance(widget.DangerImportance)
	confirmDialog.Show()
}

// Refresh обновляет панель
func (p *PreviewPanel) Refresh() {
	if p.container != nil {
		newContainer := p.createContainer()
		p.container.Objects = newContainer.Objects
		p.container.Refresh()
	}
}
