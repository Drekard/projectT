package create_item

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// FileUploadState содержит состояние для загрузки файлов
type FileUploadState struct {
	SelectedFiles *[]string
	UpdateDisplay func()
}

// openWindowsFileDialog открывает стандартный диалог выбора файлов Windows
func OpenWindowsFileDialog(filter []string, multiSelect bool) ([]string, error) {
	// Создаем PowerShell скрипт для открытия диалога выбора файлов
	psScript := `
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Add-Type -AssemblyName System.Windows.Forms
$dialog = New-Object System.Windows.Forms.OpenFileDialog
$dialog.Title = "Выберите файлы"
$dialog.Multiselect = $true
`

	// Добавляем фильтр, если он задан
	if len(filter) > 0 {
		// Создаем строку фiterа в формате: "Image files (*.jpg, *.png)|*.jpg;*.png"
		filterExtensions := []string{}

		for _, ext := range filter {
			cleanExt := strings.TrimPrefix(ext, ".")
			filterExtensions = append(filterExtensions, "*."+cleanExt)
		}

		filterStr := strings.Join(filterExtensions, ";")
		displayName := "Files"

		// Определяем отображаемое имя в зависимости от типа фильтра
		if len(filter) == 1 && (strings.Contains(filter[0], "jpg") ||
			strings.Contains(filter[0], "jpeg") ||
			strings.Contains(filter[0], "png") ||
			strings.Contains(filter[0], "gif") ||
			strings.Contains(filter[0], "bmp")) {
			displayName = "Image files"
		} else if len(filter) == 1 && (strings.Contains(filter[0], "pdf") ||
			strings.Contains(filter[0], "doc") ||
			strings.Contains(filter[0], "txt")) {
			displayName = "Document files"
		}

		psScript += fmt.Sprintf(`$dialog.Filter = "%s (%s)|%s"`,
			displayName,
			strings.Join(filterExtensions, ", "),
			filterStr)

		// Добавляем опцию "Все файлы" для удобства
		psScript += fmt.Sprintf(`
$dialog.FilterIndex = 1
$dialog.DefaultExt = "%s"`, filterExtensions[0])
	}

	psScript += `
$result = $dialog.ShowDialog()
if ($result -eq [System.Windows.Forms.DialogResult]::OK) {
    $dialog.FileNames | ForEach-Object {
        Write-Output $_
    }
} else {
    Write-Output ""
}
`

	// Выполняем PowerShell скрипт с явным указанием кодировки
	cmd := exec.Command("powershell", "-Command", psScript)

	// Устанавливаем кодировку для ввода/вывода
	cmd.Env = append(os.Environ(),
		"PYTHONIOENCODING=utf-8",
		"LANG=en_US.UTF-8",
	)

	// Используем правильную кодировку для вывода
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Проверяем, может быть это просто отмена выбора
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 0 {
			return []string{}, nil
		}
		return nil, fmt.Errorf("ошибка открытия диалога: %v\nstderr: %s", err, stderr.String())
	}

	// Преобразуем вывод с учетом кодировки
	outputBytes := stdout.Bytes()

	// Пробуем декодировать в UTF-8, если не получается - пробуем windows-1251
	var outputStr string
	if utf8.Valid(outputBytes) {
		outputStr = string(outputBytes)
	} else {
		// Пробуем другие кодировки
		if dec, err := charmap.Windows1251.NewDecoder().Bytes(outputBytes); err == nil {
			outputStr = string(dec)
		} else {
			// Последняя попытка - преобразовать как есть
			outputStr = string(outputBytes)
		}
	}

	outputStr = strings.TrimSpace(outputStr)
	if outputStr == "" {
		return []string{}, nil
	}

	// Разделяем строки
	lines := strings.ReplaceAll(outputStr, "\r\n", "\n")
	files := strings.Split(lines, "\n")

	// Очищаем пробелы
	var cleanFiles []string
	for _, file := range files {
		cleanFile := strings.TrimSpace(file)
		if cleanFile != "" && cleanFile != "\"" {
			// Проверяем существование файла
			if _, err := os.Stat(cleanFile); err == nil {
				cleanFiles = append(cleanFiles, cleanFile)
			} else {
				// Если файл не найден, продолжаем
			}
		}
	}

	return cleanFiles, nil
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
	fileSelectorButton := widget.NewButton("Выбрать файл/картинку/видео", nil) // Изначально без обработчика

	// Контейнер для отображения выбранного файла с кнопкой удаления
	fileDisplayContainer := container.NewVBox()

	// Назначаем обработчик событий для кнопки выбора файла
	fileSelectorButton.OnTapped = func() {
		// Получаем текущий Canvas
		canvas := fyne.CurrentApp().Driver().CanvasForObject(fileSelectorButton)

		// Открываем стандартный диалог Windows
		selectedFiles, err := OpenWindowsFileDialog(nil, true) // без фильтра - любые файлы
		if err != nil {
			// В случае ошибки показываем сообщение
			errorLabel := widget.NewLabel(fmt.Sprintf("Ошибка при выборе файлов:\n%v", err))
			errorLabel.Wrapping = fyne.TextWrapWord

			closeButton := widget.NewButton("Закрыть", nil)

			popupContent := container.NewVBox(
				errorLabel,
				container.NewCenter(closeButton),
			)

			dialog := widget.NewModalPopUp(popupContent, canvas)

			closeButton.OnTapped = func() {
				dialog.Hide()
			}

			dialog.Show()
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
