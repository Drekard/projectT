package saved

import (
	"image/color"
	"math"
	"sync"
	"time"

	"projectT/internal/services"
	db_models "projectT/internal/storage/database/models"
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
	currentParentID   int                                       // ID текущей папки
	cardSizeCache     map[db_models.ItemType]ui_models.CardSize // Кэш размеров карточек по типу
	widgetSizeCache   map[int]fyne.Size                         // Кэш фактических размеров виджетов по ID элемента
	debouncer         *utils.Debouncer                          // Дебаунсер для обновления макета
	throttler         *utils.SafeThrottler                      // Троттлер для обработки событий
	sortOptions       *services.FilterOptions                   // Настройки сортировки и фильтрации
	lastScrollPos     fyne.Position                             // Последняя позиция скролла для оптимизации
	scrollThreshold   float32                                   // Порог изменения скролла для обновления (в пикселях)
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
		cardSizeCache:   make(map[db_models.ItemType]ui_models.CardSize),
		widgetSizeCache: make(map[int]fyne.Size), // Инициализация кэша размеров виджетов
		debouncer:       utils.NewDebouncer(utils.DebounceDelay * time.Millisecond),
		throttler:       utils.NewSafeThrottler(utils.ThrottleInterval * time.Millisecond),
		sortOptions:     services.GlobalSortSettingsService.GetFilterOptions(), // Используем глобальные настройки сортировки
		lastScrollPos:   fyne.Position{X: 0, Y: 0},                             // Инициализируем начальную позицию
		scrollThreshold: utils.ScrollThreshold,                                 // Порог изменения скролла
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
func (gm *GridManager) AddItem(item *db_models.Item) error {
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
func (gm *GridManager) createCard(item *db_models.Item) *ui_models.CardInfo {
	var cardInfo *ui_models.CardInfo

	// Проверяем, является ли элемент папкой и есть ли обработчик навигации
	if item.Type == db_models.ItemTypeFolder && gm.navigationHandler != nil {
		// Создаем карточку папки с обработчиком навигации
		cardRenderer := gm.renderFactory.CreateCard(item, rendering.WithNavigation(gm.navigationHandler))
		widget := cardRenderer.GetWidget()
		// НЕ вызываем Refresh() - карточка уже инициализирована при создании

		widthCells, heightCells := gm.getCardSize(item)

		// Вычисляем ActualHeight для одной карточки
		actualHeight := widget.MinSize().Height
		if actualHeight < utils.DefaultMinHeight {
			actualHeight = utils.DefaultMinHeight
		}

		cardInfo = &ui_models.CardInfo{
			Item:         item,
			Widget:       widget,
			Position:     ui_models.CellPosition{X: 0, Y: 0},
			WidthCells:   widthCells,
			HeightCells:  heightCells,
			ActualHeight: actualHeight,
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
	// Проверяем, что контейнер инициализирован
	if gm.container == nil {
		return
	}

	// Очищаем контейнер один раз
	gm.container.Objects = gm.container.Objects[:0]

	// Вычисляем количество колонок на основе доступной ширины
	scrollSize := gm.scroll.Size()
	availableWidth := gm.sizeManager.CalculateColumnCount(scrollSize.Width)

	positions := gm.layoutEngine.CalculatePositions(gm.cards, availableWidth)

	if len(positions) != len(gm.cards) {
		return // Позиции будут пересчитаны при следующем обновлении
	}

	// Предвыделяем память для объектов контейнера
	gm.container.Objects = make([]fyne.CanvasObject, 0, len(gm.cards))

	// Вычисляем фиксированную ширину один раз
	width := gm.sizeManager.GetFixedWidth()

	// Оптимизация: откладываем перерисовку до конца всех операций (Batch Refresh)
	// Это предотвращает множественные синхронные перерисовки Fyne

	// Обновляем позиции и размеры карточек
	// Оптимизация: используем кэш размеров для избежания лишних Resize()
	resizeCount := 0
	skipCount := 0

	for i, pos := range positions {
		cardInfo := gm.cards[i]
		cardInfo.Position = pos

		// Используем уже вычисленную ActualHeight из createCardsConcurrently
		actualHeight := cardInfo.ActualHeight
		if actualHeight <= 0 {
			actualHeight = utils.DefaultMinHeight
		}

		targetSize := fyne.NewSize(width, actualHeight)

		// Проверяем кэш размеров
		cachedSize, hasCached := gm.widgetSizeCache[cardInfo.Item.ID]

		// Вызываем Resize() только если размера нет в кэше или он отличается
		// Оптимизация: Lazy Layout — используем Resize() но без немедленной перерисовки
		if !hasCached || cachedSize != targetSize {
			// Используем Resize() для корректной отрисовки
			cardInfo.Widget.Resize(targetSize)
			gm.widgetSizeCache[cardInfo.Item.ID] = targetSize // Кэшируем размер
			resizeCount++
		} else {
			skipCount++
		}

		// Перемещаем виджет на новую позицию
		x, _ := gm.sizeManager.CalculatePixelPosition(pos.X, pos.Y)
		cardInfo.Widget.Move(fyne.NewPos(x, float32(pos.Y)))

		gm.container.Objects = append(gm.container.Objects, cardInfo.Widget)
	}

	gm.updateContainerSize()
}

// Обработчик изменения размера
func (gm *GridManager) onSizeChanged(pos fyne.Position) {
	// Проверяем, изменилась ли позиция скролла достаточно, чтобы обновить макет
	scrollDeltaX := pos.X - gm.lastScrollPos.X
	scrollDeltaY := pos.Y - gm.lastScrollPos.Y
	scrollDistance := float32(math.Sqrt(float64(scrollDeltaX*scrollDeltaX + scrollDeltaY*scrollDeltaY)))

	// Обновляем последнюю позицию скролла
	gm.lastScrollPos = pos

	// Если изменение скролла меньше порога (100 пикселей), пропускаем обновление
	// Это предотвращает цепную реакцию перерисовок при скролле
	if scrollDistance < 100 {
		return
	}

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

	containerHeight := maxY + utils.DefaultMinHeight + utils.GapSize // используем размер ячейки + промежуток

	// Обновляем размеры обоих элементов: контейнера и фона
	gm.container.Resize(fyne.NewSize(containerWidth, containerHeight))
	gm.backgroundRect.SetMinSize(fyne.NewSize(containerWidth, containerHeight))
}

// LoadItems загружает элементы в сетку
func (gm *GridManager) LoadItems(items []*db_models.Item) {
	gm.loadItems(items, true)
}

// LoadItemsWithoutCreateElement загружает элементы в сетку без добавления элемента "Создать элемент"
func (gm *GridManager) LoadItemsWithoutCreateElement(items []*db_models.Item) {
	gm.loadItems(items, false)
}

// loadItems загружает элементы в сетку (внутренний метод)
// if addCreateElement=true, добавляется элемент "Создать элемент"
func (gm *GridManager) loadItems(items []*db_models.Item, addCreateElement bool) {
	gm.clear()

	// НЕ очищаем кэш размеров виджетов — он используется для ускорения повторных загрузок
	// Кэш очищается только при явном вызове Clear() или перезапуске приложения

	// Предвыделяем память для карточек
	capacity := len(items)
	if addCreateElement {
		capacity++
	}
	gm.cards = make([]*ui_models.CardInfo, 0, capacity)

	// Добавляем переданные элементы параллельно с использованием worker pool
	gm.createCardsConcurrently(items)

	// Добавляем элемент "Создать элемент" если требуется (последовательно, т.к. это один элемент)
	if addCreateElement {
		// Здесь можно добавить логику создания элемента "Создать элемент"
		// Например: gm.cards = append(gm.cards, gm.createCreateElementCard())
		_ = struct{}{} //nolint:staticcheck
	}

	// Обновляем макет один раз после добавления всех элементов
	// Расчёт позиций остается последовательным
	gm.updateLayout()

	// Вызываем canvas.Refresh() СИНХРОННО для корректной отрисовки
	// Асинхронный вызов приводил к проблемам с отображением
	canvas.Refresh(gm.container)
}

// createCardsConcurrently создает карточки параллельно с использованием worker pool
func (gm *GridManager) createCardsConcurrently(items []*db_models.Item) {
	if len(items) == 0 {
		return
	}

	// Создаем канал для результатов и WaitGroup
	resultChan := make(chan rendering.CardCreationResult, len(items))
	var wg sync.WaitGroup

	// Предвыделяем результат для сохранения в правильном порядке
	results := make([]*ui_models.CardInfo, len(items))

	// Считаем статистику по типам элементов
	typeStats := make(map[db_models.ItemType]int)
	for _, item := range items {
		typeStats[item.Type]++
	}

	// Запускаем воркеров - создание виджетов происходит параллельно
	wg.Add(len(items))

	// Логирование запуска каждой горутины
	for i, item := range items {
		itemIndex := i // Сохраняем индекс для горутины
		go func(it *db_models.Item, idx int) {
			gm.renderFactory.CreateCardInfoConcurrent(idx, it, gm.navigationHandler, resultChan, &wg)
		}(item, itemIndex)
	}

	// Закрываем канал результатов после завершения всех воркеров
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Собираем результаты (порядок сохраняется по индексу)
	receivedCount := 0
	for result := range resultChan {
		if result.Error != nil {
			continue
		}
		results[result.Index] = result.CardInfo
		receivedCount++
	}

	// Вычисляем размеры и делаем refresh в main goroutine (требуется Fyne)
	// Это делается последовательно, но быстро - только MinSize и Refresh

	// Считаем общую статистику по типам элементов
	processedStats := make(map[db_models.ItemType]int)

	for _, cardInfo := range results {
		if cardInfo != nil {
			itemType := cardInfo.Item.Type

			// Применяем размеры из кэша
			widthCells, heightCells := gm.getCardSize(cardInfo.Item)
			cardInfo.WidthCells = widthCells
			cardInfo.HeightCells = heightCells

			// Вычисляем фактическую высоту в main goroutine
			// НЕ вызываем Refresh() - карточки уже созданы в createCardsConcurrently
			if cardInfo.Widget != nil {
				minSize := cardInfo.Widget.MinSize()

				cardInfo.ActualHeight = minSize.Height
				if cardInfo.ActualHeight < utils.DefaultMinHeight {
					cardInfo.ActualHeight = utils.DefaultMinHeight
				}
			}

			gm.cards = append(gm.cards, cardInfo)
			processedStats[itemType]++
		}
	}
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
	// Очищаем кэш размеров при полной очистке сетки
	gm.widgetSizeCache = make(map[int]fyne.Size)
	// НЕ вызываем container.Refresh() - Fyne сам перерисует после возврата
}

// getCardSize возвращает размер карточки в ячейках
func (gm *GridManager) getCardSize(item *db_models.Item) (int, int) {
	// Используем кэш для получения размеров
	return gm.cardCache.GetCardSize(item.Type)
}

// Вычисление размера для текстовых элементов
func (gm *GridManager) calculateTextSize(item *db_models.Item) (int, int) { //nolint:unused
	// Для 3-колоночной системы все карточки имеют ширину 1 ячейки
	// Высота будет определяться по содержимому
	return 1, 1
}
