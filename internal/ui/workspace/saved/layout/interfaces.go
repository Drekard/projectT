package layout

import (
	"projectT/internal/ui/workspace/saved/models"
)

// LayoutEngineInterface интерфейс для расчета позиций карточек
type LayoutEngineInterface interface {
	CalculatePositions(cards []*models.CardInfo, availableWidth int) []models.CellPosition
}

// PositionCalculator интерфейс для расчета позиций
type PositionCalculator interface {
	FindPosition(widthCells, heightCells int) (int, int)
	CanPlaceAt(x, y, widthCells, heightCells int) bool
	MarkCellsAsOccupied(x, y, widthCells, heightCells int)
}
