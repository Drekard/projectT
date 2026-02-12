package profile

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"projectT/internal/services"
	"projectT/internal/storage/database/queries"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (p *UI) CreateView() fyne.CanvasObject {
	return p.content
}

func (p *UI) GetContent() fyne.CanvasObject {
	return p.content
}

// Методы для работы с данными (будут использоваться при интеграции с БД)
func (p *UI) SetUserName(name string) {
	p.userNameLabel.SetText(name)
	p.userNameEntry.SetText(name)
}

// GetAvatarPath возвращает текущий путь к аватару
func (p *UI) GetAvatarPath() string {
	return p.avatarPath
}

// SetWindow устанавливает окно для UI
func (p *UI) SetWindow(window fyne.Window) {
	p.window = window
}

func (p *UI) SetUserStatus(status string) {
	if p.userStatusLabel != nil {
		p.userStatusLabel.SetText(status)
	}
	p.userStatusEntry.SetText(status)
}

func (p *UI) SetCustomField(index int, title, value string) {
	if index >= 0 && index < len(p.customFields) {
		p.customFields[index].titleLabel.SetText(title + ":")
		p.customFields[index].valueEntry.SetText(value)
	}
}

// AddCharacteristic добавляет новую характеристику в интерфейс
func (p *UI) AddCharacteristic() {
	row := &fieldRow{}

	// Присваиваем уникальный ID
	row.id = p.nextID
	p.nextID++ // увеличиваем счетчик для следующего элемента

	row.titleLabel = widget.NewLabel(":")
	row.titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	row.titleEntry = widget.NewEntry()
	row.titleEntry.PlaceHolder = "Название"

	// Добавляем обработчик изменения текста для автосохранения
	row.titleEntry.OnChanged = func(text string) {
		p.scheduleAutoSave(row)
	}

	entryWrapper := canvas.NewRectangle(color.Transparent)
	entryWrapper.SetMinSize(fyne.NewSize(140, 40)) // 165 + ширина кнопки
	entryContainer := container.NewStack(entryWrapper, row.titleEntry)

	row.valueEntry = widget.NewEntry()
	row.valueEntry.PlaceHolder = "Значение"

	// Добавляем обработчик изменения текста для автосохранения
	row.valueEntry.OnChanged = func(text string) {
		p.scheduleAutoSave(row)
	}

	// Кнопка удаления
	row.removeButton = widget.NewButton("❌", func() {
		p.RemoveCharacteristic(row)
	})
	row.removeButton.Importance = widget.LowImportance

	// Создаем контейнер с растягивающимися элементами
	row.container = container.NewBorder(
		nil,
		nil,
		container.NewHBox(entryContainer, row.titleLabel),
		row.removeButton,
		row.valueEntry,
	)

	p.characteristicsContainer.Add(row.container)

	// Добавляем в список полей для последующего сохранения
	p.customFields = append(p.customFields, row)
}

// scheduleAutoSave планирует автосохранение поля через 2 секунды
func (p *UI) scheduleAutoSave(row *fieldRow) {
	// Отменяем предыдущий таймер, если он существует
	if row.timer != nil {
		row.timer.Stop()
	}

	// Создаем новый таймер на 2 секунды
	row.timer = time.AfterFunc(2*time.Second, func() {
		p.autoSaveField(row)
	})
}

// autoSaveField сохраняет поле в базу данных
func (p *UI) autoSaveField(row *fieldRow) {

	// Получаем текущий профиль
	profile, err := queries.GetProfile()
	if err != nil {
		return
	}

	// Обновляем основные поля профиля
	profile.Username = p.userNameEntry.Text
	profile.Status = p.userStatusEntry.Text

	// Обновляем характеристики в JSON-формате
	characteristicsJSON, err := p.SaveCharacteristicsToJSON()
	if err != nil {
		return
	}
	profile.ContentCharacteristic = characteristicsJSON

	// Сохраняем изменения в базу данных
	err = queries.UpdateProfile(profile)
	if err != nil {
		return
	}

}

// RemoveCharacteristic удаляет характеристику из интерфейса
func (p *UI) RemoveCharacteristic(row *fieldRow) {

	// Отменяем таймер, если он существует
	if row.timer != nil {
		row.timer.Stop()
	}

	p.characteristicsContainer.Remove(row.container)

	// Удаляем из списка, если нужно сохранить ссылки
	for i, r := range p.customFields {
		if r == row {
			p.customFields = append(p.customFields[:i], p.customFields[i+1:]...)
			break
		}
	}
}

// LoadCharacteristicsFromJSON загружает характеристики из JSON-строки
func (p *UI) LoadCharacteristicsFromJSON(jsonStr string) {
	var characteristics []ContentCharacteristicItem
	if jsonStr != "" {
		err := json.Unmarshal([]byte(jsonStr), &characteristics)
		if err != nil {
			// В случае ошибки, просто выходим
			return
		}
	}

	// Очищаем текущий контейнер
	p.characteristicsContainer.Objects = nil
	p.characteristicsContainer.Refresh()

	// Добавляем характеристики в интерфейс
	for _, item := range characteristics {
		p.AddCharacteristic()
		// Устанавливаем значения для последнего добавленного элемента
		if len(p.customFields) > 0 {
			lastRow := p.customFields[len(p.customFields)-1]
			lastRow.id = item.ID
			lastRow.titleEntry.SetText(item.Title)
			lastRow.valueEntry.SetText(item.Value)
			// Обновляем метку названия тоже для режима просмотра
			lastRow.titleLabel.SetText(":")
		}
	}
}

// SaveCharacteristicsToJSON сохраняет характеристики в JSON-строку
func (p *UI) SaveCharacteristicsToJSON() (string, error) {
	var characteristics []ContentCharacteristicItem

	// Собираем все характеристики из интерфейса
	for _, row := range p.customFields {
		// Получаем название из поля ввода
		title := row.titleEntry.Text

		// Получаем значение из поля ввода
		value := row.valueEntry.Text

		characteristics = append(characteristics, ContentCharacteristicItem{
			ID:    row.id,
			Title: title,
			Value: value,
		})
	}

	jsonData, err := json.Marshal(characteristics)
	if err != nil {
		return "", err
	}

	jsonStr := string(jsonData)

	return jsonStr, nil
}

// showBackgroundDialog показывает диалоговое окно для управления фоновым изображением
func (p *UI) showBackgroundDialog() {

	// Создаем контейнер для миниатюр
	thumbnailsContainer := container.NewVBox()

	// Получаем список файлов из директории assets/background
	backgroundDir := "assets/background"
	files, err := os.ReadDir(backgroundDir)
	if err != nil {
		// Если директория не существует или произошла ошибка, создаем пустой контейнер
		thumbnailsContainer.Add(widget.NewLabel("Нет сохраненных фонов"))
	} else {

		// Фильтруем только файлы изображений
		imageExtensions := map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
			".gif":  true,
			".bmp":  true,
		}

		hasImages := false
		for _, file := range files {
			if !file.IsDir() { // Проверяем, что это файл, а не директория
				ext := strings.ToLower(filepath.Ext(file.Name()))
				if imageExtensions[ext] {
					// Создаем путь к файлу
					imagePath := filepath.Join(backgroundDir, file.Name())

					// Создаем миниатюру изображения
					imageThumb := canvas.NewImageFromFile(imagePath)
					imageThumb.FillMode = canvas.ImageFillContain
					imageThumb.SetMinSize(fyne.NewSize(100, 100))

					// Создаем контейнер для миниатюры с именем файла
					fileLabel := widget.NewLabel(file.Name())
					fileLabel.Alignment = fyne.TextAlignCenter

					thumbContainer := container.NewVBox(imageThumb, fileLabel)

					// Добавляем возможность выбора этого фона
					thumbButton := widget.NewButton("", func() {
						// Используем сервис для установки фона
						backgroundService := services.NewBackgroundService()
						err := backgroundService.SetBackground(imagePath)
						if err != nil {
							dialog.ShowError(fmt.Errorf("ошибка установки фона: %v", err), p.window)
							return
						}

						// Обновляем путь в UI
						p.backgroundPath = imagePath

						// Обновляем UI
						p.createView()

						// Восстанавливаем правильное состояние кнопок в зависимости от режима редактирования
						if p.editMode {
							p.addCharacteristicButton.Show()
						} else {
							p.addCharacteristicButton.Hide()
						}

						p.content.Refresh()

						// Закрываем диалог
						dialog.ShowInformation("Успех", "Фон успешно установлен", p.window)
					})
					thumbButton.Importance = widget.LowImportance
					thumbButton.Hidden = true // Скрываем кнопку, но она нужна для обработки клика по контейнеру

					// Объединяем миниатюру и кнопку
					clickableThumb := container.NewStack(thumbContainer, thumbButton)

					thumbnailsContainer.Add(clickableThumb)
					hasImages = true
				}
			}
		}

		if !hasImages {
			thumbnailsContainer.Add(widget.NewLabel("Нет сохраненных фонов"))
		}
	}

	// Создаем кнопки
	loadButton := widget.NewButton("Загрузить фон", func() {
		p.selectBackgroundImage()
	})

	deleteButton := widget.NewButton("Удалить фон", func() {
		p.deleteBackgroundImage()
	})

	closeButton := widget.NewButton("Закрыть", func() {
		// Диалог закроется автоматически при нажатии на кнопку "Закрыть" в заголовке
	})

	// Создаем контейнер для кнопок
	buttonsContainer := container.NewHBox(loadButton, deleteButton, closeButton)

	// Создаем основной контейнер для диалога
	dialogContent := container.NewVBox(
		widget.NewLabel("Выберите фоновое изображение:"),
		thumbnailsContainer,
		buttonsContainer,
	)

	// Показываем диалог
	dialog.ShowCustom("Фон", "Закрыть", dialogContent, p.window)
}
