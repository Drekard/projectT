package concrete

import (
	"fmt"
	"image/color"
	"os"
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

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/vorbis"
	"github.com/gopxl/beep/v2/wav"
)

// AudioCard карточка для аудиофайлов
type AudioCard struct {
	*cards.BaseCard
	audioFiles      []*cards.Block
	selectedFileIdx int
	currentFileIdx  int
	isPlaying       bool
	isPaused        bool
	playBtn         *widget.Button
	progress        *widget.ProgressBar
	timeLabel       *widget.Label
	durationLabel   *widget.Label
	volumeSlider    *widget.Slider

	// Аудио плеер
	ctrl     *beep.Ctrl
	streamer beep.Streamer //nolint:unused
	stopChan chan struct{}
}

// NewAudioCard создает новую карточку для аудио
func NewAudioCard(item *models.Item) interfaces.CardRenderer {
	return NewAudioCardWithCallback(item, nil)
}

// NewAudioCardWithCallback создает новую карточку для аудио с пользовательским обработчиком клика
func NewAudioCardWithCallback(item *models.Item, clickCallback func()) interfaces.CardRenderer {
	audioCard := &AudioCard{
		BaseCard:        cards.NewBaseCard(item),
		audioFiles:      make([]*cards.Block, 0),
		selectedFileIdx: -1,
		currentFileIdx:  0,
		isPlaying:       false,
		isPaused:        false,
		stopChan:        make(chan struct{}),
	}

	// Извлекаем все аудиофайлы из ContentMeta
	blocks, err := cards.ParseBlocks(audioCard.Item.ContentMeta)
	if err != nil {
		fmt.Printf("[ERROR] Ошибка парсинга ContentMeta: %v\n", err)
	}

	// Собираем аудиофайлы
	for _, block := range blocks {
		if block.Type == "audio" && block.FileHash != "" {
			audioCard.audioFiles = append(audioCard.audioFiles, &block)
		}
	}

	// Если нет аудиофайлов, показываем заглушку
	if len(audioCard.audioFiles) == 0 {
		placeholder := widget.NewLabel("Аудиофайлы не найдены")
		placeholder.Alignment = fyne.TextAlignCenter
		audioCard.Container = container.NewCenter(placeholder)
		return audioCard
	}

	// Создаем UI для первого аудиофайла
	audioCard.createAudioUI(0)

	return audioCard
}

// createAudioUI создает интерфейс для воспроизведения аудио
func (ac *AudioCard) createAudioUI(fileIndex int) {
	if fileIndex < 0 || fileIndex >= len(ac.audioFiles) {
		return
	}

	block := ac.audioFiles[fileIndex]

	// Получаем имя файла для отображения
	fileName := ac.getDisplayName(block)

	// Создаем иконку аудио
	audioIcon := widget.NewIcon(theme.MediaMusicIcon())
	iconCircle := canvas.NewCircle(color.RGBA{R: 0, G: 123, B: 255, A: 255})
	iconContainer := container.NewStack(iconCircle, audioIcon)

	// Информация о файле
	nameLabel := widget.NewRichText(&widget.TextSegment{
		Text: fileName,
		Style: widget.RichTextStyle{
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})
	nameLabel.Truncation = fyne.TextTruncateEllipsis

	// Прогресс бар
	ac.progress = widget.NewProgressBar()
	ac.progress.Value = 0

	// Метки времени
	ac.timeLabel = widget.NewLabel("0:00")
	ac.durationLabel = widget.NewLabel("--:--")

	// Кнопка Play/Pause
	ac.playBtn = widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
		ac.togglePlayPause()
	})
	ac.playBtn.Importance = widget.HighImportance

	// Кнопка предыдущего трека
	prevBtn := widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() {
		ac.playPrevious()
	})
	prevBtn.Importance = widget.LowImportance

	// Кнопка следующего трека
	nextBtn := widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {
		ac.playNext()
	})
	nextBtn.Importance = widget.LowImportance

	// Регулятор громкости
	ac.volumeSlider = widget.NewSlider(0, 100)
	ac.volumeSlider.Value = 80
	ac.volumeSlider.OnChanged = func(value float64) {
		ac.setVolume(value)
	}

	// Контейнер управления
	controlsContainer := container.NewHBox(
		prevBtn,
		ac.playBtn,
		nextBtn,
	)

	// Контейнер прогресса
	progressContainer := container.NewBorder(
		nil,
		nil,
		container.NewHBox(ac.timeLabel),
		container.NewHBox(ac.durationLabel),
		ac.progress,
	)

	// Основная информация
	infoContainer := container.NewVBox(
		nameLabel,
		progressContainer,
		container.NewHBox(controlsContainer, container.NewHBox(ac.volumeSlider)),
	)

	// Общий контейнер
	mainContent := container.NewHBox(iconContainer, infoContainer)

	// Оборачиваем в кликабельный виджет
	ac.Container = hover_preview.NewClickableCardWithDoubleTap(
		mainContent,
		func() {
			// Одинарный клик - показываем меню
			menuManager := hover_preview.NewMenuManager()
			menuManager.ShowSimpleMenu(ac.Item, ac.Container, nil)
		},
		func() {
			// Двойной клик - открываем файл в проводнике
			ac.openCurrentFileWithDefaultApp()
		},
	)

	// Загружаем аудио для получения длительности
	ac.loadAudioInfo(block)
}

// getDisplayName возвращает отображаемое имя файла
func (ac *AudioCard) getDisplayName(block *cards.Block) string {
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

// loadAudioInfo загружает информацию об аудиофайле (длительность)
func (ac *AudioCard) loadAudioInfo(block *cards.Block) {
	filePath := filesystem.GetFilePathByHash(block.FileHash)
	if filePath == "" {
		ac.durationLabel.SetText("--:--")
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		ac.durationLabel.SetText("--:--")
		return
	}
	defer file.Close()

	var streamer beep.StreamSeekCloser
	var format beep.Format

	// Определяем формат по расширению
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".mp3":
		streamer, format, err = mp3.Decode(file)
	case ".wav":
		streamer, format, err = wav.Decode(file)
	case ".ogg":
		streamer, format, err = vorbis.Decode(file)
	default:
		err = fmt.Errorf("неподдерживаемый формат: %s", ext)
	}

	if err != nil {
		fmt.Printf("[WARN] Ошибка декодирования аудио: %v\n", err)
		ac.durationLabel.SetText("--:--")
		return
	}

	// Вычисляем длительность (приблизительно)
	// Для простоты показываем заглушку - реальная длительность требует чтения метаданных
	_ = format
	ac.durationLabel.SetText("--:--")

	// Освобождаем ресурсы
	streamer.Close()
}

// togglePlayPause переключает воспроизведение
func (ac *AudioCard) togglePlayPause() {
	if ac.currentFileIdx < 0 || ac.currentFileIdx >= len(ac.audioFiles) {
		return
	}

	if ac.isPaused {
		ac.resume()
		return
	}

	if ac.isPlaying {
		ac.pause()
		return
	}

	block := ac.audioFiles[ac.currentFileIdx]
	filePath := filesystem.GetFilePathByHash(block.FileHash)
	if filePath == "" {
		return
	}

	ac.play(filePath)
}

// play начинает воспроизведение
func (ac *AudioCard) play(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("[ERROR] Ошибка открытия файла: %v\n", err)
		return
	}

	var streamer beep.Streamer
	var format beep.Format
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".mp3":
		streamer, format, err = mp3.Decode(file)
	case ".wav":
		streamer, format, err = wav.Decode(file)
	case ".ogg":
		streamer, format, err = vorbis.Decode(file)
	default:
		err = fmt.Errorf("неподдерживаемый формат: %s", ext)
	}

	if err != nil {
		fmt.Printf("[ERROR] Ошибка декодирования: %v\n", err)
		file.Close()
		return
	}

	// Инициализируем спикер если нужно
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10)) //nolint:errcheck

	// Создаем контроллер с паузой
	ac.ctrl = &beep.Ctrl{
		Streamer: streamer,
		Paused:   false,
	}

	// Запускаем воспроизведение
	speaker.Play(ac.ctrl)

	ac.isPlaying = true
	ac.isPaused = false

	// Обновляем UI
	ac.playBtn.SetIcon(theme.MediaPauseIcon())
	ac.playBtn.Refresh()

	// Запускаем обновление прогресса
	go ac.updateProgress()
}

// updateProgress обновляет прогресс воспроизведения
func (ac *AudioCard) updateProgress() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	startTime := time.Now()

	for ac.isPlaying && !ac.isPaused {
		select {
		case <-ticker.C:
			if ac.ctrl != nil && !ac.ctrl.Paused {
				elapsed := time.Since(startTime)

				// Получаем длительность из метки
				durationText := ac.durationLabel.Text
				duration := parseDuration(durationText)

				if duration > 0 {
					progress := float64(elapsed.Seconds()) / float64(duration.Seconds())
					if progress > 1.0 {
						progress = 1.0
					}

					ac.progress.Value = progress
					ac.timeLabel.SetText(formatDuration(elapsed))
					ac.progress.Refresh()
					ac.timeLabel.Refresh()

					// Если достигли конца
					if progress >= 1.0 {
						ac.stop()
						return
					}
				}
			}
		case <-ac.stopChan:
			return
		}
	}
}

// pause приостанавливает воспроизведение
func (ac *AudioCard) pause() {
	if ac.ctrl != nil {
		ac.ctrl.Paused = true
	}
	ac.isPaused = true
	ac.isPlaying = false

	ac.playBtn.SetIcon(theme.MediaPlayIcon())
	ac.playBtn.Refresh()
}

// resume возобновляет воспроизведение
func (ac *AudioCard) resume() {
	if ac.ctrl != nil {
		ac.ctrl.Paused = false
	}
	ac.isPaused = false
	ac.isPlaying = true

	ac.playBtn.SetIcon(theme.MediaPauseIcon())
	ac.playBtn.Refresh()

	// Перезапускаем обновление прогресса
	go ac.updateProgress()
}

// stop останавливает воспроизведение
func (ac *AudioCard) stop() {
	ac.isPlaying = false
	ac.isPaused = false

	if ac.ctrl != nil {
		// Останавливаем стример устанавливая Streamer в nil
		speaker.Lock()
		ac.ctrl.Streamer = nil
		speaker.Unlock()
		ac.ctrl = nil
	}

	// Сигнализируем о остановке
	select {
	case <-ac.stopChan:
	default:
		close(ac.stopChan)
	}
	ac.stopChan = make(chan struct{})

	ac.playBtn.SetIcon(theme.MediaPlayIcon())
	ac.playBtn.Refresh()
	ac.progress.Value = 0
	ac.progress.Refresh()
	ac.timeLabel.SetText("0:00")
	ac.timeLabel.Refresh()
}

// setVolume устанавливает громкость
func (ac *AudioCard) setVolume(value float64) {
	// Громкость будет применена при следующем воспроизведении
	_ = value // пока не используется
}

// playPrevious переключает на предыдущий трек
func (ac *AudioCard) playPrevious() {
	if len(ac.audioFiles) <= 1 {
		return
	}

	// Останавливаем текущее воспроизведение
	ac.stop()

	ac.currentFileIdx--
	if ac.currentFileIdx < 0 {
		ac.currentFileIdx = len(ac.audioFiles) - 1
	}

	ac.createAudioUI(ac.currentFileIdx)
}

// playNext переключает на следующий трек
func (ac *AudioCard) playNext() {
	if len(ac.audioFiles) <= 1 {
		return
	}

	// Останавливаем текущее воспроизведение
	ac.stop()

	ac.currentFileIdx++
	if ac.currentFileIdx >= len(ac.audioFiles) {
		ac.currentFileIdx = 0
	}

	ac.createAudioUI(ac.currentFileIdx)
}

// openCurrentFileWithDefaultApp открывает текущий файл в проводнике
func (ac *AudioCard) openCurrentFileWithDefaultApp() {
	if ac.currentFileIdx < 0 || ac.currentFileIdx >= len(ac.audioFiles) {
		return
	}

	block := ac.audioFiles[ac.currentFileIdx]
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
	if err := cmd.Run(); err != nil {
		fmt.Printf("[ERROR] Ошибка при открытии файла: %v\n", err)
	}
}

// formatDuration форматирует длительность в MM:SS
func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// parseDuration парсит длительность из строки MM:SS
func parseDuration(s string) time.Duration {
	var minutes, seconds int
	_, err := fmt.Sscanf(s, "%d:%d", &minutes, &seconds)
	if err != nil {
		return 0
	}
	return time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second
}

// Методы интерфейса CardRenderer
func (ac *AudioCard) GetContainer() fyne.CanvasObject {
	return ac.Container
}

func (ac *AudioCard) GetWidget() fyne.CanvasObject {
	return ac.Container
}

func (ac *AudioCard) SetContainer(container fyne.CanvasObject) {
	ac.Container = container
}

func (ac *AudioCard) UpdateContent() {
	newCard := NewAudioCardWithCallback(ac.Item, nil)
	ac.Container = newCard.GetContainer()
}
