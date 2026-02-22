package sizing

import (
	"fmt"
	"time"

	"projectT/internal/ui/workspace/saved/models"
	"projectT/internal/ui/workspace/saved/utils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// SizeManager управляет размерами карточек
type SizeManager struct {
	fixedWidth         float32 // Фиксированная ширина карточки
	minHeight          float32 // Минимальная высота карточки
	gapSize            float32 // Размер промежутка между карточками
	defaultColumnCount int     // Количество колонок по умолчанию (3 колонки)
	totalWidth         float32 // Общая ширина, занимаемая карточками и промежутками
}

// NewSizeManager создает новый менеджер размеров
func NewSizeManager() *SizeManager {
	fixedWidth := utils.FixedCardWidth
	minHeight := utils.DefaultMinHeight
	gapSize := utils.GapSize
	columnCount := utils.DefaultColumnCount

	// Общая ширина = ширина 3 карточек + 2 промежутка между ними
	totalWidth := fixedWidth*float32(columnCount) + gapSize*float32(columnCount-1)

	return &SizeManager{
		fixedWidth:         fixedWidth,
		minHeight:          minHeight,
		gapSize:            gapSize,
		defaultColumnCount: columnCount,
		totalWidth:         totalWidth,
	}
}

// CalculatePixelSize вычисляет размер в пикселях
func (sm *SizeManager) CalculatePixelSize(widthCells, heightCells int) (float32, float32) {
	startTime := time.Now()
	fmt.Printf("[%s] Calculating pixel size for %d x %d cells\n", time.Now().Format("15:04:05.000"), widthCells, heightCells)
	
	// Для новой системы мы используем фиксированную ширину и переменную высоту
	width := sm.fixedWidth
	height := float32(heightCells)*sm.minHeight + float32(heightCells-1)*sm.gapSize
	
	fmt.Printf("[%s] Pixel size calculation completed in %v\n", time.Now().Format("15:04:05.000"), time.Since(startTime))
	return width, height
}

// CalculatePixelPosition вычисляет позицию в пикселях
func (sm *SizeManager) CalculatePixelPosition(x, y int) (float32, float32) {
	// x теперь будет номером колонки (0, 1 или 2)
	colX := float32(x) * (sm.fixedWidth + sm.gapSize)
	return colX, float32(y)
}

// GetAvailableWidth возвращает доступную ширину для размещения карточек
func (sm *SizeManager) GetAvailableWidth(scroll *container.Scroll) float32 {
	scrollSize := scroll.Size()
	if scrollSize.Width <= 0 {
		return sm.totalWidth // Возвращаем ширину для 3 колонок
	}

	// Проверяем, помещаются ли 3 колонки в доступное пространство
	if scrollSize.Width >= sm.totalWidth {
		return sm.totalWidth
	}

	// Если нет, то используем доступное пространство
	return scrollSize.Width
}

// CalculateMaxDimensions вычисляет максимальные размеры
func (sm *SizeManager) CalculateMaxDimensions(cards []*models.CardInfo) (float32, float32) {
	startTime := time.Now()
	fmt.Printf("[%s] Calculating max dimensions for %d cards\n", time.Now().Format("15:04:05.000"), len(cards))
	
	var maxX, maxY float32

	for i, card := range cards {
		posCalcStart := time.Now()
		x, y := sm.CalculatePixelPosition(card.Position.X, card.Position.Y)
		fmt.Printf("[%s] Position calculation for card %d took %v\n", time.Now().Format("15:04:05.000"), i, time.Since(posCalcStart))

		// Для новой системы ширина фиксирована, высота берется из самой карточки
		width := sm.fixedWidth
		height := card.ActualHeight

		right := x + width
		bottom := y + height

		if right > maxX {
			maxX = right
		}
		if bottom > maxY {
			maxY = bottom
		}
	}

	fmt.Printf("[%s] Max dimensions calculation completed in %v\n", time.Now().Format("15:04:05.000"), time.Since(startTime))
	return maxX, maxY
}

// GetFixedWidth возвращает фиксированную ширину карточки
func (sm *SizeManager) GetFixedWidth() float32 {
	return sm.fixedWidth
}

// GetMinHeight возвращает минимальную высоту карточки
func (sm *SizeManager) GetMinHeight() float32 {
	return sm.minHeight
}

// GetGapSize возвращает размер промежутка
func (sm *SizeManager) GetGapSize() float32 {
	return sm.gapSize
}

// GetColumnCount возвращает количество колонок
func (sm *SizeManager) GetColumnCount() int {
	return sm.defaultColumnCount
}

// GetTotalWidth возвращает общую ширину всех колонок
func (sm *SizeManager) GetTotalWidth() float32 {
	return sm.totalWidth
}

// CalculateActualPixelSize вычисляет фактический размер карточки по ее содержимому
func (sm *SizeManager) CalculateActualPixelSize(widget fyne.CanvasObject) (float32, float32) {
	startTime := time.Now()

	if widget == nil {
		fmt.Printf("[%s] Widget is nil, returning default size\n", time.Now().Format("15:04:05.000"))
		return sm.fixedWidth, sm.minHeight
	}

	// Получаем предпочтительный размер виджета
	sizeCalcStart := time.Now()
	preferredSize := widget.MinSize()
	fmt.Printf("[%s] MinSize calculation took %v\n", time.Now().Format("15:04:05.000"), time.Since(sizeCalcStart))

	// Устанавливаем фиксированную ширину и переменную высоту
	width := sm.fixedWidth
	maybeAnomalousHeight := preferredSize.Height

	// Проверяем, не является ли высота аномальной (например, для изображений)
	if maybeAnomalousHeight > utils.AnomalousHeightThreshold {
		// Если высота аномально большая, устанавливаем разумное значение
		maybeAnomalousHeight = utils.DefaultAnomalousHeight
	}

	height := maybeAnomalousHeight
	if height < sm.minHeight {
		height = sm.minHeight
	}

	fmt.Printf("[%s] Size calculation completed in %v\n", time.Now().Format("15:04:05.000"), time.Since(startTime))
	return width, height
}

// CalculateColumnCount вычисляет количество колонок на основе доступной ширины
func (sm *SizeManager) CalculateColumnCount(availableWidth float32) int {
	if availableWidth <= 0 {
		return sm.defaultColumnCount // Возвращаем количество колонок по умолчанию
	}

	// Рассчитываем количество колонок: (ширина_доступная - все_промежутки) / ширина_одной_колонки
	// Формула: n = (availableWidth + gapSize) / (fixedWidth + gapSize)
	// где n - количество колонок, которое умещается в доступной ширине
	columnCount := int((availableWidth + sm.gapSize) / (sm.fixedWidth + sm.gapSize))

	if columnCount <= 0 {
		return 1 // Всегда должно быть хотя бы 1 колонка
	}

	return columnCount
}
