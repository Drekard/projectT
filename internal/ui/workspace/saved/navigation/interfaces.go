package navigation

import (
	"fyne.io/fyne/v2/container"
)

// NavigationHandlerInterface интерфейс для обработки навигации
type NavigationHandlerInterface interface {
	SetupNavigation(scroll *container.Scroll)
	OnSizeChanged(pos interface{})
	NavigateToFolder(folderID int) error
}

// Navigator интерфейс для навигации
type Navigator interface {
	GoToFolder(folderID int) error
	GetCurrentFolder() int
}
