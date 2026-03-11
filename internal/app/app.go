package app

import (
	"log"

	"projectT/internal/config"
	"projectT/internal/services/p2p/network"
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
	p2pNetwork *network.P2PNetwork
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

	// Инициализируем P2P сеть
	p2pNetwork := network.NewP2PNetwork()

	return &App{
		fyneApp:    fyneApp,
		mainWindow: window,
		UI:         nil,
		config:     cfg,
		p2pNetwork: p2pNetwork,
	}
}

func (a *App) Run() {
	a.fyneApp.Settings().SetTheme(theme.GetFyneTheme())

	// Запускаем P2P если включён в конфигурации
	if a.config.P2P.Enabled {
		if err := a.p2pNetwork.Start(); err != nil {
			log.Printf("Предупреждение: P2P не запущен: %v", err)
		} else {
			log.Println("P2P запущен")
		}
	}

	a.UI = ui.NewUI(a.mainWindow, a.p2pNetwork)
	a.mainWindow.ShowAndRun()

	// Останавливаем P2P при выходе
	if a.p2pNetwork != nil {
		if err := a.p2pNetwork.Stop(); err != nil {
			log.Printf("Предупреждение: ошибка остановки P2P: %v", err)
		}
	}
}

// GetConfig возвращает текущую конфигурацию приложения
func (a *App) GetConfig() *config.Config {
	return a.config
}

// GetP2PNetwork возвращает P2P сеть
func (a *App) GetP2PNetwork() *network.P2PNetwork {
	return a.p2pNetwork
}
