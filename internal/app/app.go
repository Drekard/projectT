package app

import (
	"log"

	"projectT/internal/config"
	"projectT/internal/storage/database"
	"projectT/internal/storage/filesystem"
	"projectT/internal/ui"
	"projectT/internal/ui/theme"

	"fyne.io/fyne/v2"
	fyneApp "fyne.io/fyne/v2/app"
)

type App struct {
	fyneApp    fyne.App
	mainWindow fyne.Window
	UI         *ui.UI
	config     *config.Config
}

func NewApp() *App {
	// Загружаем конфигурацию
	loader := config.NewLoader()
	cfg, err := loader.Load()
	if err != nil {
		log.Printf("Предупреждение: ошибка при загрузке конфигурации: %v", err)
		cfg = config.DefaultConfig()
	}

	// Инициализируем базу данных с конфигурацией
	database.InitDBWithConfig(cfg.Database)
	database.RunMigrations()

	// Инициализируем файловое хранилище с конфигурацией
	filesystem.InitStorage(cfg.Storage)

	fyneApp := fyneApp.New()

	window := fyneApp.NewWindow("ㅤ")
	window.Resize(fyne.NewSize(1110, 600))

	iconRes, _ := fyne.LoadResourceFromPath("./assets/icons/ProjctT.png")
	window.SetIcon(iconRes)

	return &App{
		fyneApp:    fyneApp,
		mainWindow: window,
		UI:         nil,
		config:     cfg,
	}
}

func (a *App) Run() {
	a.fyneApp.Settings().SetTheme(theme.GetFyneTheme())

	a.UI = ui.NewUI(a.mainWindow)
	a.mainWindow.ShowAndRun()
}

// GetConfig возвращает текущую конфигурацию приложения
func (a *App) GetConfig() *config.Config {
	return a.config
}
