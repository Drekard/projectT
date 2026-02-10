package sizing

import (
	"fyne.io/fyne/v2/container"
)

// CardInfo структура информации о карточке (временное определение до рефакторинга)
type CardInfo struct {
	// Поля будут определены после рефакторинга
}

// CellPosition структура позиции ячейки (временное определение до рефакторинга)
type CellPosition struct {
	X, Y int
}

// SizeManagerInterface интерфейс для управления размерами карточек
type SizeManagerInterface interface {
	CalculatePixelSize(widthCells, heightCells int) (float32, float32)
	CalculatePixelPosition(x, y int) (float32, float32)
	GetAvailableWidth(scroll *container.Scroll) int
	CalculateMaxDimensions(cards []*CardInfo) (float32, float32)
}

// SizeCalculator интерфейс для вычисления размеров
type SizeCalculator interface {
	GetCellSize() int
	GetGapSize() int
	GetCellAndGapSize() int
}
