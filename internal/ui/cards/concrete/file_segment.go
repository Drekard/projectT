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
	"projectT/internal/ui/cards/interfaces"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// FileCard карточка для файлов
type FileCard struct {
	*cards.BaseCard
	nameLabel     *widget.RichText
	extraFilesBtn *widget.Button
	extraFiles    []string
}

// TapGestureDetector структура для обработки жестов клика
type TapGestureDetector struct {
	widget.BaseWidget
	onTapped func()
}

// Убедимся, что TapGestureDetector реализует интерфейс fyne.DoubleTappable
var _ fyne.DoubleTappable = (*TapGestureDetector)(nil)

func (t *TapGestureDetector) CreateRenderer() fyne.WidgetRenderer {
	// Создаем прозрачный прямоугольник, который будет занимать все пространство и обрабатывать клики
	rect := canvas.NewRectangle(color.RGBA{0, 0, 0, 0}) // полностью прозрачный
	rect.Resize(fyne.NewSize(100, 40))                  // начальные размеры
	return &TapGestureDetectorRenderer{
		obj:  t,
		rect: rect,
	}
}

func (t *TapGestureDetector) DoubleTapped(_ *fyne.PointEvent) {
	if t.onTapped != nil {
		t.onTapped()
	}
}

// TapGestureDetectorRenderer рендерер для TapGestureDetector
type TapGestureDetectorRenderer struct {
	obj     *TapGestureDetector
	rect    *canvas.Rectangle
	objects []fyne.CanvasObject
}

func (r *TapGestureDetectorRenderer) Layout(size fyne.Size) {
	r.rect.Resize(size)
}

func (r *TapGestureDetectorRenderer) MinSize() fyne.Size {
	return fyne.NewSize(10, 10)
}

func (r *TapGestureDetectorRenderer) Refresh() {
	r.rect.FillColor = color.RGBA{0, 0, 0, 0} // убедиться, что прозрачный
	canvas.Refresh(r.obj)
}

func (r *TapGestureDetectorRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.rect}
}

func (r *TapGestureDetectorRenderer) Destroy() {
}

// NewFileCard создает новую карточку для файла
func NewFileCard(item *models.Item) interfaces.CardRenderer {
	return NewFileCardWithCallback(item, nil)
}

// NewFileCardWithCallback создает новую карточку для файла с пользовательским обработчиком клика
func NewFileCardWithCallback(item *models.Item, clickCallback func()) interfaces.CardRenderer {
	fileCard := &FileCard{
		BaseCard: cards.NewBaseCard(item),
	}

	// Извлекаем все блоки для определения типа файлов
	blocks, err := cards.ParseBlocks(fileCard.Item.ContentMeta)
	if err != nil {
		fmt.Printf("[ERROR] Ошибка парсинга ContentMeta: %v\n", err)
	}

	// Определяем, какие файлы показывать на карточке (все, кроме изображений)
	var displayFiles []string

	for _, block := range blocks {
		if (block.Type == "file" || block.Type == "image") && block.FileHash != "" {
			// Добавляем файлы, но исключаем изображения из основного отображения
			if block.Type != "image" {
				if block.OriginalName != "" {
					// Извлекаем имя файла из OriginalName
					lastSlash := strings.LastIndex(block.OriginalName, "\\")
					if lastSlash == -1 {
						lastSlash = strings.LastIndex(block.OriginalName, "/")
					}

					if lastSlash != -1 {
						displayFiles = append(displayFiles, block.OriginalName[lastSlash+1:])
					} else {
						displayFiles = append(displayFiles, block.OriginalName)
					}
				} else {
					// Если OriginalName нет, используем хэш как отображаемое имя
					if block.Extension != "" {
						displayName := block.FileHash + "." + block.Extension
						displayFiles = append(displayFiles, displayName)
					} else {
						displayFiles = append(displayFiles, block.FileHash)
					}
				}
			}
		}
	}

	// Создаем контейнер для всех файлов
	var allFileContainers []fyne.CanvasObject

	// Для каждого файла создаем отдельный контейнер с иконкой и обработчиком клика
	for i, fileName := range displayFiles {
		circle := canvas.NewCircle(color.Gray{Y: 40})

		// Иконка файла
		fileIcon := widget.NewIcon(theme.FileIcon())

		blackRect := canvas.NewRectangle(&color.NRGBA{R: 0, G: 0, B: 0, A: 220})
		blackRect.SetMinSize(fyne.NewSize(40, 40))

		// Контейнер для левой части (иконка файла)
		leftContainer := container.NewStack(
			blackRect,
			container.NewStack(circle),
			container.NewCenter(fileIcon),
		)

		// Создаем метку с именем файла
		fileLabel := widget.NewRichText(&widget.TextSegment{
			Text: fileName,
			Style: widget.RichTextStyle{
				TextStyle: fyne.TextStyle{Italic: true},
			},
		})

		// Контейнер для одного файла с обработчиком клика
		contentContainer := container.NewHBox(leftContainer, fileLabel)

		// Обработчик клика для конкретного файла
		idx := i // Сохраняем индекс для замыкания
		tapGestureRecognizer := &TapGestureDetector{}
		tapGestureRecognizer.onTapped = func() {
			fileCard.openSpecificFileWithDefaultApp(idx)
		}

		// Оборачиваем контейнер с файлом в стек с обработчиком клика
		clickableFileContainer := container.NewStack(contentContainer, tapGestureRecognizer)

		allFileContainers = append(allFileContainers, clickableFileContainer)
	}

	// Если есть несколько файлов, оборачиваем в вертикальный контейнер, иначе используем горизонтальный
	var mainContent fyne.CanvasObject
	if len(allFileContainers) > 1 {
		mainContent = container.NewVBox(allFileContainers...)
	} else if len(allFileContainers) == 1 {
		mainContent = allFileContainers[0]
	} else {
		// Если нет файлов для отображения, создаем пустой контейнер
		mainContent = container.NewHBox()
	}

	// Контейнер без фона, рамки и отступов, так как будет использоваться внутри другой карточки
	fileCard.Container = mainContent

	return fileCard
}

// Методы, необходимые для реализации интерфейса CardRenderer
func (fc *FileCard) GetContainer() fyne.CanvasObject {
	return fc.Container
}

func (fc *FileCard) GetWidget() fyne.CanvasObject {
	return fc.Container
}

func (fc *FileCard) SetContainer(container fyne.CanvasObject) {
	fc.Container = container
}

func (fc *FileCard) UpdateContent() {
	// Обновляем содержимое карточки
	// Пересоздаем карточку с обновленным элементом
	newCard := NewFileCardWithCallback(fc.Item, nil)

	// Копируем контейнер новой карточки в текущую
	fc.Container = newCard.GetContainer()
}

// openFirstFileWithDefaultApp открывает первый файл в списке средствами Windows
func (fc *FileCard) openFirstFileWithDefaultApp() {
	fmt.Printf("[DEBUG] Двойной клик по файловой карточке\n")

	// Получаем все блоки файлов
	blocks, err := cards.ParseBlocks(fc.Item.ContentMeta)
	if err != nil {
		fmt.Printf("[ERROR] Ошибка парсинга ContentMeta: %v\n", err)
		return
	}

	if len(blocks) == 0 {
		fmt.Printf("[DEBUG] Нет файлов для открытия\n")
		return
	}

	// Ищем первый блок с файлом
	var targetBlock *cards.Block
	for i := range blocks {
		if (blocks[i].Type == "file" || blocks[i].Type == "image") && blocks[i].FileHash != "" {
			targetBlock = &blocks[i]
			break
		}
	}

	if targetBlock == nil || targetBlock.FileHash == "" {
		fmt.Printf("[DEBUG] Не удалось найти файл для открытия\n")
		return
	}

	fmt.Printf("[DEBUG] Открываем файл: hash=%s, type=%s\n",
		targetBlock.FileHash, targetBlock.Type)

	// Получаем путь к файлу по хешу
	filePath := filesystem.GetFilePathByHash(targetBlock.FileHash)
	if filePath == "" {
		fmt.Printf("[ERROR] Не удалось получить путь к файлу\n")
		return
	}

	fmt.Printf("[DEBUG] Путь к файлу: %s\n", filePath)

	// Получаем абсолютный путь
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		fmt.Printf("[WARN] Ошибка получения абсолютного пути: %v\n", err)
		absPath = filePath
	}

	// Открываем файл через explorer.exe
	fmt.Printf("[DEBUG] Открываем через explorer.exe: %s\n", absPath)
	cmd := exec.Command("explorer.exe", absPath)
	if err := cmd.Run(); err != nil {
		fmt.Printf("[ERROR] Ошибка при открытии файла: %v\n", err)

		// Пробуем альтернативный способ через cmd start
		cmd = exec.Command("cmd", "/c", "start", "", absPath)
		cmd.Run()
	} else {
		fmt.Printf("[DEBUG] Файл успешно открыт\n")
	}
}

// extractAllFilenames извлекает имена всех файлов из JSON-строки ContentMeta
func (fc *FileCard) extractAllFilenames(contentMeta string) []string {
	var filenames []string
	if contentMeta == "" {
		return filenames
	}

	// Используем общую структуру Block из пакета cards
	blocks, err := cards.ParseBlocks(contentMeta)
	if err != nil {
		return filenames
	}

	// Проходим по всем блокам и собираем все файлы с типом "file" или "image"
	for _, block := range blocks {
		if block.Type == "file" || block.Type == "image" {
			// Сначала пробуем использовать OriginalName для отображения
			if block.OriginalName != "" {
				// Извлекаем имя файла из OriginalName
				lastSlash := strings.LastIndex(block.OriginalName, "\\")
				if lastSlash == -1 {
					lastSlash = strings.LastIndex(block.OriginalName, "/")
				}

				if lastSlash != -1 {
					filenames = append(filenames, block.OriginalName[lastSlash+1:])
				} else {
					filenames = append(filenames, block.OriginalName)
				}
			} else if block.FileHash != "" {
				// Если OriginalName недоступен, используем хэш
				if block.Extension != "" {
					filename := block.FileHash + "." + block.Extension
					filenames = append(filenames, filename)
				} else {
					// Если расширение отсутствует, используем только хэш
					filenames = append(filenames, block.FileHash)
				}
			}
		}
	}

	return filenames
}

// openSpecificFileWithDefaultApp открывает конкретный файл по индексу средствами Windows
func (fc *FileCard) openSpecificFileWithDefaultApp(index int) {
	fmt.Printf("[DEBUG] Клик по файлу с индексом %d\n", index)

	// Получаем все блоки файлов
	blocks, err := cards.ParseBlocks(fc.Item.ContentMeta)
	if err != nil {
		fmt.Printf("[ERROR] Ошибка парсинга ContentMeta: %v\n", err)
		return
	}

	// Находим все файлы (кроме изображений)
	var fileBlocks []*cards.Block
	for i := range blocks {
		if (blocks[i].Type == "file" || blocks[i].Type == "image") && blocks[i].FileHash != "" {
			if blocks[i].Type != "image" {
				fileBlocks = append(fileBlocks, &blocks[i])
			}
		}
	}

	// Проверяем, что индекс в пределах диапазона
	if index < 0 || index >= len(fileBlocks) {
		fmt.Printf("[ERROR] Индекс файла вне диапазона: %d\n", index)
		return
	}

	targetBlock := fileBlocks[index]

	fmt.Printf("[DEBUG] Открываем файл: hash=%s, type=%s\n",
		targetBlock.FileHash, targetBlock.Type)

	// Получаем путь к файлу по хешу
	filePath := filesystem.GetFilePathByHash(targetBlock.FileHash)
	if filePath == "" {
		fmt.Printf("[ERROR] Не удалось получить путь к файлу\n")
		return
	}

	fmt.Printf("[DEBUG] Путь к файлу: %s\n", filePath)

	// Получаем абсолютный путь
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		fmt.Printf("[WARN] Ошибка получения абсолютного пути: %v\n", err)
		absPath = filePath
	}

	// Открываем файл через explorer.exe
	fmt.Printf("[DEBUG] Открываем через explorer.exe: %s\n", absPath)
	cmd := exec.Command("explorer.exe", absPath)
	cmd.Run()
}
