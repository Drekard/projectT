package interfaces

import (
	"projectT/internal/storage/database/models"

	"fyne.io/fyne/v2"
)

// CardRenderer интерфейс для рендеринга карточек
type CardRenderer interface {
	GetItem() *models.Item
	GetContainer() fyne.CanvasObject
	SetContainer(fyne.CanvasObject)
	UpdateContent()
	GetWidget() fyne.CanvasObject
}
