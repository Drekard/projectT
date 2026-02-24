package rendering

import (
	"sync"

	db_models "projectT/internal/storage/database/models"
	concrete "projectT/internal/ui/cards/concrete"
	interfaces "projectT/internal/ui/cards/interfaces"
	ui_models "projectT/internal/ui/workspace/saved/models"
	"projectT/internal/ui/workspace/saved/utils"

	"fyne.io/fyne/v2"
)

// FolderCardNavigationHandler интерфейс для обработки навигации по папкам
type FolderCardNavigationHandler interface {
	NavigateToFolder(folderID int) error
}

// CardOption опциональный параметр для создания карточки
type CardOption func(*cardOptions)

type cardOptions struct {
	navigationHandler FolderCardNavigationHandler
	parentID          int
}

// WithNavigation устанавливает обработчик навигации для карточки
func WithNavigation(handler FolderCardNavigationHandler) CardOption {
	return func(opts *cardOptions) {
		opts.navigationHandler = handler
	}
}

// WithParent устанавливает родительский ID для карточки
func WithParent(parentID int) CardOption {
	return func(opts *cardOptions) {
		opts.parentID = parentID
	}
}

// RenderFactory создает карточки для элементов
type RenderFactory struct{}

// NewRenderFactory создает новую фабрику рендеринга
func NewRenderFactory() *RenderFactory {
	return &RenderFactory{}
}

// CreateCard создает карточку для элемента с опциональными параметрами
func (rf *RenderFactory) CreateCard(item *db_models.Item, options ...CardOption) interfaces.CardRenderer {
	opts := &cardOptions{}
	for _, option := range options {
		option(opts)
	}

	switch item.Type {
	case db_models.ItemTypeFolder:
		if opts.navigationHandler != nil {
			return concrete.NewFolderCardWithNavigation(item, opts.navigationHandler)
		}
		return concrete.NewFolderCard(item)
	default:
		return concrete.NewCompositeCard(item)
	}
}

// CreateCardInfo создает информацию о карточке
func (rf *RenderFactory) CreateCardInfo(item *db_models.Item) *ui_models.CardInfo {
	cardRenderer := rf.CreateCard(item)
	widget := cardRenderer.GetWidget()
	widget.Refresh()

	// Здесь должна быть логика получения размеров карточки из кэша или настройки по умолчанию
	widthCells := 1 // Для 3-колоночной системы
	heightCells := 1

	// Вычисляем фактическую высоту карточки
	actualHeight := float32(0)
	if widget != nil {
		// Для всех типов получаем предпочтительный размер виджета
		minSize := widget.MinSize()
		actualHeight = minSize.Height

		// Убедимся, что высота не меньше минимальной
		if actualHeight < utils.DefaultMinHeight {
			actualHeight = utils.DefaultMinHeight
		}
	}

	result := &ui_models.CardInfo{
		Item:         item,
		Widget:       widget,
		Position:     ui_models.CellPosition{X: 0, Y: 0},
		WidthCells:   widthCells,
		HeightCells:  heightCells,
		ActualHeight: actualHeight,
	}

	return result
}

// calculateImageCardHeight вычисляет высоту карточки изображения с учетом пропорций
func (rf *RenderFactory) calculateImageCardHeight(widget fyne.CanvasObject, item *db_models.Item) float32 {
	// Получаем минимальный размер виджета
	minSize := widget.MinSize()

	// Если минимальная высота слишком большая, возможно это связано с аномальным масштабированием
	// Попробуем вычислить разумную высоту для карточки изображения

	// Для карточек с несколькими сегментами, особенно с изображениями размером 300px,
	// может потребоваться дополнительная корректировка высоты
	if minSize.Height > utils.AnomalousHeightThreshold {
		// Если высота аномально большая, возвращаем разумное значение
		return utils.DefaultAnomalousHeight
	}

	return minSize.Height
}

// CreateCardInfoWithNavigation создает информацию о карточке с обработчиком навигации
func (rf *RenderFactory) CreateCardInfoWithNavigation(item *db_models.Item, navigationHandler FolderCardNavigationHandler) *ui_models.CardInfo {
	var cardRenderer interfaces.CardRenderer
	if item.Type == db_models.ItemTypeFolder && navigationHandler != nil {
		cardRenderer = concrete.NewFolderCardWithNavigation(item, navigationHandler)
	} else {
		cardRenderer = rf.CreateCard(item)
	}
	widget := cardRenderer.GetWidget()
	widget.Refresh()

	// Здесь должна быть логика получения размеров карточки из кэша или настройки по умолчанию
	widthCells := 1 // Для 3-колоночной системы
	heightCells := 1

	// Вычисляем фактическую высоту карточки
	actualHeight := float32(0)
	if widget != nil {
		// Для всех типов получаем предпочтительный размер виджета
		minSize := widget.MinSize()
		actualHeight = minSize.Height

		// Убедимся, что высота не меньше минимальной
		if actualHeight < utils.DefaultMinHeight {
			actualHeight = utils.DefaultMinHeight
		}
	}

	result := &ui_models.CardInfo{
		Item:         item,
		Widget:       widget,
		Position:     ui_models.CellPosition{X: 0, Y: 0},
		WidthCells:   widthCells,
		HeightCells:  heightCells,
		ActualHeight: actualHeight,
	}

	return result
}

// CardCreationResult результат асинхронного создания карточки
type CardCreationResult struct {
	Index    int
	CardInfo *ui_models.CardInfo
	Error    error
}

// CreateCardInfoConcurrent создает информацию о карточке и отправляет результат в канал
// Используется для параллельной обработки в worker pool
// ВАЖНО: не вызывает widget.Refresh() и widget.MinSize() - это должно делаться в main goroutine
func (rf *RenderFactory) CreateCardInfoConcurrent(
	index int,
	item *db_models.Item,
	navigationHandler FolderCardNavigationHandler,
	resultChan chan<- CardCreationResult,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	var cardRenderer interfaces.CardRenderer
	if item.Type == db_models.ItemTypeFolder && navigationHandler != nil {
		cardRenderer = concrete.NewFolderCardWithNavigation(item, navigationHandler)
	} else {
		cardRenderer = rf.CreateCard(item)
	}
	widget := cardRenderer.GetWidget()
	// Убрали widget.Refresh() отсюда - будет вызвано в main goroutine

	result := &ui_models.CardInfo{
		Item:        item,
		Widget:      widget,
		Position:    ui_models.CellPosition{X: 0, Y: 0},
		WidthCells:  1,
		HeightCells: 1,
		// ActualHeight будет вычислен в main goroutine
	}

	resultChan <- CardCreationResult{
		Index:    index,
		CardInfo: result,
		Error:    nil,
	}
}
