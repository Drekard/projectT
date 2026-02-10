package models

import (
	"projectT/internal/storage/database/models"

	"fyne.io/fyne/v2"
)

// CardInfo хранит информацию о карточке элемента
type CardInfo struct {
	Item         *models.Item
	Widget       fyne.CanvasObject
	Position     CellPosition
	WidthCells   int
	HeightCells  int
	ActualHeight float32 // Фактическая высота карточки после рендеринга
}

// CellPosition позиция карточки в сетке
type CellPosition struct {
	X, Y int
}

// CardSize размер карточки в ячейках
type CardSize struct {
	Width, Height int
}

// GridManagerNavigationHandler интерфейс для обработки навигации
type GridManagerNavigationHandler interface {
	NavigateToFolder(folderID int) error
}
