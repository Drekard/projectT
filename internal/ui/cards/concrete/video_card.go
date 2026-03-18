package concrete

import (
	"fmt"
	"image/color"
	"os/exec"
	"path/filepath"
	"strings"

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
	videoFiles           []*cards.Block
	currentFileIdx       int
	isContentInitialized bool // Флаг: контент уже инициализирован
	previewContent       fyne.CanvasObject
	playOverlay          *fyne.Container
}

// NewVideoCard создает новую карточку для видео
func NewVideoCard(item *models.Item) interfaces.CardRenderer {
	return NewVideoCardWithCallback(item, nil)
}

// NewVideoCardWithCallback создает новую карточку для видео с пользовательским обработчиком клика
func NewVideoCardWithCallback(item *models.Item, clickCallback func()) interfaces.CardRenderer {
	videoCard := &VideoCard{
		BaseCard:       cards.NewBaseCard(item),
		videoFiles:     make([]*cards.Block, 0),
		currentFileIdx: 0,
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

	// Создаем UI для первого видеофайла
	videoCard.createVideoUI(0)

	// Устанавливаем флаг, что контент инициализирован
	videoCard.isContentInitialized = true

	return videoCard
}

// createVideoUI создает компактный интерфейс с превью
func (vc *VideoCard) createVideoUI(fileIndex int) {
	if fileIndex < 0 || fileIndex >= len(vc.videoFiles) {
		return
	}

	block := vc.videoFiles[fileIndex]
	filePath := filesystem.GetFilePathByHash(block.FileHash)
	if filePath == "" {
		return
	}

	// Получаем имя файла для отображения
	fileName := vc.getDisplayName(block)

	// Создаем превью (заглушку) для видео
	// В будущем можно генерировать реальный кадр из видео через ffmpeg
	vc.previewContent = vc.createVideoPreview(filePath)

	// Создаем оверлей с кнопкой Play по центру
	playBtn := widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
		vc.openCurrentFileWithDefaultApp()
	})
	playBtn.Importance = widget.HighImportance

	// Полупрозрачный фон для кнопки
	playBg := canvas.NewCircle(color.RGBA{R: 0, G: 0, B: 0, A: 180})
	playOverlayContent := container.NewStack(playBg, playBtn)
	vc.playOverlay = container.NewCenter(playOverlayContent)

	// Контейнер с превью и оверлеем
	previewContainer := container.NewStack(vc.previewContent, vc.playOverlay)

	// Нижняя панель с информацией
	infoLabel := widget.NewRichText(&widget.TextSegment{
		Text: fileName,
		Style: widget.RichTextStyle{
			TextStyle: fyne.TextStyle{Bold: true},
		},
	})
	infoLabel.Truncation = fyne.TextTruncateEllipsis
	infoLabel.Wrapping = fyne.TextWrapWord

	// Нижняя панель
	bottomPanel := container.NewVBox(
		widget.NewSeparator(),
		infoLabel,
	)

	// Основной контейнер
	mainContent := container.NewBorder(
		nil,
		bottomPanel,
		nil,
		nil,
		previewContainer,
	)

	// Оборачиваем в кликабельный виджет
	vc.Container = hover_preview.NewClickableCardWithDoubleTap(
		mainContent,
		func() {
			// Одинарный клик - показываем меню
			menuManager := hover_preview.NewMenuManager()
			menuManager.ShowSimpleMenu(vc.Item, vc.Container, nil)
		},
		func() {
			// Двойной клик - открываем файл
			vc.openCurrentFileWithDefaultApp()
		},
	)
}

// createVideoPreview создает превью для видео (заглушку)
func (vc *VideoCard) createVideoPreview(filePath string) fyne.CanvasObject {
	// Создаем градиентный фон в стиле видео
	gradient := canvas.NewHorizontalGradient(
		color.RGBA{R: 30, G: 30, B: 50, A: 255},
		color.RGBA{R: 60, G: 40, B: 80, A: 255},
	)
	gradient.SetMinSize(fyne.NewSize(0, 180))

	// Иконка видео по центру
	videoIcon := widget.NewIcon(theme.MediaVideoIcon())
	videoContainer := container.NewCenter(videoIcon)

	// Контейнер для превью с минимальной высотой
	previewContent := container.NewStack(gradient, videoContainer)

	return previewContent
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
	if err := cmd.Run(); err != nil {
		fmt.Printf("[ERROR] Ошибка при открытии файла: %v\n", err)
	}
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
	// Если контент уже инициализирован, просто обновляем контейнер
	if vc.isContentInitialized {
		vc.Container.Refresh()
		return
	}

	// Первый вызов - пересоздаем карточку с обновленным элементом
	newCard := NewVideoCardWithCallback(vc.Item, nil)
	vc.Container = newCard.GetContainer()
}
