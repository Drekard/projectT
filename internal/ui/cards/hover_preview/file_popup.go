package hover_preview

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"projectT/internal/storage/filesystem"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// openFileWithDefaultApp opens a file using the default application in Windows
func openFileWithDefaultApp(filePath string) {
	fmt.Printf("[DEBUG] Opening file: %s\n", filePath)

	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		fmt.Printf("[WARN] Error getting absolute path: %v\n", err)
		absPath = filePath
	}

	// Open file through explorer.exe
	fmt.Printf("[DEBUG] Opening via explorer.exe: %s\n", absPath)
	cmd := exec.Command("explorer.exe", absPath)
	if err := cmd.Run(); err != nil {
		fmt.Printf("[ERROR] Error opening file: %v\n", err)

		// Try alternative method through cmd start
		cmd = exec.Command("cmd", "/c", "start", "", absPath)
		if err := cmd.Run(); err != nil {
			fmt.Printf("[ERROR] Alternative method also failed: %v\n", err)
		}
	} else {
		fmt.Printf("[DEBUG] File successfully opened\n")
	}
}

// FileItem represents a file with its display name and hash
type FileItem struct {
	DisplayName string // Name to display on the button
	Hash        string // Hash to use for file lookup
}

// FilePopup компонент для отображения всплывающего окна со списком файлов
type FilePopup struct {
	popup *widget.PopUp
}

// NewFilePopup создает новый компонент для отображения списка файлов
func NewFilePopup(fileItems []FileItem, trigger fyne.CanvasObject) *FilePopup {
	fp := &FilePopup{}

	// Create list of files
	var fileObjects []fyne.CanvasObject
	for _, fileItem := range fileItems {
		// Create button for each file
		fileButton := widget.NewButton(fileItem.DisplayName, func() {
			// Use the hash from the FileItem to get file path
			currentFileItem := fileItem // Capture loop variable
			if currentFileItem.Hash != "" && filesystem.IsValidHash(currentFileItem.Hash) {
				// Get file path by hash
				filePath := filesystem.GetFilePathByHash(currentFileItem.Hash)
				if filePath != "" {
					openFileWithDefaultApp(filePath)
				} else {
					fmt.Printf("[ERROR] Could not get file path for hash: %s\n", currentFileItem.Hash)
				}
			} else {
				fmt.Printf("[ERROR] Invalid hash format detected: %s\n", currentFileItem.Hash)
			}
		})

		// Fyne buttons center text by default, and there's no direct way to change this
		// One workaround is to use a custom widget that mimics a button with left-aligned text

		// For now, we'll use the standard button, but if left alignment is critical,
		// we could implement a custom widget with CanvasObjects and event handling

		fileObjects = append(fileObjects, fileButton)
	}

	// Создаем контент для всплывающего окна
	var content fyne.CanvasObject
	if len(fileItems) <= 5 {
		// Если файлов 5 или меньше, показываем без прокрутки
		content = container.NewVBox(
			widget.NewLabel("Все файлы:"),
			container.NewVBox(fileObjects...),
		)
	} else {
		// Если больше 5 файлов, добавляем прокрутку
		content = container.NewVBox(
			widget.NewLabel("Все файлы:"),
			container.NewVScroll(container.NewVBox(fileObjects...)),
		)
	}

	// Создаем всплывающее окно
	canvas := fyne.CurrentApp().Driver().CanvasForObject(trigger)
	fp.popup = widget.NewPopUp(content, canvas)

	// Устанавливаем размер окна в зависимости от количества файлов
	maxHeight := float32(400)   // Максимальная высота окна
	itemHeight := float32(40)   // Приблизительная высота одной кнопки файла
	headerHeight := float32(80) // Высота заголовка и других элементов

	calculatedHeight := headerHeight + float32(len(fileItems))*itemHeight
	if calculatedHeight > maxHeight {
		// Если расчетная высота больше максимальной, используем максимальную и добавляем прокрутку
		fp.popup.Resize(fyne.NewSize(400, maxHeight))
	} else {
		// Иначе используем расчетную высоту
		fp.popup.Resize(fyne.NewSize(400, calculatedHeight))
	}

	return fp
}

// Show показывает всплывающее окно с файлами
func (fp *FilePopup) Show(pos fyne.Position) {
	fp.popup.ShowAtPosition(pos)
}

// Hide скрывает всплывающее окно
func (fp *FilePopup) Hide() {
	fp.popup.Hide()
}

// IsVisible возвращает видимость всплывающего окна
func (fp *FilePopup) IsVisible() bool {
	return fp.popup.Visible()
}
