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
	loadMu            sync.Mutex                                // Мьютекс для защиты от конкурентных загрузок
	loadGeneration    int                                       // Счётчик поколения загрузки для отмены старых
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
		loadGeneration:  0,
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
	availableWidth := scrollSize.Width
	if availableWidth <= 0 {
		availableWidth = gm.sizeManager.GetTotalWidth()
	}
	columnCount := gm.sizeManager.CalculateColumnCount(availableWidth)

	positions := gm.layoutEngine.CalculatePositions(gm.cards, columnCount)

	if len(positions) != len(gm.cards) {
		return // Позиции будут пересчитаны при следующем обновлении
	}

	// Предвыделяем память для объектов контейнера
	gm.container.Objects = make([]fyne.CanvasObject, 0, len(gm.cards))

	// Вычисляем фиксированную ширину один раз
	width := gm.sizeManager.GetFixedWidth()

	// Обновляем позиции и размеры карточек
	// Оптимизация: используем кэш размеров для избежания лишних Resize()
	for i, pos := range positions {
		cardInfo := gm.cards[i]
		cardInfo.Position = pos

		// Используем уже вычисленную ActualHeight
		actualHeight := cardInfo.ActualHeight
		if actualHeight <= 0 {
			actualHeight = utils.DefaultMinHeight
		}

		targetSize := fyne.NewSize(width, actualHeight)

		// Проверяем кэш размеров
		cachedSize, hasCached := gm.widgetSizeCache[cardInfo.Item.ID]

		// Вызываем Resize() только если размера нет в кэше или он отличается
		if !hasCached || cachedSize != targetSize {
			cardInfo.Widget.Resize(targetSize)
			gm.widgetSizeCache[cardInfo.Item.ID] = targetSize // Кэшируем размер
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

	// Если scroll.Size() возвращает 0 (при первой отрисовке), используем дефолтную ширину
	containerWidth := scrollSize.Width
	if containerWidth <= 0 {
		containerWidth = gm.sizeManager.GetTotalWidth()
	}

	// Вычисляем ширину на основе количества колонок
	calculatedWidth := gm.sizeManager.CalculateColumnCount(containerWidth)*int(gm.sizeManager.GetFixedWidth()+gm.sizeManager.GetGapSize()) - int(gm.sizeManager.GetGapSize())

	if containerWidth <= 0 || maxX > containerWidth {
		containerWidth = float32(calculatedWidth)
	}

	containerHeight := maxY + utils.DefaultMinHeight + utils.GapSize

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
	gm.loadMu.Lock()

	// Увеличиваем поколение — предыдущая загрузка (если ещё работает) будет отменена
	gm.loadGeneration++
	currentGeneration := gm.loadGeneration

	gm.clear()

	// Отпускаем мьютекс, чтобы UI не блокировался
	gm.loadMu.Unlock()

	// Запускаем асинхронную загрузку с прогрессивной отрисовкой
	go gm.loadItemsAsync(items, addCreateElement, currentGeneration)
}

// loadItemsAsync загружает элементы асинхронно с прогрессивной отрисовкой
func (gm *GridManager) loadItemsAsync(items []*db_models.Item, addCreateElement bool, generation int) {
	if len(items) == 0 {
		return
	}

	// === ФАЗА 1: Создание карточек параллельно (без UI) ===
	resultChan := make(chan rendering.CardCreationResult, len(items))
	var wg sync.WaitGroup

	wg.Add(len(items))
	for i, item := range items {
		itemIndex := i
		go func(it *db_models.Item, idx int) {
			gm.renderFactory.CreateCardInfoConcurrent(idx, it, gm.navigationHandler, resultChan, &wg)
		}(item, itemIndex)
	}

	// Закрываем канал после завершения всех воркеров
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Собираем ВСЕ результаты в массив по индексам (порядок сохраняется)
	results := make([]*ui_models.CardInfo, len(items))
	receivedCount := 0

	for result := range resultChan {
		if result.Error != nil {
			continue
		}
		if result.CardInfo == nil {
			continue
		}
		results[result.Index] = result.CardInfo
		receivedCount++
	}

	if generation != gm.loadGeneration {
		return
	}

	// === ФАЗА 2: Прогрессивная отрисовка БАТЧАМИ ===
	// Используем ЛОКАЛЬНЫЙ срез для карточек, чтобы избежать гонки с другими поколениями
	localCards := make([]*ui_models.CardInfo, 0, len(items))
	const batchSize = 10

	for batchStart := 0; batchStart < len(results); batchStart += batchSize {
		// Проверяем поколение перед каждым батчем
		if generation != gm.loadGeneration {
			return
		}

		batchEnd := batchStart + batchSize
		if batchEnd > len(results) {
			batchEnd = len(results)
		}

		// Шаг 1: Подготавливаем карточки батча (MinSize, размеры) — БЕЗ UI
		batchCards := make([]*ui_models.CardInfo, 0, batchEnd-batchStart)
		for i := batchStart; i < batchEnd; i++ {
			cardInfo := results[i]
			if cardInfo == nil {
				continue
			}

			// Применяем размеры из кэша
			widthCells, heightCells := gm.getCardSize(cardInfo.Item)
			cardInfo.WidthCells = widthCells
			cardInfo.HeightCells = heightCells

			// Вычисляем фактическую высоту
			if cardInfo.Widget != nil {
				minSize := cardInfo.Widget.MinSize()
				cardInfo.ActualHeight = minSize.Height
				if cardInfo.ActualHeight < utils.DefaultMinHeight {
					cardInfo.ActualHeight = utils.DefaultMinHeight
				}
			}

			// Добавляем в ЛОКАЛЬНЫЙ срез, НЕ в gm.cards
			localCards = append(localCards, cardInfo)
			batchCards = append(batchCards, cardInfo)
		}

		// Шаг 2: Пересчитываем позиции для ЛОКАЛЬНОГО среза
		scrollSize := gm.scroll.Size()
		availableWidth := scrollSize.Width
		if availableWidth <= 0 {
			availableWidth = gm.sizeManager.GetTotalWidth()
		}
		columnCount := gm.sizeManager.CalculateColumnCount(availableWidth)
		positions := gm.layoutEngine.CalculatePositions(localCards, columnCount)

		if len(positions) != len(localCards) {
			continue
		}

		// Шаг 3: Выполняем ВСЕ UI-операции в main goroutine через DoFromGoroutine
		uiDone := make(chan struct{})
		fyne.CurrentApp().Driver().DoFromGoroutine(func() {
			defer close(uiDone)

			// Проверяем поколение ещё раз в main goroutine
			if generation != gm.loadGeneration {
				return
			}

			// ТОЛЬКО здесь записываем в gm.cards — под мьютексом и с проверкой поколения
			gm.loadMu.Lock()
			gm.cards = localCards
			gm.loadMu.Unlock()

			width := gm.sizeManager.GetFixedWidth()

			for i, cardInfo := range batchCards {
				if cardInfo == nil || cardInfo.Widget == nil {
					continue
				}

				pos := positions[batchStart+i]
				cardInfo.Position = pos

				actualHeight := cardInfo.ActualHeight
				if actualHeight <= 0 {
					actualHeight = utils.DefaultMinHeight
				}

				targetSize := fyne.NewSize(width, actualHeight)
				cardInfo.Widget.Resize(targetSize)
				gm.widgetSizeCache[cardInfo.Item.ID] = targetSize

				x, _ := gm.sizeManager.CalculatePixelPosition(pos.X, pos.Y)
				cardInfo.Widget.Move(fyne.NewPos(x, float32(pos.Y)))

				gm.container.Objects = append(gm.container.Objects, cardInfo.Widget)
			}

			gm.updateContainerSize()
			canvas.Refresh(gm.container)
		}, false)

		// Ждём завершения UI-операций
		<-uiDone
	}

	// Проверяем поколение ещё раз перед завершением
	if generation != gm.loadGeneration {
		return
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

// SetColumnCount устанавливает количество колонок в сетке
func (gm *GridManager) SetColumnCount(count int) {
	// Обновляем количество колонок в SizeManager
	// Поскольку SizeManager не имеет публичного метода для установки columnCount,
	// создадим новый SizeManager с нужным количеством колонок
	gm.sizeManager = sizing.NewSizeManagerWithColumnCount(count)

	// Обновляем LayoutEngine с новым количеством колонок
	gm.layoutEngine.UpdateColumnCount(count)
}

// SetSortOptions устанавливает настройки сортировки
func (gm *GridManager) SetSortOptions(options *services.FilterOptions) {
	gm.sortOptions = options
}

// GetSortOptions возвращает текущие настройки сортировки
func (gm *GridManager) GetSortOptions() *services.FilterOptions {
	return gm.sortOptions
}

// SetItemMode устанавливает режим фильтрации элементов (saved/preview)
func (gm *GridManager) SetItemMode(mode string) {
	gm.itemLoader.SetItemMode(mode)
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
}

// getCardSize возвращает размер карточки в ячейках
func (gm *GridManager) getCardSize(item *db_models.Item) (int, int) {
	// Используем кэш для получения размеров
	return gm.cardCache.GetCardSize(item.Type)
}
