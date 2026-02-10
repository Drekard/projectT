package header

import (
	"image/color"

	"projectT/internal/storage/database/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// BreadcrumbItem представляет элемент хлебных крошек
type BreadcrumbItem struct {
	button *widget.Button
	item   *models.Item
}

// BreadcrumbManager управляет хлебными крошками
type BreadcrumbManager struct {
	buttons    *fyne.Container
	container  *fyne.Container
	bg         *canvas.Rectangle
	items      []*BreadcrumbItem // Изменяем на хранение элементов с информацией
	onNavigate func(int)         // Колбэк для навигации по папкам
	onRefresh  func()            // Колбэк для обновления текущей папки
}

// CreateBreadcrumbs создает хлебные крошки с текстом текущего раздела
func CreateBreadcrumbs() (*fyne.Container, *BreadcrumbManager) {
	// Цвета
	bgColor := color.RGBA{44, 44, 44, 255} // Пример цвета фона (#2C2C2C)
	// Фон с закруглением и рамкой
	bg := canvas.NewRectangle(bgColor)
	bg.StrokeColor = color.RGBA{191, 46, 215, 255} // Цвет рамки, чуть светлее фона, по стандарту 80
	bg.StrokeWidth = 1
	bg.CornerRadius = 8
	bg.Resize(fyne.NewSize(400, 36))

	// Контейнер для текстов и кнопок - Horizontal layout
	content := container.NewHBox()
	buttons := container.NewHBox()

	// Создаем менеджер хлебных крошек
	bm := &BreadcrumbManager{
		buttons:   buttons,
		container: content,
		bg:        bg,
		items:     make([]*BreadcrumbItem, 0),
	}

	// Добавляем иконки перед начальным элементом
	refreshButton := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		// Обработчик перерисовки текущей папки
		if bm.onRefresh != nil {
			bm.onRefresh()
		} else {
		}
	})
	refreshButton.Importance = widget.LowImportance

	folderButton := widget.NewIcon(theme.FolderOpenIcon())

	// Добавляем иконки в контейнер
	buttons.Add(refreshButton)
	buttons.Add(folderButton)

	// Добавляем начальный элемент "Сохраненное"
	bm.AddItem("> Сохраненное", 0)

	// Оборачиваем все в Stack: фон + контент с отступами
	padded := container.NewHBox(
		buttons,
		container.NewPadded(content),
	)
	breadcrumbs := container.NewMax(bg, padded)
	breadcrumbs.Resize(fyne.NewSize(400, 36))

	return breadcrumbs, bm
}

// AddItem добавляет элемент в хлебные крошки
func (bm *BreadcrumbManager) AddItem(title string, folderID int) {
	// Добавляем разделитель, если уже есть элементы
	if len(bm.items) > 0 {
		separator := canvas.NewText(" > ", color.RGBA{143, 143, 143, 255})
		separator.TextSize = 14
		bm.container.Add(separator)
	}

	// Создаем элемент для хранения информации
	item := &models.Item{ID: folderID, Title: title}

	// Создаем кнопку для элемента
	button := widget.NewButton(title, func() {
		if bm.onNavigate != nil {
			// Переходим к выбранной папке
			bm.onNavigate(folderID)
		}
	})
	button.Importance = widget.LowImportance
	button.Resize(fyne.NewSize(80, 24))

	breadcrumbItem := &BreadcrumbItem{
		button: button,
		item:   item,
	}

	bm.items = append(bm.items, breadcrumbItem)
	bm.container.Add(button)
}

// clearItemsAfterIndex удаляет все элементы после указанного индекса
func (bm *BreadcrumbManager) clearItemsAfterIndex(index int) {
	if index >= len(bm.items) {
		return
	}

	// Удаляем элементы после указанного индекса
	remainingItems := bm.items[:index+1]

	// Очищаем контейнер
	bm.container.Objects = nil

	// Добавляем оставшиеся элементы обратно
	for i, item := range remainingItems {
		if i > 0 {
			// Добавляем разделитель
			separator := canvas.NewText(" > ", color.RGBA{143, 143, 143, 255})
			separator.TextSize = 14
			bm.container.Add(separator)
		}
		bm.container.Add(item.button)
	}

	// Обновляем список элементов
	bm.items = remainingItems
}

// UpdateBreadcrumbs обновляет хлебные крошки на основе пути
func (bm *BreadcrumbManager) UpdateBreadcrumbs(path []*models.Item) {
	bm.Clear()

	// Добавляем корневой элемент
	bm.AddItem("> Сохраненное", 0)

	// Добавляем остальные элементы пути
	for _, item := range path {
		bm.AddItem(item.Title, item.ID)
	}
}

// Clear очищает хлебные крошки
func (bm *BreadcrumbManager) Clear() {
	bm.container.Objects = nil
	bm.items = make([]*BreadcrumbItem, 0)
}

// SetNavigationCallback устанавливает колбэк для навигации
func (bm *BreadcrumbManager) SetNavigationCallback(callback func(int)) {
	bm.onNavigate = callback
}

// SetRefreshCallback устанавливает колбэк для обновления текущей папки
func (bm *BreadcrumbManager) SetRefreshCallback(callback func()) {
	bm.onRefresh = callback
}

// Refresh вызывает установленный колбэк для обновления текущей папки
func (bm *BreadcrumbManager) Refresh() {
	if bm.onRefresh != nil {
		bm.onRefresh()
	}
}

// GetCurrentFolderID возвращает ID текущей папки (последнего добавленного элемента)
func (bm *BreadcrumbManager) GetCurrentFolderID() *int {
	if len(bm.items) == 0 {
		// Если нет элементов, возвращаем ID корневой папки (0)
		defaultID := 0
		return &defaultID
	}

	lastItem := bm.items[len(bm.items)-1]
	if lastItem.item != nil {
		return &lastItem.item.ID
	}

	return nil
}
