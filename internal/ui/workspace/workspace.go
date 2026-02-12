package workspace

import (
	"fmt"
	"image/color"
	"projectT/internal/services"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/workspace/profile"
	"projectT/internal/ui/workspace/saved"
	"projectT/internal/ui/workspace/saved/sorting"
	"projectT/internal/ui/workspace/tags"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// itemsService - глобальный экземпляр сервиса элементов
var itemsService = services.NewItemsService()

// ContentType определяет тип отображаемого контента
type ContentType string

const (
	ContentTypeSaved   ContentType = "saved"
	ContentTypeProfile ContentType = "profile"
	ContentTypeTags    ContentType = "tags"
)

// NavigationHandler интерфейс для обработки навигации
type NavigationHandler interface {
	NavigateToFolder(folderID int) error
	UpdateContent(contentType string, param ...interface{})
	SearchByTag(tagName string) error
	SetSearchQuery(query string) error
	ApplyFilters(options services.FilterOptions)
}

// Workspace управляет рабочей областью
type Workspace struct {
	container         *fyne.Container
	gridManager       *saved.GridManager
	currentType       ContentType
	contentCache      map[ContentType]fyne.CanvasObject
	navigationManager *NavigationManager // Менеджер навигации
	profileUI         *profile.UI
	tagsUI            *tags.UI
	window            fyne.Window
	// Флаги для отслеживания, были ли UI-компоненты инициализированы
	tagsInitialized bool
	// Фоновое изображение
	background     *ScaledBackground // кастомный фон с масштабированием
	backgroundRect *canvas.Rectangle // прямоугольник фона по умолчанию
	// Режим отображения элементов
	showMode string // "current_folder" или "all_items"
	// Флаг для предотвращения рекурсивного обновления фона
	updatingBackground bool
}

// CreateWorkspace создает и возвращает рабочую область
func CreateWorkspace(window fyne.Window) *Workspace {
	ws := &Workspace{
		container:    container.NewStack(),
		currentType:  ContentTypeSaved,
		contentCache: make(map[ContentType]fyne.CanvasObject),
		window:       window,
	}

	// Инициализируем UI компоненты
	ws.profileUI = profile.New()
	// Не инициализируем tagsUI сразу - ленивая загрузка
	ws.tagsUI = nil

	// Устанавливаем окно для profile UI
	ws.profileUI.SetWindow(window)

	// Инициализируем GridManager для сохраненного контента
	ws.gridManager = saved.NewGridManager()

	// Инициализируем NavigationManager
	ws.navigationManager = NewNavigationManager()

	// Устанавливаем навигацию для GridManager
	ws.gridManager.SetNavigationHandler(ws)

	// Создаем прямоугольник фона по умолчанию
	ws.backgroundRect = canvas.NewRectangle(color.Black)

	// Загружаем фоновое изображение из профиля
	ws.loadBackground()

	// Загружаем начальный контент (сохраненное)
	ws.loadSavedContent()

	// Подписываемся на события изменения фона
	backgroundEventChan := services.GetBackgroundEventManager().Subscribe()
	go func() {
		for eventType := range backgroundEventChan {
			if eventType == "background_changed" || eventType == "background_cleared" {
				fmt.Printf("DEBUG: Workspace - Получено событие фона: %s\n", eventType)
				// Обновляем фон рабочей области
				ws.UpdateBackground()
			}
		}
	}()

	return ws
}

// UpdateContent обновляет содержимое рабочей области
func (ws *Workspace) UpdateContent(contentType string, param ...interface{}) {
	ct := ContentType(contentType)
	ws.currentType = ct

	// Проверяем, есть ли дополнительные параметры для фильтрации
	var extraParam interface{}
	if len(param) > 0 {
		extraParam = param[0]
	}

	// Если тип контента - "tags", то мы должны обновить данные независимо от кэша
	if ct == ContentTypeTags && ws.tagsInitialized {
		// Обновляем данные тегов без проверки кэша
		ws.initializeTagsUI() // Убедимся, что UI инициализирован
		ws.tagsUI.Refresh()
		// Обновляем кэш для этой вкладки
		ws.contentCache[ct] = ws.createTagsContent()
	} else {
		// Проверяем кэш для других типов контента
		if content, exists := ws.contentCache[ct]; exists && extraParam == nil {
			ws.container.Objects = []fyne.CanvasObject{content}
			ws.container.Refresh()
			return
		}
	}

	// Создаем новый контент
	var newContent fyne.CanvasObject
	switch ct {
	case ContentTypeSaved:
		if extraParam != nil {
			// Если передан ID папки, переходим к этой папке
			if folderID, ok := extraParam.(int); ok {
				ws.NavigateToFolder(folderID)
			}
		}
		newContent = ws.createSavedContent()
	case ContentTypeProfile:
		newContent = ws.createProfileContent()
	case ContentTypeTags:
		if extraParam != nil {
			// Если передан ID тега, можно реализовать фильтрацию по тегу
			if tagID, ok := extraParam.(int); ok {
				// TODO: реализовать фильтрацию по тегу
				_ = tagID
			}
		}
		newContent = ws.createTagsContent()
	default:
		newContent = ws.createSavedContent()
	}

	// Сохраняем в кэш
	ws.contentCache[ct] = newContent
	ws.container.Objects = []fyne.CanvasObject{newContent}
	ws.container.Refresh()
}

// loadSavedContent загружает сохраненные элементы
func (ws *Workspace) loadSavedContent() {
	items, err := itemsService.GetItemsByParent(0)
	if err != nil {
		items = []*models.Item{}
	}
	ws.gridManager.LoadItems(items)

	// Устанавливаем корневой элемент как текущий
	ws.gridManager.SetCurrentParentID(0)
}

// NavigateToFolder переходит в указанную папку
func (ws *Workspace) NavigateToFolder(folderID int) error {
	err := ws.navigationManager.GoToFolderInPath(folderID)
	if err != nil {
		return err
	}

	// Загружаем элементы текущей папки с учетом настроек сортировки
	currentParentID := ws.navigationManager.GetCurrentFolderID()
	err = ws.gridManager.LoadItemsByParentWithSort(currentParentID)
	if err != nil {
		return err
	}

	// Обновляем текущий тип контента на "папка"
	ws.currentType = ContentType("folder_" + fmt.Sprintf("%d", currentParentID))
	return nil
}

// RefreshCurrentFolder обновляет содержимое текущей папки
func (ws *Workspace) RefreshCurrentFolder() error {
	currentParentID := ws.navigationManager.GetCurrentFolderID()
	err := ws.gridManager.LoadItemsByParentWithSort(currentParentID)
	if err != nil {
		return err
	}

	// Обновляем текущий тип контента на "папка"
	ws.currentType = ContentType("folder_" + fmt.Sprintf("%d", currentParentID))
	return nil
}

// SearchByTag выполняет поиск элементов по тегу
func (ws *Workspace) SearchByTag(tagName string) error {
	return ws.SearchItems(tagName)
}

// SetSearchQuery устанавливает значение в поисковую строку
func (ws *Workspace) SetSearchQuery(query string) error {
	// В данной реализации Workspace не имеет прямого доступа к поисковому полю
	// Это будет обрабатываться через main_layout, который содержит ссылку на searchEntry
	return nil
}

// SetupNavigation настраивает навигацию
func (ws *Workspace) SetupNavigation(scroll *container.Scroll) {
	// Настройка навигации - в данном случае просто устанавливаем обработчик скролла
	// Используем существующий метод onSizeChanged из GridManager
	scroll.OnScrolled = func(pos fyne.Position) {
		if ws.gridManager != nil {
			ws.gridManager.UpdateLayout()
		}
	}
}

// OnSizeChanged обработчик изменения размера
func (ws *Workspace) OnSizeChanged(pos interface{}) {
	// Обновляем макет сетки при изменении размера
	if ws.gridManager != nil {
		// Вызываем обновление макета
		go ws.gridManager.UpdateLayout()
	}
}

// SearchItems выполняет поиск элементов по запросу
func (ws *Workspace) SearchItems(query string) error {
	if query == "" {
		// Если запрос пустой, возвращаемся к обычному отображению
		currentParentID := ws.navigationManager.GetCurrentFolderID()
		return ws.gridManager.LoadItemsByParentWithSort(currentParentID)
	}

	// Загружаем элементы поисковому запросу с учетом настроек сортировки
	err := ws.gridManager.LoadItemsBySearchWithSort(query)
	if err != nil {
		return err
	}

	// Обновляем текущий тип контента на "поиск"
	ws.currentType = ContentType("search_" + query)
	return nil
}

// ClearSearch очищает результаты поиска и возвращает к нормальному отображению
func (ws *Workspace) ClearSearch() error {
	currentParentID := ws.navigationManager.GetCurrentFolderID()
	err := ws.gridManager.LoadItemsByParentWithSort(currentParentID)
	if err != nil {
		return err
	}

	// Возвращаемся к нормальному типу контента
	ws.currentType = ContentType("folder_" + fmt.Sprintf("%d", currentParentID))
	return nil
}

// GetGridManager возвращает менеджер сетки
func (ws *Workspace) GetGridManager() *saved.GridManager {
	return ws.gridManager
}

// GetNavigationManager возвращает менеджер навигации
func (ws *Workspace) GetNavigationManager() *NavigationManager {
	return ws.navigationManager
}

// createSavedContent создает контент для "Сохраненного"
func (ws *Workspace) createSavedContent() fyne.CanvasObject {
	// Загружаем актуальные данные
	ws.loadSavedContent()
	return ws.gridManager.GetContainer()
}

// createProfileContent создает контент для профиля
func (ws *Workspace) createProfileContent() fyne.CanvasObject {
	return ws.profileUI.CreateView()
}

// createTagsContent создает контент для тегов
func (ws *Workspace) createTagsContent() fyne.CanvasObject {
	ws.initializeTagsUI()
	// Обновляем содержимое при каждом открытии вкладки
	ws.tagsUI.Refresh()
	return ws.tagsUI.GetContent()
}

// initializeTagsUI инициализирует UI тегов при первом обращении
func (ws *Workspace) initializeTagsUI() {
	if !ws.tagsInitialized {
		ws.tagsUI = tags.New()
		ws.tagsInitialized = true
	}
}

// loadBackground загружает фоновое изображение из профиля
func (ws *Workspace) loadBackground() {
	profile, err := queries.GetProfile()
	if err == nil && profile.BackgroundPath != "" {
		// Создаем кастомный фон с масштабированием
		ws.background = NewScaledBackground(profile.BackgroundPath)
	} else {
		// Используем стандартный фон (черный прямоугольник)
		ws.background = nil
	}
}

// GetContainer возвращает контейнер рабочей области с учетом фона
func (ws *Workspace) GetContainer() *fyne.Container {
	// Если есть кастомный фон, используем его
	if ws.background != nil {
		return container.NewStack(ws.background, ws.container)
	}
	// Иначе используем стандартный фон
	return container.NewStack(ws.backgroundRect, ws.container)
}

// UpdateBackground обновляет фон рабочей области
func (ws *Workspace) UpdateBackground() {
	// Предотвращаем рекурсивные вызовы
	if ws.updatingBackground {
		return
	}
	
	ws.updatingBackground = true
	fmt.Println("DEBUG: Workspace - Обновление фона рабочей области")
	
	// Загружаем фоновое изображение из профиля
	profile, err := queries.GetProfile()
	if err == nil && profile.BackgroundPath != "" {
		// Создаем кастомный фон с масштабированием
		ws.background = NewScaledBackground(profile.BackgroundPath)
		fmt.Println("DEBUG: Workspace - Установка нового фона")
	} else {
		// Используем стандартный фон (черный прямоугольник)
		ws.background = nil
		fmt.Println("DEBUG: Workspace - Установка стандартного фона")
	}

	// Обновляем контейнер, чтобы применить изменения фона
	var newContainer *fyne.Container
	if ws.background != nil {
		newContainer = container.NewStack(ws.background, ws.container)
	} else {
		newContainer = container.NewStack(ws.backgroundRect, ws.container)
	}

	// Заменяем объекты в основном контейнере
	ws.container.Objects = newContainer.Objects
	ws.container.Refresh()
	fmt.Println("DEBUG: Workspace - Фон рабочей области обновлен")
	
	// Сбрасываем флаг
	ws.updatingBackground = false
}

// ApplyFilters применяет фильтры и обновляет сетку элементов
func (ws *Workspace) ApplyFilters(options services.FilterOptions) {
	// Обновляем настройки сортировки в GridManager
	ws.gridManager.SetSortOptions(&options)

	// Определяем, какой режим отображения использовать
	if options.TabMode == "all_items" {
		// Режим "Все элементы" - отображаем все элементы без учета ParentID
		ws.showMode = "all_items"
		// Загружаем все элементы из базы данных
		allItems, err := itemsService.GetAllItemsWithoutParentFilter() // используем метод, который возвращает все элементы
		if err != nil {
			// В случае ошибки загружаем пустой список
			allItems = []*models.Item{}
		}
		// Применяем сортировку к полученным элементам
		// Так как GridManager может не иметь метода GetSorter, применяем сортировку напрямую
		sortedItems := ws.sortItems(allItems, &options)
		ws.gridManager.LoadItems(sortedItems)
	} else {
		// Режим "Эта папка" - отображаем элементы текущей папки
		ws.showMode = "current_folder"
		currentParentID := ws.navigationManager.GetCurrentFolderID()
		err := ws.gridManager.LoadItemsByParentWithSort(currentParentID)
		if err != nil {
			// В случае ошибки можно залогировать или обработать по-другому
			// Для простоты просто игнорируем ошибку в этом месте
		}
	}
}

// sortItems сортирует элементы по заданным настройкам
func (ws *Workspace) sortItems(items []*models.Item, options *services.FilterOptions) []*models.Item {
	// Используем сортировщик из пакета sorting
	itemSorter := sorting.NewItemSorter()
	return itemSorter.SortItems(items, options)
}
