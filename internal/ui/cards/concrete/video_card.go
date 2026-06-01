package concrete

import (
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
	isContentInitialized bool
	noButtonsMode        bool
}

// NewVideoCard создает новую карточку для видео
// Опциональный параметр noButtons управляет режимом отображения меню без кнопок
func NewVideoCard(item *models.Item, noButtons ...bool) interfaces.CardRenderer {
	noButtonsMode := len(noButtons) > 0 && noButtons[0]
	return NewVideoCardWithCallback(item, nil, noButtonsMode)
}

// NewVideoCardWithCallback создает новую карточку для видео с пользовательским обработчиком клика
// Опциональный параметр noButtons управляет режимом отображения меню без кнопок
func NewVideoCardWithCallback(item *models.Item, clickCallback func(), noButtons ...bool) interfaces.CardRenderer {
	noButtonsMode := len(noButtons) > 0 && noButtons[0]

	videoCard := &VideoCard{
		BaseCard:       cards.NewBaseCard(item),
		noButtonsMode:  noButtonsMode,
		videoFiles:     make([]*cards.Block, 0),
		currentFileIdx: 0,
	}

	// Извлекаем все видеофайлы из ContentMeta
	blocks, _ := cards.ParseBlocks(videoCard.Item.ContentMeta)

	// Собираем видеофайлы
	for _, block := range blocks {
		if block.Type == "video" && block.FileHash != "" {
			videoCard.videoFiles = append(videoCard.videoFiles, &block)
		}
	}

	// Если нет видеофайлов, показываем заглушку
	if len(videoCard.videoFiles) == 0 {
		placeholder := widget.NewLabel("No video files found")
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

	// Создаем превью (заглушка с градиентом)
	previewContent := vc.createVideoPlaceholder()

	// Создаем оверлей с кнопкой Play по центру
	playBtn := widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
		vc.openCurrentFileWithDefaultApp()
	})
	playBtn.Importance = widget.HighImportance

	// Полупрозрачный фон для кнопки
	playBg := canvas.NewCircle(color.RGBA{R: 0, G: 0, B: 0, A: 180})
	playOverlayContent := container.NewStack(playBg, playBtn)
	playOverlay := container.NewCenter(playOverlayContent)

	// Контейнер с превью и оверлеем
	previewContainer := container.NewStack(previewContent, playOverlay)

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
			menuManager.ShowSimpleMenu(vc.Item, vc.Container, nil, vc.noButtonsMode)
		},
		func() {
			// Двойной клик - открываем файл
			vc.openCurrentFileWithDefaultApp()
		},
	)
}

// createVideoPlaceholder создает заглушку для видео с градиентом и иконкой
func (vc *VideoCard) createVideoPlaceholder() fyne.CanvasObject {
	gradient := canvas.NewHorizontalGradient(
		color.RGBA{R: 30, G: 30, B: 50, A: 255},
		color.RGBA{R: 60, G: 40, B: 80, A: 255},
	)
	gradient.SetMinSize(fyne.NewSize(0, 180))

	videoIcon := widget.NewIcon(theme.MediaVideoIcon())
	videoContainer := container.NewCenter(videoIcon)

	return container.NewStack(gradient, videoContainer)
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
	_ = cmd.Run()
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
	newCard := NewVideoCardWithCallback(vc.Item, nil, vc.noButtonsMode)
	vc.Container = newCard.GetContainer()
}
