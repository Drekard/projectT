package header

import (
	"image/color"
	"projectT/internal/storage/database/models"
	apptheme "projectT/internal/ui/theme"

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
	// Remote mode
	isRemoteMode        bool
	remotePeerID        string
	remotePeerName      string
	onRemoteNavigate    func(string) // Колбэк для навигации по remote папкам (folderUUID)
	onOpenRemoteProfile func(string) // Колбэк для открытия remote профиля (peerID)
}

// CreateBreadcrumbs создает хлебные крошки с текстом текущего раздела
func CreateBreadcrumbs() (*fyne.Container, *BreadcrumbManager) {
	// Цвета
	bgColor := apptheme.GetTheme().BackgroundColor
	// Фон с закруглением и рамкой
	bg := canvas.NewRectangle(bgColor)
	bg.StrokeColor = apptheme.GetTheme().BorderColor
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

	// Подписываемся на смену темы
	apptheme.OnThemeChange(func() {
		bg.FillColor = apptheme.GetTheme().BackgroundColor
		bg.StrokeColor = apptheme.GetTheme().BorderColor
		bg.Refresh()
	})

	// Добавляем иконки перед начальным элементом
	refreshButton := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		// Обработчик перерисовки текущей папки
		if bm.onRefresh != nil {
			bm.onRefresh()
		} else {
			_ = struct{}{} //nolint:staticcheck // Пустой обработчик по умолчанию
		}
	})
	refreshButton.Importance = widget.LowImportance

	folderButton := widget.NewIcon(theme.FolderOpenIcon())

	// Добавляем иконки в контейнер
	buttons.Add(refreshButton)
	buttons.Add(folderButton)

	// Добавляем начальный элемент "Saved"
	bm.AddItem("Saved", 0)

	// Оборачиваем все в Stack: фон + контент с отступами
	// Контент прокручивается горизонтально, если элементов слишком много
	// Border layout позволяет scroll занимать всё оставшееся пространство
	scroll := container.NewHScroll(content)
	padded := container.NewBorder(nil, nil, buttons, nil, container.NewPadded(scroll))
	breadcrumbs := container.NewStack(bg, padded)
	breadcrumbs.Resize(fyne.NewSize(400, 36))

	return breadcrumbs, bm
}

// AddItem добавляет элемент в хлебные крошки
func (bm *BreadcrumbManager) AddItem(title string, folderID int) {
	// Добавляем разделитель, если уже есть элементы
	if len(bm.items) > 0 && bm.container != nil {
		separator := canvas.NewText(" > ", color.RGBA{143, 143, 143, 255})
		separator.TextSize = 14
		bm.container.Add(separator)
	}

	// Создаем элемент для хранения информации
	item := &models.Item{ID: folderID, Title: title}

	// Создаем кнопку для элемента
	button := widget.NewButton(title, func() {
		// Навигация к папке (onNavigate сам переключит вкладку на "Сохраненное")
		if bm.onNavigate != nil {
			bm.onNavigate(folderID)
		}
	})
	button.Importance = widget.LowImportance
	if bm.container != nil {
		button.Resize(fyne.NewSize(80, 24))
	}

	breadcrumbItem := &BreadcrumbItem{
		button: button,
		item:   item,
	}

	bm.items = append(bm.items, breadcrumbItem)
	if bm.container != nil {
		bm.container.Add(button)
	}
}

// UpdateBreadcrumbs обновляет хлебные крошки на основе пути
func (bm *BreadcrumbManager) UpdateBreadcrumbs(path []*models.Item) {
	bm.Clear()

	// Добавляем корневой элемент
	bm.AddItem("Saved", 0)

	// Добавляем остальные элементы пути
	for _, item := range path {
		bm.AddItem(item.Title, item.ID)
	}
}

// Clear очищает хлебные крошки
func (bm *BreadcrumbManager) Clear() {
	if bm.container != nil {
		bm.container.Objects = nil
	}
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

// UpdateRemoteBreadcrumbs обновляет хлебные крошки для удалённого профиля
// peerName — имя удалённого пользователя (первый элемент, кликабельный)
// peerID — ID пира для открытия профиля
// path — путь по папкам
func (bm *BreadcrumbManager) UpdateRemoteBreadcrumbs(peerName string, peerID string, path []*models.Item) {
	bm.isRemoteMode = true
	bm.remotePeerID = peerID
	bm.remotePeerName = peerName
	bm.Clear()

	// Добавляем имя пира как первый элемент (кликабельный → открывает профиль)
	bm.AddRemoteItem(peerName, peerID, "")

	// Добавляем остальные элементы пути (папки)
	for _, item := range path {
		bm.AddRemoteFolderItem(item.Title, item.ElementUUID)
	}

	if bm.container != nil {
		bm.container.Refresh()
	}
}

// AddRemoteItem добавляет элемент remote breadcrumbs (имя пира)
func (bm *BreadcrumbManager) AddRemoteItem(title string, peerID string, folderUUID string) {
	// Добавляем разделитель, если уже есть элементы
	if len(bm.items) > 0 && bm.container != nil {
		separator := canvas.NewText(" > ", color.RGBA{143, 143, 143, 255})
		separator.TextSize = 14
		bm.container.Add(separator)
	}

	// Создаём кнопку для элемента
	button := widget.NewButton(title, func() {
		if folderUUID == "" && bm.onOpenRemoteProfile != nil {
			bm.onOpenRemoteProfile(peerID)
		} else if bm.onRemoteNavigate != nil {
			bm.onRemoteNavigate(folderUUID)
		}
	})
	button.Importance = widget.LowImportance
	if bm.container != nil {
		button.Resize(fyne.NewSize(80, 24))
	}

	// Сохраняем элемент с remote данными
	item := &BreadcrumbItem{
		button: button,
		item:   &models.Item{Title: title},
	}

	bm.items = append(bm.items, item)
	if bm.container != nil {
		bm.container.Add(button)
	}
}

// AddRemoteFolderItem добавляет элемент папки в remote breadcrumbs
func (bm *BreadcrumbManager) AddRemoteFolderItem(title string, folderUUID string) {
	// Добавляем разделитель, если уже есть элементы
	if len(bm.items) > 0 && bm.container != nil {
		separator := canvas.NewText(" > ", color.RGBA{143, 143, 143, 255})
		separator.TextSize = 14
		bm.container.Add(separator)
	}

	// Создаём кнопку для элемента
	button := widget.NewButton(title, func() {
		if bm.onRemoteNavigate != nil {
			bm.onRemoteNavigate(folderUUID)
		}
	})
	button.Importance = widget.LowImportance
	if bm.container != nil {
		button.Resize(fyne.NewSize(80, 24))
	}

	item := &BreadcrumbItem{
		button: button,
		item:   &models.Item{Title: title},
	}

	bm.items = append(bm.items, item)
	if bm.container != nil {
		bm.container.Add(button)
	}
}

// SetRemoteNavigationCallback устанавливает колбэк для remote навигации
func (bm *BreadcrumbManager) SetRemoteNavigationCallback(callback func(string)) {
	bm.onRemoteNavigate = callback
}

// SetOpenRemoteProfileCallback устанавливает колбэк для открытия remote профиля
func (bm *BreadcrumbManager) SetOpenRemoteProfileCallback(callback func(string)) {
	bm.onOpenRemoteProfile = callback
}

// ResetToLocalMode сбрасывает режим на локальный
func (bm *BreadcrumbManager) ResetToLocalMode() {
	bm.isRemoteMode = false
	bm.remotePeerID = ""
	bm.remotePeerName = ""
}
