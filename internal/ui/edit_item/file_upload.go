package edit_item

import (
	"bytes"
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/text/encoding/charmap"
)

// FileUploadType определяет тип загрузки файла
type FileUploadType int

const (
	ImageUpload FileUploadType = iota
	FileUpload
)

// FileUploadConfig содержит конфигурацию для загрузки файлов
type FileUploadConfig struct {
	Label           string
	Filter          []string
	BackgroundColor color.Color
	MinSize         fyne.Size
	UploadType      FileUploadType
}

// FileUploadState содержит состояние для загрузки файлов
type FileUploadState struct {
	SelectedFiles *[]string
	UpdateDisplay func()
}

// openWindowsFileDialog открывает стандартный диалог выбора файлов Windows
func openWindowsFileDialog(filter []string, multiSelect bool) ([]string, error) {
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
		// Создаем строку фильтра в формате: "Image files (*.jpg, *.png)|*.jpg;*.png"
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

// CreateFileUploadArea создает область для загрузки файлов
func CreateFileUploadArea(config FileUploadConfig, state *FileUploadState, parentWindow fyne.Window) *fyne.Container {
	// Инициализируем SelectedFiles если nil
	if state.SelectedFiles == nil {
		emptySlice := []string{}
		state.SelectedFiles = &emptySlice
	}

	// Инициализируем UpdateDisplay если nil (ВАЖНО!)
	if state.UpdateDisplay == nil {
		state.UpdateDisplay = func() {}
	}

	// Прямоугольник с фоном
	box := canvas.NewRectangle(config.BackgroundColor)
	box.CornerRadius = 8
	box.SetMinSize(config.MinSize)

	// Контейнер для отображения файлов
	filesContainer := container.NewVBox()

	// Метка
	label := widget.NewLabel(config.Label)
	label.Alignment = fyne.TextAlignCenter

	// Контейнер для содержимого внутри box
	contentContainer := container.NewVBox(filesContainer, container.NewCenter(label))

	containerWithContent := container.NewStack(box, contentContainer)

	// Кнопка для клика (скрывается под содержимым)
	clickButton := widget.NewButton("", func() {
		// Открываем стандартный диалог Windows
		selectedFiles, err := openWindowsFileDialog(config.Filter, true)
		if err != nil {
			// В случае ошибки показываем сообщение
			errorLabel := widget.NewLabel(fmt.Sprintf("Ошибка при выборе файлов:\n%v", err))
			errorLabel.Wrapping = fyne.TextWrapWord

			closeButton := widget.NewButton("Закрыть", nil)

			popupContent := container.NewVBox(
				errorLabel,
				container.NewCenter(closeButton),
			)

			dialog := widget.NewModalPopUp(popupContent, parentWindow.Canvas())

			closeButton.OnTapped = func() {
				dialog.Hide()
			}

			dialog.Show()
			return
		}

		if len(selectedFiles) > 0 {
			// Добавляем выбранные файлы в список
			*state.SelectedFiles = append(*state.SelectedFiles, selectedFiles...)
			state.UpdateDisplay()
		}
	})
	clickButton.Importance = widget.LowImportance

	containerWithClick := container.NewStack(clickButton, containerWithContent)

	// Переопределяем функцию обновления отображения
	state.UpdateDisplay = func() {
		filesContainer.Objects = []fyne.CanvasObject{}

		// Разыменовываем указатель
		selectedFiles := *state.SelectedFiles

		// Добавляем выбранные файлы
		for i, filepath := range selectedFiles {
			// Извлекаем только имя файла из полного пути
			filename := filepath
			if lastSlash := strings.LastIndex(filepath, "\\"); lastSlash != -1 {
				filename = filepath[lastSlash+1:]
			} else if lastSlash := strings.LastIndex(filepath, "/"); lastSlash != -1 {
				filename = filepath[lastSlash+1:]
			}

			// Создаем контейнер для каждого файла с именем и кнопкой удаления
			fileLabel := widget.NewLabel(filename)

			removeButton := widget.NewButton("❌", func(index int) func() {
				return func() {
					// Удаляем файл из списка
					currentFiles := *state.SelectedFiles
					newSelectedFiles := make([]string, 0, len(currentFiles)-1)
					for j, name := range currentFiles {
						if j != index {
							newSelectedFiles = append(newSelectedFiles, name)
						}
					}
					*state.SelectedFiles = newSelectedFiles
					state.UpdateDisplay()
				}
			}(i))
			removeButton.Importance = widget.LowImportance

			hBox := container.NewHBox(fileLabel, container.NewPadded(removeButton))

			// Добавляем горизонтальный разделитель после каждого элемента, кроме последнего
			filesContainer.Add(hBox)
			if i < len(selectedFiles)-1 {
				filesContainer.Add(widget.NewSeparator())
			}
		}

		// Обновляем содержимое контейнера
		contentContainer.Objects = []fyne.CanvasObject{filesContainer, container.NewCenter(label)}
		contentContainer.Refresh()
	}

	// Вызываем первоначальное отображение
	state.UpdateDisplay()

	return containerWithClick
}
