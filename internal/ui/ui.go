package ui

import (
	"projectT/internal/services/p2p/network"
	"projectT/internal/ui/layout"

	"fyne.io/fyne/v2"
)

// UI представляет собой основной интерфейс приложения
type UI struct {
	mainLayout *fyne.Container
	window     fyne.Window
}

// NewUI создает новый экземпляр UI
func NewUI(window fyne.Window, p2pNetwork *network.P2PNetwork) *UI {
	window.SetPadded(false)

	ui := &UI{
		mainLayout: layout.CreateMainLayout(window, p2pNetwork),
		window:     window,
	}

	ui.window.SetContent(ui.mainLayout)

	return ui
}
