package app

import (
	"log"
	"os"
	"path/filepath"

	"projectT/internal/config"
	"projectT/internal/metrics"
	"projectT/internal/services/p2p/core"
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
	p2pNetwork *core.P2PNetwork
	metricsMgr *metrics.Manager
}

func NewApp() *App {
	// Загружаем конфигурацию
	loader := config.NewLoader()
	cfg, err := loader.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Инициализируем базу данных с конфигурацией
	database.InitDBWithConfig(cfg.Database)
	database.RunMigrations()

	// Инициализируем файловое хранилище с конфигурацией
	filesystem.InitStorage(cfg.Storage)

	fyneApp := fyneApp.New()

	window := fyneApp.NewWindow("ㅤ")
	window.Resize(fyne.NewSize(1200, 600))

	// Загружаем иконку: сначала рядом с .exe, затем из assets/icons/
	exePath, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	iconPath := filepath.Join(exePath, "ProjctT.png")
	iconRes, _ := fyne.LoadResourceFromPath(iconPath)

	// Если не найдено рядом с exe, пробуем assets/icons/ (для go run)
	if iconRes == nil {
		iconPath = filepath.Join("assets", "icons", "ProjctT.png")
		iconRes, _ = fyne.LoadResourceFromPath(iconPath)
	}

	window.SetIcon(iconRes)

	// Инициализируем P2P сеть
	p2pNetwork := core.NewP2PNetwork()

	// Устанавливаем порт из конфигурации
	if cfg.P2P.Port > 0 {
		p2pNetwork.SetPort(cfg.P2P.Port)
	}

	// Инициализируем Prometheus метрики
	metricsMgr := metrics.Init()

	if cfg.Prometheus.Enabled {
		// Устанавливаем Prometheus registry в P2P сеть для libp2p метрик
		if cfg.Prometheus.EnableP2PMetrics {
			p2pNetwork.SetPrometheusRegistry(metricsMgr.Registry)
		}

		// Запускаем HTTP сервер метрик
		if err := metricsMgr.InitServer(true, cfg.Prometheus.Port, cfg.Prometheus.Path); err != nil {
			_ = err // Ignore error
		}
	}

	return &App{
		fyneApp:    fyneApp,
		mainWindow: window,
		UI:         nil,
		config:     cfg,
		p2pNetwork: p2pNetwork,
		metricsMgr: metricsMgr,
	}
}

func (a *App) Run() {
	a.fyneApp.Settings().SetTheme(theme.GetFyneTheme())
	a.UI = ui.NewUI(a.mainWindow, a.p2pNetwork)

	// Запускаем P2P сеть
	if err := a.p2pNetwork.Start(); err != nil {
		log.Printf("[P2P] Ошибка запуска P2P сети: %v", err)
	} else {
		log.Printf("[P2P] ✅ P2P сеть запущена")
	}

	a.mainWindow.ShowAndRun()
}

// GetConfig возвращает текущую конфигурацию приложения
func (a *App) GetConfig() *config.Config {
	return a.config
}

// GetP2PNetwork возвращает P2P сеть
func (a *App) GetP2PNetwork() *core.P2PNetwork {
	return a.p2pNetwork
}
