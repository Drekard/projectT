package app

import (
	"projectT/internal/storage/database"
	"projectT/internal/storage/filesystem"
	"projectT/internal/ui"
	"projectT/internal/ui/theme"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

type App struct {
	fyneApp    fyne.App
	mainWindow fyne.Window
	UI         *ui.UI
}

func NewApp() *App {
	database.InitDB()
	database.RunMigrations()

	filesystem.EnsureStorageStructure()

	fyneApp := app.New()

	window := fyneApp.NewWindow("ㅤ")
	window.Resize(fyne.NewSize(1110, 600))

	iconRes, _ := fyne.LoadResourceFromPath("./assets/icons/ProjctT.png")
	window.SetIcon(iconRes)

	return &App{
		fyneApp:    fyneApp,
		mainWindow: window,
		UI:         nil,
	}
}

func (a *App) Run() {
	a.fyneApp.Settings().SetTheme(theme.GetFyneTheme())

	a.UI = ui.NewUI(a.mainWindow)
	a.mainWindow.ShowAndRun()
}
