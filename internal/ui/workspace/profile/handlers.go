package profile

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"projectT/internal/services/background"
	"projectT/internal/storage/database/queries"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/dialog"
	"golang.org/x/text/encoding/charmap"
)

// selectBackgroundImage открывает диалог выбора изображения для фона
func (p *UI) selectBackgroundImage() {
	// Используем Windows-диалог как в файле file_upload.go
	selectedFiles, err := openWindowsFileDialog([]string{".png", ".jpg", ".jpeg", ".gif", ".bmp"}, false)
	if err != nil {
		dialog.ShowError(err, p.window)
		return
	}

	if len(selectedFiles) == 0 {
		return // пользователь отменил операцию
	}

	originalFilePath := selectedFiles[0]

	// Создаем имя файла на основе оригинального имени
	originalFilename := filepath.Base(originalFilePath)
	ext := filepath.Ext(originalFilename)
	filename := strings.TrimSuffix(originalFilename, ext)
	safeFilename := fmt.Sprintf("%s%s", filename, ext)

	// Путь для сохранения в assets/background
	backgroundDir := "assets/background"
	finalPath := filepath.Join(backgroundDir, safeFilename)

	// Создаем директорию, если её нет
	if err := os.MkdirAll(backgroundDir, os.ModePerm); err != nil {
		dialog.ShowError(fmt.Errorf("ошибка создания директории: %v", err), p.window)
		return
	}

	// Копируем файл в assets/background
	err = copyFile(originalFilePath, finalPath)
	if err != nil {
		dialog.ShowError(fmt.Errorf("ошибка копирования файла: %v", err), p.window)
		return
	}

	// Используем сервис для установки фона
	backgroundService := background.NewService()
	err = backgroundService.SetBackground(finalPath)
	if err != nil {
		dialog.ShowError(fmt.Errorf("ошибка установки фона: %v", err), p.window)
		return
	}

	// Обновляем путь в UI
	p.backgroundPath = finalPath

	// Сохраняем изменения в БД
	p.saveToDatabase()

	// Обновляем UI профиля
	p.createView()

	dialog.ShowInformation("Успех", "Фон успешно установлен", p.window)
}

// saveToDatabase сохраняет изменения в базу данных
func (p *UI) saveToDatabase() {
	// Получаем текущий профиль
	profile, err := queries.GetLocalProfile()
	if err != nil {
		dialog.ShowError(fmt.Errorf("ошибка получения профиля: %v", err), p.window)
		return
	}

	// Обновляем основные поля профиля
	profile.Username = p.userNameEntry.Text
	profile.Title = p.userTitleEntry.Text
	profile.AvatarPath = p.avatarPath
	profile.BackgroundPath = p.backgroundPath

	// Сохраняем характеристики в JSON-формате
	characteristicsJSON, err := p.SaveCharacteristicsToJSON()
	if err != nil {
		dialog.ShowError(fmt.Errorf("ошибка сохранения характеристик: %v", err), p.window)
		return
	}
	profile.ContentChar = characteristicsJSON

	// Сохраняем изменения в базу данных
	err = queries.UpdateLocalProfile(profile)
	if err != nil {
		dialog.ShowError(fmt.Errorf("ошибка сохранения профиля: %v", err), p.window)
		return
	}

}

// copyFile копирует файл из src в dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// openWindowsFileDialog открывает стандартный диалог выбора файлов Windows
func openWindowsFileDialog(filter []string, multiSelect bool) ([]string, error) {
	// Создаем PowerShell скрипт для открытия диалога выбора файлов
	psScript := `
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Add-Type -AssemblyName System.Windows.Forms
$dialog = New-Object System.Windows.Forms.OpenFileDialog
$dialog.Title = "Выберите файлы"
$dialog.Multiselect = $false
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
				_ = err //nolint:staticcheck // Если файл не найден, продолжаем
			}
		}
	}

	return cleanFiles, nil
}

// selectAvatarImage открывает диалог выбора изображения для аватара
func (p *UI) selectAvatarImage() {
	// Используем Windows-диалог как в файле file_upload.go
	selectedFiles, err := openWindowsFileDialog([]string{".png", ".jpg", ".jpeg", ".gif", ".bmp"}, false)
	if err != nil {
		dialog.ShowError(err, p.window)
		return
	}

	if len(selectedFiles) == 0 {
		return // пользователь отменил операцию
	}

	originalFilePath := selectedFiles[0]

	// Создаем имя файла на основе оригинального имени
	originalFilename := filepath.Base(originalFilePath)
	ext := filepath.Ext(originalFilename)
	filename := strings.TrimSuffix(originalFilename, ext)
	safeFilename := fmt.Sprintf("%s%s", filename, ext)

	// Путь для сохранения в assets/avatars
	avatarDir := "assets/avatars"
	finalPath := filepath.Join(avatarDir, safeFilename)

	// Создаем директорию, если её нет
	if err := os.MkdirAll(avatarDir, os.ModePerm); err != nil {
		dialog.ShowError(fmt.Errorf("ошибка создания директории: %v", err), p.window)
		return
	}

	// Копируем файл в assets/avatars
	err = copyFile(originalFilePath, finalPath)
	if err != nil {
		dialog.ShowError(fmt.Errorf("ошибка копирования файла: %v", err), p.window)
		return
	}

	// Обновляем путь в UI
	p.avatarPath = finalPath

	// Сохраняем изменения в базу данных
	p.saveToDatabase()

	// Обновляем изображение аватара напрямую
	if p.avatarContainer != nil && len(p.avatarContainer.Objects) > 0 {
		if avatarClickable, ok := p.avatarContainer.Objects[0].(*AvatarClickableImage); ok {
			// Создаем новое изображение
			newImage := canvas.NewImageFromFile(finalPath)
			newImage.FillMode = canvas.ImageFillContain
			newImage.SetMinSize(fyne.NewSize(100, 100))

			// Обновляем p.avatarImage для консистентности
			p.avatarImage = newImage

			// Обновляем content в AvatarClickableImage
			avatarClickable.content = newImage
			avatarClickable.Refresh()
		}
	}

	dialog.ShowInformation("Успех", "Аватар успешно установлен", p.window)
}
