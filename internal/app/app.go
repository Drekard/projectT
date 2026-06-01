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
	"projectT/internal/tray"
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
	configPath string
	p2pNetwork *core.P2PNetwork
	metricsMgr *metrics.Manager
	trayMgr    *tray.Manager
	enableTray bool
}

func NewApp() *App {
	// Загружаем конфигурацию
	loader := config.NewLoader()
	cfg, err := loader.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Определяем путь к конфигу для сохранения
	configPath := "config.yaml"
	if _, statErr := os.Stat("config.yaml"); statErr != nil {
		configPath = ""
	}

	// Инициализируем базу данных с конфигурацией
	database.InitDBWithConfig(cfg.Database)
	database.RunMigrations()

	// Инициализируем файловое хранилище с конфигурацией
	filesystem.InitStorage(cfg.Storage)

	// Восстанавливаем тему из конфига
	if cfg.UISettings.Theme != "" {
		theme.SetTheme(theme.ThemeNameToAppTheme(cfg.UISettings.Theme))
	}

	fyneApp := fyneApp.New()

	window := fyneApp.NewWindow("ㅤ")

	// Восстанавливаем размер и позицию окна из конфига
	if cfg.UISettings.WindowWidth > 0 && cfg.UISettings.WindowHeight > 0 {
		window.Resize(fyne.NewSize(cfg.UISettings.WindowWidth, cfg.UISettings.WindowHeight))
	} else {
		window.Resize(fyne.NewSize(1200, 600))
	}

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

	// Устанавливаем настройки автоподключения и обмена профилями
	p2pNetwork.SetAutoConnect(cfg.P2P.EnableAutoConnect)
	p2pNetwork.SetAutoProfileEx(cfg.P2P.EnableAutoProfileEx)

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
		configPath: configPath,
		p2pNetwork: p2pNetwork,
		metricsMgr: metricsMgr,
	}
}

func (a *App) Run() {
	a.fyneApp.Settings().SetTheme(theme.GetFyneTheme())
	a.UI = ui.NewUI(a.mainWindow, a.p2pNetwork, a.config, a.saveUISettings)

	// Устанавливаем обработчик закрытия окна для сохранения позиции и размера
	a.mainWindow.SetCloseIntercept(func() {
		a.saveWindowGeometry()
		if a.enableTray {
			a.mainWindow.Hide()
		} else {
			a.mainWindow.Close()
		}
	})

	// Запускаем системный трей если включен
	if a.enableTray {
		a.trayMgr = tray.New(a.mainWindow)
		a.trayMgr.Start()
	}

	// Запускаем P2P сеть асинхронно, чтобы UI не зависал
	go func() {
		if err := a.p2pNetwork.Start(); err != nil {
			log.Printf("[P2P] Ошибка запуска P2P сети: %v", err)
		}
	}()

	a.mainWindow.ShowAndRun()
}

// EnableTray включает сворачивание в системный трей
func (a *App) EnableTray() {
	a.enableTray = true
}

// saveUISettings сохраняет настройки UI в конфиг
func (a *App) saveUISettings() {
	if a.configPath == "" {
		return
	}
	if err := config.Save(a.config, a.configPath); err != nil {
		log.Printf("[Config] Ошибка сохранения настроек UI: %v", err)
	}
}

// saveWindowGeometry сохраняет позицию и размер окна
func (a *App) saveWindowGeometry() {
	if a.configPath == "" {
		return
	}

	size := a.mainWindow.Canvas().Size()
	a.config.UISettings.WindowWidth = size.Width
	a.config.UISettings.WindowHeight = size.Height

	// Fyne не предоставляет прямой метод для получения позиции окна
	// Позиция сохраняется только если она была восстановлена при запуске
	// Для полного сохранения позиции потребуется использование нативных API

	if err := config.Save(a.config, a.configPath); err != nil {
		log.Printf("[Config] Ошибка сохранения геометрии окна: %v", err)
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
