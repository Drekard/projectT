package concrete

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/filesystem"
	"projectT/internal/ui/cards"
	"projectT/internal/ui/cards/hover_preview"
	"projectT/internal/ui/cards/interfaces"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ImageCard карточка для изображений
type ImageCard struct {
	*cards.BaseCard
	imageWidgets          []*canvas.Image
	currentIndex          int
	imageWidget           *canvas.Image
	titleLabel            *widget.RichText
	imageCounter          *widget.Label
	totalImages           int
	mainContainer         *fyne.Container // Добавляем ссылку на основной контейнер
	fixedWidth            float32         // Фиксированная ширина контейнера
	fixedHeight           float32         // Фиксированная высота контейнера
	isFixedSizeCalculated bool            // Флаг, указывающий, что фиксированные размеры уже вычислены
}

// NewImageCard создает новую карточку для изображения
func NewImageCard(item *models.Item) interfaces.CardRenderer {
	return NewImageCardWithCallback(item, nil)
}

// NewImageCardWithCallback создает новую карточку для изображения с пользовательским обработчиком клика
func NewImageCardWithCallback(item *models.Item, clickCallback func()) interfaces.CardRenderer {
	imageCard := &ImageCard{
		BaseCard: cards.NewBaseCard(item),
	}

	// Извлекаем все изображения из content_meta
	imageCard.imageWidgets = imageCard.extractAllImagesFromContentMeta(item.ContentMeta)
	imageCard.totalImages = len(imageCard.imageWidgets)

	if imageCard.totalImages > 0 {
		imageCard.currentIndex = 0
		imageCard.imageWidget = imageCard.imageWidgets[imageCard.currentIndex]

		// Настраиваем изображение - используем FillMode для лучшего контроля размеров
		imageCard.imageWidget.FillMode = canvas.ImageFillContain
		imageCard.imageWidget.ScaleMode = canvas.ImageScaleSmooth

		// Создаем счетчик изображений
		if imageCard.totalImages > 1 {
			imageCard.imageCounter = widget.NewLabelWithStyle(
				imageCard.getImageCounterText(),
				fyne.TextAlignTrailing,
				fyne.TextStyle{Bold: false},
			)
		}

		// Если есть несколько изображений, создаем навигационную панель
		var bottomBar fyne.CanvasObject
		if imageCard.totalImages > 1 {
			prevBtn := widget.NewButton("<", func() {
				imageCard.showPreviousImage()
			})
			prevBtn.Importance = widget.LowImportance

			nextBtn := widget.NewButton(">", func() {
				imageCard.showNextImage()
			})
			nextBtn.Importance = widget.LowImportance

			navigationContainer := container.NewHBox(
				prevBtn,
				container.NewCenter(container.NewVBox()), // Пустой контейнер вместо заголовка
				nextBtn,
				imageCard.imageCounter,
			)

			bottomBar = navigationContainer
		} else {
			bottomBar = container.NewCenter(container.NewVBox()) // Пустой контейнер вместо заголовка
		}

		// Вычисляем фиксированные размеры на основе первого изображения
		imageCard.fixedWidth = 250  // фиксируем ширину
		imageCard.fixedHeight = 250 // по умолчанию используем 250, если не можем вычислить

		// Получаем размеры первого изображения
		if fileContent, _, err := filesystem.ReadFileByHash(imageCard.getCurrentImageHash()); err == nil && fileContent != nil {
			if height, err := calculateImageHeight(fileContent, 250); err == nil {
				imageCard.fixedHeight = height
			}
		}

		imageCard.isFixedSizeCalculated = true

		// Создаем прямоугольник с фиксированным размером
		rect := canvas.NewRectangle(nil)
		rect.SetMinSize(fyne.NewSize(imageCard.fixedWidth, imageCard.fixedHeight))

		// Контейнер для изображения с ограничением минимального размера
		imageContainer := container.NewStack(rect, imageCard.imageWidget)

		// Ограничиваем размер изображения, чтобы оно соответствовало фиксированному размеру
		imageCard.imageWidget.SetMinSize(fyne.NewSize(imageCard.fixedWidth, imageCard.fixedHeight))

		// Основной контейнер с изображением и панелью
		imageCard.mainContainer = container.NewBorder(
			nil,
			bottomBar,
			nil,
			nil,
			imageContainer,
		)

		// Оборачиваем в кликабельный виджет с поддержкой двойного клика
		clickableCard := hover_preview.NewClickableCardWithDoubleTap(imageCard.mainContainer, func() {
			// Обработчик одинарного клика - показываем меню действий
			menuManager := hover_preview.NewMenuManager()
			menuManager.ShowSimpleMenu(imageCard.Item, imageCard.mainContainer, nil)
		}, func() {
			// Обработчик двойного клика - открываем изображение в проводнике
			imageCard.openImageWithDefaultWindowsApp()
		})

		// Контейнер без фона, рамки и отступов, так как будет использоваться внутри другой карточки
		imageCard.Container = clickableCard
	} else {
		// Если изображение не найдено
		placeholder := widget.NewLabel("Изображение не найдено")
		placeholder.Alignment = fyne.TextAlignCenter

		// Контейнер без фона, рамки и отступов, так как будет использоваться внутри другой карточки
		imageCard.Container = container.NewCenter(placeholder)
	}

	return imageCard
}

// Методы, необходимые для реализации интерфейса CardRenderer
func (ic *ImageCard) GetContainer() fyne.CanvasObject {
	return ic.Container
}

func (ic *ImageCard) GetWidget() fyne.CanvasObject {
	return ic.Container
}

func (ic *ImageCard) SetContainer(container fyne.CanvasObject) {
	ic.Container = container
}

func (ic *ImageCard) UpdateContent() {
	// Обновляем содержимое карточки
	// Пересоздаем карточку с обновленным элементом
	newCard := NewImageCardWithCallback(ic.Item, nil)

	// Копируем контейнер новой карточки в текущую
	ic.Container = newCard.GetContainer()
}

// extractAllImagesFromContentMeta извлекает все изображения из ContentMeta
func (ic *ImageCard) extractAllImagesFromContentMeta(contentMeta string) []*canvas.Image {
	var images []*canvas.Image
	blocks, err := cards.ParseBlocks(contentMeta)
	if err != nil {
		return images
	}

	// Ищем все изображения в блоках
	for _, block := range blocks {
		if block.Type == "image" && block.FileHash != "" {
			// Загружаем содержимое файла изображения из файловой системы по хешу
			fileContent, _, err := filesystem.ReadFileByHash(block.FileHash)
			if err != nil || fileContent == nil {
				continue // Пропускаем, если не удалось прочитать файл
			}

			// Создаем изображение из байтов
			img := canvas.NewImageFromReader(bytes.NewReader(fileContent), "")
			if img != nil {
				img.FillMode = canvas.ImageFillContain
				images = append(images, img)
			}
		}
	}

	return images
}

// getImageCounterText возвращает текст для счетчика изображений
func (ic *ImageCard) getImageCounterText() string {
	if ic.totalImages > 0 {
		return fmt.Sprintf("%d/%d", ic.currentIndex+1, ic.totalImages)
	}
	return "0/0"
}

// showPreviousImage показывает предыдущее изображение
func (ic *ImageCard) showPreviousImage() {
	if ic.totalImages <= 1 {
		return
	}

	ic.currentIndex--
	if ic.currentIndex < 0 {
		ic.currentIndex = ic.totalImages - 1
	}

	ic.updateImage()
}

// showNextImage показывает следующее изображение
func (ic *ImageCard) showNextImage() {
	if ic.totalImages <= 1 {
		return
	}

	ic.currentIndex++
	if ic.currentIndex >= ic.totalImages {
		ic.currentIndex = 0
	}

	ic.updateImage()
}

// updateImage обновляет отображаемое изображение и счетчик
func (ic *ImageCard) updateImage() {
	if ic.totalImages <= 0 {
		return
	}

	// Обновляем текущее изображение
	ic.imageWidget = ic.imageWidgets[ic.currentIndex]
	ic.imageWidget.FillMode = canvas.ImageFillContain
	ic.imageWidget.ScaleMode = canvas.ImageScaleSmooth

	// Устанавливаем фиксированный минимальный размер для текущего изображения
	// Используем размеры первого изображения, чтобы избежать изменения размера контейнера при листании
	ic.imageWidget.SetMinSize(fyne.NewSize(ic.fixedWidth, ic.fixedHeight))

	// Обновляем счетчик
	if ic.imageCounter != nil {
		ic.imageCounter.SetText(ic.getImageCounterText())
	}

	// Обновляем контейнер с изображением
	// Находим контейнер изображения в основном контейнере
	if ic.mainContainer != nil && len(ic.mainContainer.Objects) > 0 {
		// mainContainer имеет Border layout, изображение - это первый объект
		if borderContainer, ok := ic.mainContainer.Objects[0].(*fyne.Container); ok {
			// Очищаем и добавляем новое изображение
			borderContainer.Objects = []fyne.CanvasObject{ic.imageWidget}
			borderContainer.Refresh()
		}
	}

	// Обновляем весь контейнер
	if ic.Container != nil {
		ic.Container.Refresh()
	}
}

// extractImageFromContentMeta извлекает первое изображение из ContentMeta
func (ic *ImageCard) extractImageFromContentMeta(contentMeta string) *canvas.Image {
	blocks, err := cards.ParseBlocks(contentMeta)
	if err != nil {
		return nil
	}

	// Ищем первое изображение в блоках
	for _, block := range blocks {
		if block.Type == "image" && block.FileHash != "" {
			// Загружаем содержимое файла изображения из файловой системы по хешу
			fileContent, _, err := filesystem.ReadFileByHash(block.FileHash)
			if err != nil || fileContent == nil {
				continue // Пропускаем, если не удалось прочитать файл
			}

			// Создаем изображение из байтов
			img := canvas.NewImageFromReader(bytes.NewReader(fileContent), "")
			if img != nil {
				img.FillMode = canvas.ImageFillContain
				return img
			}
		}
	}

	return nil
}

// openImageWithDefaultWindowsApp открывает изображение средствами Windows
func (c *ImageCard) openImageWithDefaultWindowsApp() {

	if c.totalImages == 0 {
		return
	}

	// Получаем блоки из ContentMeta
	blocks, err := cards.ParseBlocks(c.Item.ContentMeta)
	if err != nil {
		return
	}

	// Ищем блок с текущим изображением
	var targetBlock *cards.Block
	imageIndex := 0
	for i := range blocks {
		fmt.Printf("[DEBUG] Блок %d: type=%s, file_hash=%s\n",
			i, blocks[i].Type, blocks[i].FileHash)
		if blocks[i].Type == "image" && blocks[i].FileHash != "" {
			fmt.Printf("[DEBUG] Найден image блок #%d, ищем #%d\n",
				imageIndex, c.currentIndex)
			if imageIndex == c.currentIndex {
				targetBlock = &blocks[i]
				break
			}
			imageIndex++
		}
	}

	if targetBlock == nil {
		return
	}

	if targetBlock.FileHash == "" {
		return
	}

	// Получаем путь к файлу изображения по хешу
	imagePath := filesystem.GetFilePathByHash(targetBlock.FileHash)
	if imagePath == "" {
		// Попробуем найти файл вручную
		imagePath = c.findFileManually(targetBlock.FileHash)
		if imagePath == "" {
			return
		}
	}

	// Проверяем существует ли файл
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return
	}

	// Получаем абсолютный путь
	absPath, err := filepath.Abs(imagePath)
	if err != nil {
		absPath = imagePath
	}

	// Пытаемся открыть через rundll32
	cmd3 := exec.Command("explorer.exe", absPath)
	if err := cmd3.Run(); err == nil {
		return
	} else {
	}
}

// calculateImageHeight вычисляет высоту изображения на основе его оригинальных размеров и заданной ширины
func calculateImageHeight(imageBytes []byte, width float32) (float32, error) {
	// Декодируем изображение для получения его размеров
	imgConfig, _, err := image.DecodeConfig(bytes.NewReader(imageBytes))
	if err != nil {
		return 0, err
	}

	// Вычисляем соотношение сторон
	ratio := float32(imgConfig.Height) / float32(imgConfig.Width)

	// Вычисляем новую высоту на основе заданной ширины
	newHeight := ratio * width

	return newHeight, nil
}

// getCurrentImageHash возвращает хеш текущего изображения
func (ic *ImageCard) getCurrentImageHash() string {
	if ic.totalImages == 0 || ic.currentIndex < 0 || ic.currentIndex >= len(ic.imageWidgets) {
		return ""
	}

	// Извлекаем хеш из ContentMeta для текущего изображения
	blocks, err := cards.ParseBlocks(ic.Item.ContentMeta)
	if err != nil {
		return ""
	}

	// Ищем блок с текущим изображением
	imageIndex := 0
	for _, block := range blocks {
		if block.Type == "image" && block.FileHash != "" {
			if imageIndex == ic.currentIndex {
				return block.FileHash
			}
			imageIndex++
		}
	}

	return ""
}

// Вспомогательный метод для поиска файла
func (c *ImageCard) findFileManually(fileHash string) string {
	// Попробуем несколько возможных путей
	possiblePaths := []string{
		filepath.Join("storage", "files", fileHash+".jpg"),
		filepath.Join("storage", "files", fileHash+".png"),
		filepath.Join("storage", "files", fileHash+".jpeg"),
		filepath.Join("storage", "files", fileHash), // без расширения
		filepath.Join(".", "files", fileHash+".jpg"),
		// Добавьте другие возможные пути
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}
