package ui

import (
	"projectT/internal/ui/layout"

	"fyne.io/fyne/v2"
)

// UI представляет собой основной интерфейс приложения
type UI struct {
	mainLayout *fyne.Container
	window     fyne.Window
}

// NewUI создает новый экземпляр UI
func NewUI(window fyne.Window) *UI {
	// Устанавливаем тему приложения
	window.SetPadded(false) // Убираем внутренние отступы

	ui := &UI{
		mainLayout: layout.CreateMainLayout(window),
		window:     window,
	}

	// Устанавливаем основной макет в окно
	ui.window.SetContent(ui.mainLayout)

	return ui
}
