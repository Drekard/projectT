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
		log.Printf("[App] P2P порт из config: %d", cfg.P2P.Port)
	}

	// Инициализируем Prometheus метрики
	metricsMgr := metrics.Init()
	log.Printf("[App] Prometheus конфиг: enabled=%v, port=%d, path=%s",
		cfg.Prometheus.Enabled, cfg.Prometheus.Port, cfg.Prometheus.Path)

	if cfg.Prometheus.Enabled {
		// Устанавливаем Prometheus registry в P2P сеть для libp2p метрик
		if cfg.Prometheus.EnableP2PMetrics {
			p2pNetwork.SetPrometheusRegistry(metricsMgr.Registry)
		}

		// Запускаем HTTP сервер метрик
		if err := metricsMgr.InitServer(true, cfg.Prometheus.Port, cfg.Prometheus.Path); err != nil {
			log.Printf("[App] Предупреждение: ошибка запуска Prometheus сервера: %v", err)
		} else {
			log.Printf("[App] Prometheus сервер инициализирован на порту %d", cfg.Prometheus.Port)
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

	// Останавливаем Prometheus сервер при выходе
	if a.metricsMgr != nil {
		if err := a.metricsMgr.Stop(); err != nil {
			log.Printf("[App] Предупреждение: ошибка остановки Prometheus: %v", err)
		}
	}
}

// GetConfig возвращает текущую конфигурацию приложения
func (a *App) GetConfig() *config.Config {
	return a.config
}

// GetP2PNetwork возвращает P2P сеть
func (a *App) GetP2PNetwork() *core.P2PNetwork {
	return a.p2pNetwork
}
