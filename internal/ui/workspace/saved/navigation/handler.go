package navigation

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// NavigationHandler обрабатывает навигацию между папками
type NavigationHandler struct {
	callback func(int) error // функция обратного вызова для навигации
}

// NewNavigationHandler создает новый обработчик навигации
func NewNavigationHandler(callback func(int) error) *NavigationHandler {
	return &NavigationHandler{
		callback: callback,
	}
}

// SetupNavigation настраивает навигацию
func (nh *NavigationHandler) SetupNavigation(scroll *container.Scroll) {
	// Настройка обработчиков событий
	scroll.OnScrolled = nh.OnSizeChanged
}

// OnSizeChanged обработчик изменения размера
func (nh *NavigationHandler) OnSizeChanged(pos fyne.Position) {
	// В реальной реализации здесь будет вызов обновления макета
}

// NavigateToFolder выполняет переход к папке
func (nh *NavigationHandler) NavigateToFolder(folderID int) error {
	if nh.callback != nil {
		return nh.callback(folderID)
	}
	return nil
}
