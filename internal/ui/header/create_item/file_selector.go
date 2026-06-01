package create_item

import (
	"bytes"
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/text/encoding/charmap"
)

// FileUploadConfig содержит конфигурацию для загрузки файлов
type FileUploadConfig struct {
	Label           string
	Filter          []string
	BackgroundColor color.Color
	MinSize         fyne.Size
}

// FileUploadState содержит состояние для загрузки файлов
type FileUploadState struct {
	SelectedFiles *[]string
	UpdateDisplay func()
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

// openWindowsFileDialog открывает стандартный диалог выбора файлов Windows
func openWindowsFileDialog(filter []string, multiSelect bool) ([]string, error) {
	psScript := `
		[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
		Add-Type -AssemblyName System.Windows.Forms
		$dialog = New-Object System.Windows.Forms.OpenFileDialog
		$dialog.Title = "Select files"
		$dialog.Multiselect = $true
		`

	if len(filter) > 0 {
		filterExtensions := []string{}

		for _, ext := range filter {
			cleanExt := strings.TrimPrefix(ext, ".")
			filterExtensions = append(filterExtensions, "*."+cleanExt)
		}

		filterStr := strings.Join(filterExtensions, ";")
		displayName := "Files"

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

	cmd := exec.Command("powershell", "-Command", psScript)

	cmd.Env = append(os.Environ(),
		"PYTHONIOENCODING=utf-8",
		"LANG=en_US.UTF-8",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 0 {
			return []string{}, nil
		}
		return nil, fmt.Errorf("dialog error: %v\nstderr: %s", err, stderr.String())
	}

	outputBytes := stdout.Bytes()

	var outputStr string
	if utf8.Valid(outputBytes) {
		outputStr = string(outputBytes)
	} else {
		if dec, err := charmap.Windows1251.NewDecoder().Bytes(outputBytes); err == nil {
			outputStr = string(dec)
		} else {
			outputStr = string(outputBytes)
		}
	}

	outputStr = strings.TrimSpace(outputStr)
	if outputStr == "" {
		return []string{}, nil
	}

	lines := strings.ReplaceAll(outputStr, "\r\n", "\n")
	files := strings.Split(lines, "\n")

	var cleanFiles []string
	for _, file := range files {
		cleanFile := strings.TrimSpace(file)
		if cleanFile != "" && cleanFile != "\"" {
			if _, err := os.Stat(cleanFile); err == nil {
				cleanFiles = append(cleanFiles, cleanFile)
			} else {
				_ = err
			}
		}
	}

	return cleanFiles, nil
}

// openNativeFileDialog открывает нативный диалог выбора файлов через Fyne
func openNativeFileDialog(window fyne.Window, filter []string, _ bool) ([]string, error) {
	var selectedFiles []string

	done := make(chan bool, 1)

	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		defer func() { done <- true }()

		if err != nil {
			return
		}
		if reader == nil {
			return
		}
		defer func() { _ = reader.Close() }()

		selectedFiles = append(selectedFiles, reader.URI().Path())
	}, window)

	if len(filter) > 0 {
		extensions := make([]string, len(filter))
		copy(extensions, filter)
		fileDialog.SetFilter(storage.NewExtensionFileFilter(extensions))
	}

	fileDialog.Show()

	<-done

	return selectedFiles, nil
}

// SelectFiles открывает диалог выбора файлов (кроссплатформенный)
func SelectFiles(window fyne.Window, filter []string, multiSelect bool) ([]string, error) {
	if runtime.GOOS == "windows" {
		return openWindowsFileDialog(filter, multiSelect)
	}

	return openNativeFileDialog(window, filter, multiSelect)
}

// CreateFileUploadArea создает область для загрузки файлов
func CreateFileUploadArea(config FileUploadConfig, state *FileUploadState, parentWindow fyne.Window) *fyne.Container {
	if state.SelectedFiles == nil {
		emptySlice := []string{}
		state.SelectedFiles = &emptySlice
	}

	if state.UpdateDisplay == nil {
		state.UpdateDisplay = func() {}
	}

	box := canvas.NewRectangle(config.BackgroundColor)
	box.CornerRadius = 8
	box.SetMinSize(config.MinSize)

	filesContainer := container.NewVBox()

	contentContainer := container.NewVBox(filesContainer)

	containerWithContent := container.NewStack(box, contentContainer)

	clickButton := widget.NewButton(config.Label, func() {
		selectedFiles, err := SelectFiles(parentWindow, config.Filter, true)
		if err != nil {
			errorLabel := widget.NewLabel(fmt.Sprintf("Error selecting files:\n%v", err))
			errorLabel.Wrapping = fyne.TextWrapWord

			closeButton := widget.NewButton("Close", nil)

			popupContent := container.NewVBox(
				errorLabel,
				container.NewCenter(closeButton),
			)

			if parentWindow != nil {
				dialog := widget.NewModalPopUp(popupContent, parentWindow.Canvas())
				closeButton.OnTapped = func() {
					dialog.Hide()
				}
				dialog.Show()
			}
			return
		}

		if len(selectedFiles) > 0 {
			*state.SelectedFiles = append(*state.SelectedFiles, selectedFiles...)
			state.UpdateDisplay()
		}
	})
	clickButton.Importance = widget.LowImportance

	containerWithClick := container.NewStack(clickButton, containerWithContent)

	state.UpdateDisplay = func() {
		filesContainer.Objects = []fyne.CanvasObject{}

		selectedFiles := *state.SelectedFiles

		for i, filepath := range selectedFiles {
			filename := filepath
			if lastSlash := strings.LastIndex(filepath, "\\"); lastSlash != -1 {
				filename = filepath[lastSlash+1:]
			} else if lastSlash := strings.LastIndex(filepath, "/"); lastSlash != -1 {
				filename = filepath[lastSlash+1:]
			}

			var emoji string
			if IsImageFile(filename) {
				emoji = "🖼️ "
			} else {
				emoji = "📎 "
			}

			fileLabel := widget.NewLabel(emoji + filename)

			removeButton := widget.NewButton("❌", func(index int) func() {
				return func() {
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

			filesContainer.Add(hBox)
			if i < len(selectedFiles)-1 {
				filesContainer.Add(widget.NewSeparator())
			}
		}

		contentContainer.Objects = []fyne.CanvasObject{filesContainer}
		contentContainer.Refresh()
	}

	state.UpdateDisplay()

	return containerWithClick
}
