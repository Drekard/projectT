package saved

import (
	"fmt"
	"image/color"
	"time"

	"projectT/internal/services"
	"projectT/internal/storage/database/models"
	ui_models "projectT/internal/ui/workspace/saved/models"
	"projectT/internal/ui/workspace/saved/utils"

	"projectT/internal/ui/workspace/saved/layout"
	"projectT/internal/ui/workspace/saved/loading"
	"projectT/internal/ui/workspace/saved/navigation"
	"projectT/internal/ui/workspace/saved/rendering"
	"projectT/internal/ui/workspace/saved/sizing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// GridManager управляет адаптивной сеткой элементов
type GridManager struct {
	container         *fyne.Container
	backgroundRect    *canvas.Rectangle // Прозрачный прямоугольник для растяжения контейнера
	scroll            *container.Scroll
	cards             []*ui_models.CardInfo
	layoutEngine      *layout.LayoutEngine
	sizeManager       *sizing.SizeManager
	itemLoader        *loading.ItemLoader
	renderFactory     *rendering.RenderFactory
	cardCache         *rendering.CardCache
	navigationHandler navigation.NavigationHandlerInterface
	currentParentID   int                                    // ID текущей папки
	cardSizeCache     map[models.ItemType]ui_models.CardSize // Кэш размеров карточек
	debouncer         *utils.Debouncer                       // Дебаунсер для обновления макета
	throttler         *utils.SafeThrottler                   // Троттлер для обработки событий
	sortOptions       *services.FilterOptions                // Настройки сортировки и фильтрации
}

// NewGridManager создает новый менеджер сетки
func NewGridManager() *GridManager {
	gm := &GridManager{
		cards:           make([]*ui_models.CardInfo, 0, 50), // Предвыделение памяти
		layoutEngine:    layout.NewLayoutEngine(),
		sizeManager:     sizing.NewSizeManager(),
		itemLoader:      loading.NewItemLoader(),
		renderFactory:   rendering.NewRenderFactory(),
		cardCache:       rendering.NewCardCache(),
		currentParentID: 0,
		cardSizeCache:   make(map[models.ItemType]ui_models.CardSize),
		debouncer:       utils.NewDebouncer(100 * time.Millisecond),            // Дебаунсинг 100ms
		throttler:       utils.NewSafeThrottler(16 * time.Millisecond),         // Троттлинг ~60 FPS
		sortOptions:     services.GlobalSortSettingsService.GetFilterOptions(), // Используем глобальные настройки сортировки
	}

	// Инициализация кэша размеров
	gm.initCardSizeCache()

	// Создаем прозрачный прямоугольник для растяжения контейнера
	gm.backgroundRect = canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 0}) // Прозрачный цвет

	// Используем контейнер без layout для ручного позиционирования
	gm.container = container.NewWithoutLayout()

	// Создаем стек из фона и основного контейнера
	stackContainer := container.NewStack(gm.backgroundRect, gm.container)
	gm.scroll = container.NewScroll(stackContainer)

	// Отслеживаем изменения размера
	gm.scroll.OnScrolled = gm.onSizeChanged

	// Выполняем начальное обновление макета для правильного отображения при запуске
	gm.updateLayout()

	return gm
}

// Инициализация кэша размеров карточек
func (gm *GridManager) initCardSizeCache() {
	// Инициализация кэша происходит автоматически при первом обращении через CardCache
}

// SetNavigationHandler устанавливает обработчик навигации
func (gm *GridManager) SetNavigationHandler(handler navigation.NavigationHandlerInterface) {
	gm.navigationHandler = handler
}

// GetContainer возвращает контейнер для встраивания в интерфейс
func (gm *GridManager) GetContainer() *container.Scroll {
	return gm.scroll
}

// AddItem добавляет элемент в сетку
func (gm *GridManager) AddItem(item *models.Item) error {
	// Проверка дубликатов
	if gm.hasItem(item.ID) {
		return nil // Игнорируем дубликаты вместо возврата ошибки
	}

	cardInfo := gm.createCard(item)
	gm.cards = append(gm.cards, cardInfo)

	// Используем дебаунсинг для обновления макета при добавлении элемента
	gm.debouncer.Call(func() {
		gm.throttler.Call(func() {
			gm.updateLayout()
		})
	})
	return nil
}

// Проверка существования элемента
func (gm *GridManager) hasItem(id int) bool {
	for _, card := range gm.cards {
		if card.Item.ID == id {
			return true
		}
	}
	return false
}

// createCard создает карточку для элемента
func (gm *GridManager) createCard(item *models.Item) *ui_models.CardInfo {
	var cardInfo *ui_models.CardInfo

	// Проверяем, является ли элемент папкой и есть ли обработчик навигации
	if item.Type == models.ItemTypeFolder && gm.navigationHandler != nil {
		// Создаем карточку папки с обработчиком навигации
		cardRenderer := gm.renderFactory.CreateCard(item, rendering.WithNavigation(gm.navigationHandler))
		widget := cardRenderer.GetWidget()
		widget.Refresh()

		widthCells, heightCells := gm.getCardSize(item)

		cardInfo = &ui_models.CardInfo{
			Item:        item,
			Widget:      widget,
			Position:    ui_models.CellPosition{X: 0, Y: 0},
			WidthCells:  widthCells,
			HeightCells: heightCells,
		}
	} else {
		// Для остальных элементов используем стандартный метод
		cardInfo = gm.renderFactory.CreateCardInfo(item)

		// Устанавливаем правильные размеры
		widthCells, heightCells := gm.getCardSize(item)
		cardInfo.WidthCells = widthCells
		cardInfo.HeightCells = heightCells
	}

	return cardInfo
}

// UpdateLayout обновляет макет сетки
func (gm *GridManager) UpdateLayout() {
	if gm.container == nil {
		return
	}

	// Используем дебаунсинг и троттлинг для обновления макета
	gm.debouncer.Call(func() {
		gm.throttler.Call(func() {
			gm.updateLayout()
		})
	})
}

// updateLayout обновляет расположение карточек в сетке
func (gm *GridManager) updateLayout() {
	startTime := time.Now()
	fmt.Printf("[%s] Starting layout update for %d cards\n", time.Now().Format("15:04:05.000"), len(gm.cards))
	
	// Проверяем, что контейнер инициализирован
	if gm.container == nil {
		fmt.Printf("[%s] Container is nil, skipping layout update\n", time.Now().Format("15:04:05.000"))
		return
	}

	// Очищаем контейнер один раз
	gm.container.Objects = gm.container.Objects[:0]

	// Вычисляем количество колонок на основе доступной ширины
	scrollSize := gm.scroll.Size()
	availableWidth := gm.sizeManager.CalculateColumnCount(scrollSize.Width)
	
	fmt.Printf("[%s] Calculated available columns: %d\n", time.Now().Format("15:04:05.000"), availableWidth)

	positionsStart := time.Now()
	positions := gm.layoutEngine.CalculatePositions(gm.cards, availableWidth)
	fmt.Printf("[%s] Position calculation took %v\n", time.Now().Format("15:04:05.000"), time.Since(positionsStart))
	
	if len(positions) != len(gm.cards) {
		fmt.Printf("[%s] Position count mismatch: got %d, expected %d\n", time.Now().Format("15:04:05.000"), len(positions), len(gm.cards))
		return // Позиции будут пересчитаны при следующем обновлении
	}

	// Предвыделяем память для объектов контейнера
	gm.container.Objects = make([]fyne.CanvasObject, 0, len(gm.cards))

	cardProcessingStart := time.Now()
	for i, pos := range positions {
		cardInfo := gm.cards[i]
		cardInfo.Position = pos

		// Для новой системы используем фиксированную ширину и переменную высоту
		width := gm.sizeManager.GetFixedWidth()
		// Вычисляем фактическую высоту карточки по содержимому
		if cardInfo.Widget != nil {
			sizeCalcStart := time.Now()
			_, actualHeight := gm.sizeManager.CalculateActualPixelSize(cardInfo.Widget)
			fmt.Printf("[%s] Size calculation for card %d took %v\n", time.Now().Format("15:04:05.000"), i, time.Since(sizeCalcStart))
			
			cardInfo.ActualHeight = actualHeight

			// Обновляем размеры виджета
			cardInfo.Widget.Resize(fyne.NewSize(width, actualHeight))

			x, _ := gm.sizeManager.CalculatePixelPosition(pos.X, pos.Y) // Используем pos.Y напрямую, так как это уже позиция по оси Y
			cardInfo.Widget.Move(fyne.NewPos(x, float32(pos.Y)))
		}

		gm.container.Objects = append(gm.container.Objects, cardInfo.Widget)
	}
	fmt.Printf("[%s] Card processing took %v\n", time.Now().Format("15:04:05.000"), time.Since(cardProcessingStart))

	containerUpdateStart := time.Now()
	gm.updateContainerSize()
	fmt.Printf("[%s] Container size update took %v\n", time.Now().Format("15:04:05.000"), time.Since(containerUpdateStart))
	
	refreshStart := time.Now()
	gm.container.Refresh()
	fmt.Printf("[%s] Container refresh took %v\n", time.Now().Format("15:04:05.000"), time.Since(refreshStart))
	
	fmt.Printf("[%s] Layout update completed in %v\n", time.Now().Format("15:04:05.000"), time.Since(startTime))
}

// Обработчик изменения размера
func (gm *GridManager) onSizeChanged(_ fyne.Position) {
	fmt.Printf("[%s] Scroll event detected, scheduling layout update\n", time.Now().Format("15:04:05.000"))
	
	// Используем дебаунсинг для обновления макета при скролле или изменении размера
	gm.debouncer.Call(func() {
		gm.throttler.Call(func() {
			gm.updateLayout()
		})
	})
}

// updateContainerSize обновляет размер контейнера
func (gm *GridManager) updateContainerSize() {
	maxX, maxY := gm.sizeManager.CalculateMaxDimensions(gm.cards)
	scrollSize := gm.scroll.Size()

	containerWidth := scrollSize.Width

	// Если ширина скролла больше, используем ширину скролла, иначе вычисляем на основе количества колонок
	calculatedWidth := gm.sizeManager.CalculateColumnCount(containerWidth)*int(gm.sizeManager.GetFixedWidth()+gm.sizeManager.GetGapSize()) - int(gm.sizeManager.GetGapSize())

	if containerWidth <= 0 || maxX > containerWidth {
		containerWidth = float32(calculatedWidth)
	}

	containerHeight := maxY + float32(75) // используем размер ячейки + промежуток

	// Обновляем размеры обоих элементов: контейнера и фона
	gm.container.Resize(fyne.NewSize(containerWidth, containerHeight))
	gm.backgroundRect.SetMinSize(fyne.NewSize(containerWidth, containerHeight))
}

// LoadItems загружает элементы в сетку
func (gm *GridManager) LoadItems(items []*models.Item) {
	startTime := time.Now()
	fmt.Printf("[%s] Starting to load %d items\n", time.Now().Format("15:04:05.000"), len(items))
	
	gm.clear()

	// Предвыделяем память для карточек
	gm.cards = make([]*ui_models.CardInfo, 0, len(items)+1)

	// Добавляем переданные элементы
	cardCreationStart := time.Now()
	for i, item := range items {
		cardCreationSingleStart := time.Now()
		cardInfo := gm.createCard(item)
		fmt.Printf("[%s] Card creation for item %d took %v\n", time.Now().Format("15:04:05.000"), i, time.Since(cardCreationSingleStart))
		gm.cards = append(gm.cards, cardInfo)
	}
	fmt.Printf("[%s] Total card creation took %v\n", time.Now().Format("15:04:05.000"), time.Since(cardCreationStart))

	// Обновляем макет один раз после добавления всех элементов
	gm.updateLayout()
	
	fmt.Printf("[%s] Item loading completed in %v\n", time.Now().Format("15:04:05.000"), time.Since(startTime))
}

// LoadItemsWithoutCreateElement загружает элементы в сетку без добавления элемента "Создать элемент"
func (gm *GridManager) LoadItemsWithoutCreateElement(items []*models.Item) {
	startTime := time.Now()
	fmt.Printf("[%s] Starting to load %d items (without create element)\n", time.Now().Format("15:04:05.000"), len(items))
	
	gm.clear()
	gm.cards = make([]*ui_models.CardInfo, 0, len(items))

	cardCreationStart := time.Now()
	for i, item := range items {
		cardCreationSingleStart := time.Now()
		cardInfo := gm.createCard(item)
		fmt.Printf("[%s] Card creation for item %d took %v\n", time.Now().Format("15:04:05.000"), i, time.Since(cardCreationSingleStart))
		gm.cards = append(gm.cards, cardInfo)
	}
	fmt.Printf("[%s] Total card creation took %v\n", time.Now().Format("15:04:05.000"), time.Since(cardCreationStart))

	// Обновляем макет один раз после добавления всех элементов
	gm.updateLayout()
	
	fmt.Printf("[%s] Item loading (without create element) completed in %v\n", time.Now().Format("15:04:05.000"), time.Since(startTime))
}

// LoadItemsByParent загружает элементы по родительскому ID
func (gm *GridManager) LoadItemsByParent(parentID int) error {
	items, err := gm.itemLoader.LoadItemsByParent(parentID)
	if err != nil {
		return err
	}

	gm.currentParentID = parentID
	gm.LoadItems(items)
	return nil
}

// LoadItemsBySearch загружает элементы по поисковому запросу
func (gm *GridManager) LoadItemsBySearch(query string) error {
	items, err := gm.itemLoader.LoadItemsBySearch(query)
	if err != nil {
		return err
	}

	gm.LoadItems(items)
	return nil
}

// ClearSearch очищает результаты поиска
func (gm *GridManager) ClearSearch() error {
	return gm.LoadItemsByParent(gm.currentParentID)
}

// GetCurrentParentID возвращает ID текущей папки
func (gm *GridManager) GetCurrentParentID() int {
	return gm.currentParentID
}

// SetCurrentParentID устанавливает ID текущей папки
func (gm *GridManager) SetCurrentParentID(parentID int) {
	gm.currentParentID = parentID
}

// SetSortOptions устанавливает настройки сортировки
func (gm *GridManager) SetSortOptions(options *services.FilterOptions) {
	gm.sortOptions = options
}

// GetSortOptions возвращает текущие настройки сортировки
func (gm *GridManager) GetSortOptions() *services.FilterOptions {
	return gm.sortOptions
}

// LoadItemsByParentWithSort загружает элементы по родительскому ID с учетом настроек сортировки
func (gm *GridManager) LoadItemsByParentWithSort(parentID int) error {
	items, err := gm.itemLoader.LoadAndSortItemsByParent(parentID, gm.sortOptions)
	if err != nil {
		return err
	}

	gm.currentParentID = parentID
	gm.LoadItems(items)
	return nil
}

// LoadItemsBySearchWithSort загружает элементы по поисковому запросу с учетом настроек сортировки
func (gm *GridManager) LoadItemsBySearchWithSort(query string) error {
	items, err := gm.itemLoader.LoadAndSortItemsBySearch(query, gm.sortOptions)
	if err != nil {
		return err
	}

	gm.LoadItems(items)
	return nil
}

// Clear очищает все элементы
func (gm *GridManager) Clear() {
	gm.clear()
}

// Внутренний метод очистки
func (gm *GridManager) clear() {
	gm.cards = gm.cards[:0]
	gm.container.Objects = gm.container.Objects[:0]
	gm.container.Refresh()
}

// getCardSize возвращает размер карточки в ячейках
func (gm *GridManager) getCardSize(item *models.Item) (int, int) {
	// Используем кэш для получения размеров
	return gm.cardCache.GetCardSize(item.Type)
}

// Вычисление размера для текстовых элементов
func (gm *GridManager) calculateTextSize(item *models.Item) (int, int) {
	// Для 3-колоночной системы все карточки имеют ширину 1 ячейку
	// Высота будет определяться по содержимому
	return 1, 1
}
