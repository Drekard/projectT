package create_item

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// FileUploadState содержит состояние для загрузки файлов
type FileUploadState struct {
	SelectedFiles *[]string
	UpdateDisplay func()
}

// openFileDialog открывает диалог выбора файлов (кроссплатформенная версия)
func openFileDialog(parentWindow fyne.Window, filter []string, multiSelect bool) ([]string, error) {
	var selectedFiles []string

	fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			return
		}
		if reader != nil {
			selectedFiles = append(selectedFiles, reader.URI().Path())
			_ = reader.Close()
		}
	}, parentWindow)

	// Устанавливаем фильтр если указан
	if len(filter) > 0 {
		extensions := make([]string, len(filter))
		for i, ext := range filter {
			extensions[i] = strings.TrimPrefix(ext, ".")
		}
		fd.SetFilter(storage.NewExtensionFileFilter(extensions))
	}

	fd.Show()

	// Ждём завершения диалога (простая реализация)
	// В идеале нужно использовать каналы или callback'и
	return selectedFiles, nil
}

// IsImageFile проверяет, является ли файл изображением по его расширению
func IsImageFile(filename string) bool {
	imageExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".tiff", ".svg"}

	lowerFilename := strings.ToLower(filename)
	for _, ext := range imageExtensions {
		if strings.HasSuffix(lowerFilename, ext) {
			return true
		}
	}

	return false
}

// CreateFileSelector создает элемент управления для выбора файлов
func CreateFileSelector(fileState *FileUploadState) fyne.CanvasObject {
	// Кнопка для выбора файла
	fileSelectorButton := widget.NewButton("Select file/image/video", nil)

	// Контейнер для отображения выбранного файла с кнопкой удаления
	fileDisplayContainer := container.NewVBox()

	// Назначаем обработчик событий для кнопки выбора файла
	fileSelectorButton.OnTapped = func() {
		// Получаем окно для диалога
		windows := fyne.CurrentApp().Driver().AllWindows()
		if len(windows) == 0 {
			return
		}
		parentWindow := windows[0]

		// Открываем кроссплатформенный диалог
		selectedFiles, err := openFileDialog(parentWindow, nil, true)
		if err != nil {
			// В случае ошибки показываем сообщение
			canvas := fyne.CurrentApp().Driver().CanvasForObject(fileSelectorButton)
			errorLabel := widget.NewLabel(fmt.Sprintf("Error selecting files:\n%v", err))
			errorLabel.Wrapping = fyne.TextWrapWord

			closeButton := widget.NewButton("Close", nil)

			popupContent := container.NewVBox(
				errorLabel,
				container.NewCenter(closeButton),
			)

			popup := widget.NewModalPopUp(popupContent, canvas)

			closeButton.OnTapped = func() {
				popup.Hide()
			}

			popup.Show()
			return
		}

		if len(selectedFiles) > 0 {
			// Добавляем выбранные файлы к уже существующим
			*fileState.SelectedFiles = append(*fileState.SelectedFiles, selectedFiles...)
			fileState.UpdateDisplay()
		}
	}
	fileSelectorButton.Importance = widget.LowImportance

	// Переопределяем функцию обновления отображения
	fileState.UpdateDisplay = func() {
		// Очищаем контейнер отображения файла
		fileDisplayContainer.Objects = nil

		// Если файлы выбраны, показываем их все с кнопками удаления
		selectedFiles := *fileState.SelectedFiles

		for i, filepath := range selectedFiles {
			// Извлекаем только имя файла из полного пути
			filename := filepath
			if lastSlash := strings.LastIndex(filepath, "\\"); lastSlash != -1 {
				filename = filepath[lastSlash+1:]
			} else if lastSlash := strings.LastIndex(filepath, "/"); lastSlash != -1 {
				filename = filepath[lastSlash+1:]
			}

			// Определяем тип файла и устанавливаем соответствующий эмодзи
			var emoji string
			if IsImageFile(filename) {
				emoji = "🖼️ "
			} else {
				emoji = "📎 "
			}

			// Создаем метку с именем файла
			fileLabel := widget.NewLabel(emoji + filename)

			// Кнопка удаления файла
			removeButton := widget.NewButton("❌", func(index int) func() {
				return func() {
					// Удаляем файл по индексу из списка
					currentFiles := *fileState.SelectedFiles
					newSelectedFiles := make([]string, 0, len(currentFiles)-1)
					for j, file := range currentFiles {
						if j != index {
							newSelectedFiles = append(newSelectedFiles, file)
						}
					}
					*fileState.SelectedFiles = newSelectedFiles
					fileState.UpdateDisplay() // Обновляем отображение
				}
			}(i))
			removeButton.Importance = widget.LowImportance

			// Добавляем метку и кнопку удаления в контейнер
			fileDisplayContainer.Add(container.NewHBox(fileLabel, removeButton))

			// Добавляем разделитель между файлами
			if i < len(selectedFiles)-1 {
				fileDisplayContainer.Add(widget.NewSeparator())
			}
		}
	}

	// Объединяем кнопку выбора файла и контейнер отображения в один контейнер
	fileSelectorContainer := container.NewVBox(fileDisplayContainer, fileSelectorButton)

	return fileSelectorContainer
}
