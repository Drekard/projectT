package concrete

import (
	"fmt"
	"image/color"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/filesystem"
	"projectT/internal/ui/cards"
	"projectT/internal/ui/cards/hover_preview"
	"projectT/internal/ui/cards/interfaces"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// VideoCard карточка для видеофайлов
type VideoCard struct {
	*cards.BaseCard
	videoFiles      []*cards.Block
	selectedFileIdx int
	currentFileIdx  int
	isPlaying       bool
	isPaused        bool
	playBtn         *widget.Button
	progress        *widget.ProgressBar
	timeLabel       *widget.Label
	durationLabel   *widget.Label
	volumeSlider    *widget.Slider
	stopChan        chan struct{}
	useVLC          bool
}

// NewVideoCard создает новую карточку для видео
func NewVideoCard(item *models.Item) interfaces.CardRenderer {
	return NewVideoCardWithCallback(item, nil)
}

// NewVideoCardWithCallback создает новую карточку для видео с пользовательским обработчиком клика
func NewVideoCardWithCallback(item *models.Item, clickCallback func()) interfaces.CardRenderer {
	videoCard := &VideoCard{
		BaseCard:        cards.NewBaseCard(item),
		videoFiles:      make([]*cards.Block, 0),
		selectedFileIdx: -1,
		currentFileIdx:  0,
		isPlaying:       false,
		isPaused:        false,
		stopChan:        make(chan struct{}),
		useVLC:          false, // По умолчанию используем системный плеер
	}

	// Извлекаем все видеофайлы из ContentMeta
	blocks, err := cards.ParseBlocks(videoCard.Item.ContentMeta)
	if err != nil {
		fmt.Printf("[ERROR] Ошибка парсинга ContentMeta: %v\n", err)
	}

	// Собираем видеофайлы
	for _, block := range blocks {
		if block.Type == "video" && block.FileHash != "" {
			videoCard.videoFiles = append(videoCard.videoFiles, &block)
		}
	}

	// Если нет видеофайлов, показываем заглушку
	if len(videoCard.videoFiles) == 0 {
		placeholder := widget.NewLabel("Видеофайлы не найдены")
		placeholder.Alignment = fyne.TextAlignCenter
		videoCard.Container = container.NewCenter(placeholder)
		return videoCard
	}

	// Пытаемся инициализировать VLC
	err = videoCard.initVLC()
	if err != nil {
		fmt.Printf("[WARN] VLC недоступен, используем системный плеер: %v\n", err)
		videoCard.useVLC = false
	}

	// Создаем UI для первого видеофайла
	videoCard.createVideoUI(0)

	return videoCard
}

// initVLC инициализирует VLC instance
func (vc *VideoCard) initVLC() error {
	// Примечание: Для полноценной работы VLC требуется установленный VLC Player
	// и соответствующие переменные окружения (VLC_PLUGIN_PATH и т.д.)
	// В данной реализации VLC отключен для упрощения развертывания
	
	// Если VLC нужен, раскомментируйте и установите VLC:
	// https://www.videolan.org/vlc/
	// После установки добавьте путь к VLC в PATH
	
	return fmt.Errorf("VLC не подключен (требуется установка VLC)")
}

// createVideoUI создает интерфейс для воспроизведения видео
func (vc *VideoCard) createVideoUI(fileIndex int) {
	if fileIndex < 0 || fileIndex >= len(vc.videoFiles) {
		return
	}

	block := vc.videoFiles[fileIndex]

	// Получаем имя файла для отображения
	fileName := vc.getDisplayName(block)

	// Создаем иконку видео
	videoIcon := widget.NewIcon(theme.MediaVideoIcon())
	iconCircle := canvas.NewCircle(color.RGBA{R: 0, G: 123, B: 255, A: 255})
	iconContainer := container.NewStack(iconCircle, videoIcon)

	// Информация о файле
	nameLabel := widget.NewRichText(&widget.TextSegment{
		Text: fileName,
		Style: widget.RichTextStyle{
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})
	nameLabel.Truncation = fyne.TextTruncateEllipsis

	// Прогресс бар
	vc.progress = widget.NewProgressBar()
	vc.progress.Value = 0

	// Метки времени
	vc.timeLabel = widget.NewLabel("0:00")
	vc.durationLabel = widget.NewLabel("--:--")

	// Кнопка Play/Pause
	vc.playBtn = widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
		vc.togglePlayPause()
	})
	vc.playBtn.Importance = widget.HighImportance

	// Кнопка предыдущего трека
	prevBtn := widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() {
		vc.playPrevious()
	})
	prevBtn.Importance = widget.LowImportance

	// Кнопка следующего трека
	nextBtn := widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {
		vc.playNext()
	})
	nextBtn.Importance = widget.LowImportance

	// Регулятор громкости
	vc.volumeSlider = widget.NewSlider(0, 100)
	vc.volumeSlider.Value = 80
	vc.volumeSlider.OnChanged = func(value float64) {
		vc.setVolume(value)
	}

	// Контейнер управления
	controlsContainer := container.NewHBox(
		prevBtn,
		vc.playBtn,
		nextBtn,
	)

	// Контейнер прогресса
	progressContainer := container.NewBorder(
		nil,
		nil,
		container.NewHBox(vc.timeLabel),
		container.NewHBox(vc.durationLabel),
		vc.progress,
	)

	// Основная информация
	infoContainer := container.NewVBox(
		nameLabel,
		progressContainer,
		container.NewHBox(controlsContainer, container.NewHBox(vc.volumeSlider)),
	)

	// Общий контейнер
	mainContent := container.NewHBox(iconContainer, infoContainer)

	// Оборачиваем в кликабельный виджет
	vc.Container = hover_preview.NewClickableCardWithDoubleTap(
		mainContent,
		func() {
			// Одинарный клик - показываем меню
			menuManager := hover_preview.NewMenuManager()
			menuManager.ShowSimpleMenu(vc.Item, vc.Container, nil)
		},
		func() {
			// Двойной клик - открываем файл в проводнике
			vc.openCurrentFileWithDefaultApp()
		},
	)

	// Загружаем видео для получения длительности
	vc.loadVideoInfo(block)
}

// getDisplayName возвращает отображаемое имя файла
func (vc *VideoCard) getDisplayName(block *cards.Block) string {
	if block.OriginalName != "" {
		lastSlash := strings.LastIndex(block.OriginalName, "\\")
		if lastSlash == -1 {
			lastSlash = strings.LastIndex(block.OriginalName, "/")
		}
		if lastSlash != -1 {
			return block.OriginalName[lastSlash+1:]
		}
		return block.OriginalName
	}

	if block.Extension != "" {
		return block.FileHash + "." + block.Extension
	}
	return block.FileHash
}

// loadVideoInfo загружает информацию о видеофайле (длительность)
func (vc *VideoCard) loadVideoInfo(block *cards.Block) {
	filePath := filesystem.GetFilePathByHash(block.FileHash)
	if filePath == "" {
		vc.durationLabel.SetText("--:--")
		return
	}

	// Для системного плеера не получаем длительность
	// Показываем заглушку
	vc.durationLabel.SetText("--:--")
	
	// Примечание: Для получения длительности можно использовать:
	// 1. ffprobe (часть ffmpeg) - требует установки ffmpeg
	// 2. VLC - требует установки VLC
	// 3. Чтение метаданных через Go библиотеки
}

// togglePlayPause переключает воспроизведение
func (vc *VideoCard) togglePlayPause() {
	if vc.currentFileIdx < 0 || vc.currentFileIdx >= len(vc.videoFiles) {
		return
	}

	if vc.isPaused {
		vc.resume()
		return
	}

	if vc.isPlaying {
		vc.pause()
		return
	}

	block := vc.videoFiles[vc.currentFileIdx]
	filePath := filesystem.GetFilePathByHash(block.FileHash)
	if filePath == "" {
		return
	}

	vc.play(filePath)
}

// play начинает воспроизведение через системный плеер
func (vc *VideoCard) play(filePath string) {
	// Открываем видео в системном плеере Windows
	cmd := exec.Command("cmd", "/c", "start", "", filePath)
	err := cmd.Start()
	if err != nil {
		fmt.Printf("[ERROR] Ошибка запуска видео: %v\n", err)
		return
	}

	vc.isPlaying = true
	vc.isPaused = false

	// Обновляем UI
	vc.playBtn.SetIcon(theme.MediaPauseIcon())
	vc.playBtn.Refresh()

	// Для системного плеера не отслеживаем прогресс
	// Просто показываем, что воспроизведение началось
	go func() {
		// Ждём немного и сбрасываем статус
		time.Sleep(2 * time.Second)
		vc.isPlaying = false
		fyne.CurrentApp().Driver().AllWindows()[0].Canvas().Refresh(vc.playBtn)
	}()
}

// pause приостанавливает воспроизведение
func (vc *VideoCard) pause() {
	// Для системного плеера пауза не поддерживается
	// Просто обновляем состояние
	vc.isPaused = true
	vc.isPlaying = false

	vc.playBtn.SetIcon(theme.MediaPlayIcon())
	vc.playBtn.Refresh()
}

// resume возобновляет воспроизведение
func (vc *VideoCard) resume() {
	// Для системного плеера возобновление не поддерживается
	// Просто обновляем состояние
	vc.isPaused = false
	vc.isPlaying = true

	vc.playBtn.SetIcon(theme.MediaPauseIcon())
	vc.playBtn.Refresh()
}

// stop останавливает воспроизведение
func (vc *VideoCard) stop() {
	vc.isPlaying = false
	vc.isPaused = false

	// Сигнализируем о остановке
	select {
	case <-vc.stopChan:
	default:
		close(vc.stopChan)
	}
	vc.stopChan = make(chan struct{})

	vc.playBtn.SetIcon(theme.MediaPlayIcon())
	vc.playBtn.Refresh()
	vc.progress.Value = 0
	vc.progress.Refresh()
	vc.timeLabel.SetText("0:00")
	vc.timeLabel.Refresh()
}

// updateProgress обновляет прогресс воспроизведения
func (vc *VideoCard) updateProgress() {
	// Для системного плеера прогресс не отслеживается
}

// setVolume устанавливает громкость
func (vc *VideoCard) setVolume(value float64) {
	// Для системного плеера громкость не управляется
	_ = value
}

// playPrevious переключает на предыдущий трек
func (vc *VideoCard) playPrevious() {
	if len(vc.videoFiles) <= 1 {
		return
	}

	// Останавливаем текущее воспроизведение
	vc.stop()

	vc.currentFileIdx--
	if vc.currentFileIdx < 0 {
		vc.currentFileIdx = len(vc.videoFiles) - 1
	}

	vc.createVideoUI(vc.currentFileIdx)
}

// playNext переключает на следующий трек
func (vc *VideoCard) playNext() {
	if len(vc.videoFiles) <= 1 {
		return
	}

	// Останавливаем текущее воспроизведение
	vc.stop()

	vc.currentFileIdx++
	if vc.currentFileIdx >= len(vc.videoFiles) {
		vc.currentFileIdx = 0
	}

	vc.createVideoUI(vc.currentFileIdx)
}

// openCurrentFileWithDefaultApp открывает текущий файл в проводнике
func (vc *VideoCard) openCurrentFileWithDefaultApp() {
	if vc.currentFileIdx < 0 || vc.currentFileIdx >= len(vc.videoFiles) {
		return
	}

	block := vc.videoFiles[vc.currentFileIdx]
	filePath := filesystem.GetFilePathByHash(block.FileHash)
	if filePath == "" {
		return
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	// Открываем через explorer.exe
	cmd := exec.Command("explorer.exe", absPath)
	cmd.Run()
}

// formatDurationVideo форматирует длительность в HH:MM:SS или MM:SS
func formatDurationVideo(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// Методы интерфейса CardRenderer
func (vc *VideoCard) GetContainer() fyne.CanvasObject {
	return vc.Container
}

func (vc *VideoCard) GetWidget() fyne.CanvasObject {
	return vc.Container
}

func (vc *VideoCard) SetContainer(container fyne.CanvasObject) {
	vc.Container = container
}

func (vc *VideoCard) UpdateContent() {
	newCard := NewVideoCardWithCallback(vc.Item, nil)
	vc.Container = newCard.GetContainer()
}
