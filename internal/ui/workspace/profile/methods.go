package profile

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"projectT/internal/services/background"
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

// GetAvatarPath возвращает текущий путь к аватару
func (p *UI) GetAvatarPath() string {
	return p.avatarPath
}

// SetWindow устанавливает окно для UI
func (p *UI) SetWindow(window fyne.Window) {
	p.window = window
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

// scheduleProfileAutoSave планирует автосохранение профиля (имя и титул) через 2 секунды
func (p *UI) scheduleProfileAutoSave() {
	// Отменяем предыдущие таймеры, если они существуют
	if p.userNameTimer != nil {
		p.userNameTimer.Stop()
	}
	if p.userTitleTimer != nil {
		p.userTitleTimer.Stop()
	}

	// Создаем новый таймер на 2 секунды
	p.userNameTimer = time.AfterFunc(2*time.Second, func() {
		p.autoSaveProfile()
	})
	p.userTitleTimer = p.userNameTimer
}

// autoSaveProfile сохраняет имя и статус профиля в базу данных
func (p *UI) autoSaveProfile() {
	p.saveCharacteristicsToDB()
}

// autoSaveField сохраняет поле в базу данных
func (p *UI) autoSaveField(row *fieldRow) {
	p.saveCharacteristicsToDB()
}

// saveCharacteristicsToDB сохраняет все характеристики в базу данных
func (p *UI) saveCharacteristicsToDB() {
	// Получаем текущий профиль
	profile, err := queries.GetLocalProfile()
	if err != nil {
		return
	}

	// Обновляем основные поля профиля
	profile.Username = p.userNameEntry.Text
	profile.Title = p.userTitleEntry.Text

	// Обновляем характеристики в JSON-формате
	characteristicsJSON, err := p.SaveCharacteristicsToJSON()
	if err != nil {
		return
	}
	profile.ContentChar = characteristicsJSON

	// Сохраняем изменения в базу данных
	err = queries.UpdateLocalProfile(profile)
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

	// Сохраняем изменения в базу данных
	p.saveCharacteristicsToDB()
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
	p.showImageDialog("Фон", "assets/background", "Нет сохраненных фонов", "Загрузить фон", "Удалить фон", func(imagePath string) error {
		backgroundService := background.NewService()
		err := backgroundService.SetBackground(imagePath)
		if err != nil {
			return fmt.Errorf("ошибка установки фона: %v", err)
		}
		p.backgroundPath = imagePath
		return nil
	}, func() {
		backgroundService := background.NewService()
		_ = backgroundService.ClearBackground() //nolint:errcheck
		p.backgroundPath = ""
	})
}

// showAvatarDialog показывает диалоговое окно для управления аватаром
func (p *UI) showAvatarDialog() {
	p.showImageDialog("Аватар", "assets/avatars", "Нет сохраненных аватаров", "Загрузить аватар", "", func(imagePath string) error {
		p.avatarPath = imagePath
		return nil
	}, func() {
		p.avatarPath = ""
	})
}

// showImageDialog показывает диалоговое окно для выбора изображения
func (p *UI) showImageDialog(
	title string,
	assetsDir string,
	noImagesLabel string,
	loadButtonLabel string,
	deleteButtonLabel string,
	onSelect func(imagePath string) error,
	onDelete func(),
) {
	// Создаем контейнер для миниатюр
	thumbnailsContainer := container.NewVBox()

	// Получаем список файлов из директории
	files, err := os.ReadDir(assetsDir)
	if err != nil {
		// Если директория не существует или произошла ошибка, создаем пустой контейнер
		thumbnailsContainer.Add(widget.NewLabel(noImagesLabel))
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
			if !file.IsDir() {
				ext := strings.ToLower(filepath.Ext(file.Name()))
				if imageExtensions[ext] {
					imagePath := filepath.Join(assetsDir, file.Name())

					// Создаем миниатюру изображения
					imageThumb := canvas.NewImageFromFile(imagePath)
					imageThumb.FillMode = canvas.ImageFillContain
					imageThumb.SetMinSize(fyne.NewSize(100, 100))

					// Создаем контейнер для миниатюры с именем файла
					fileLabel := widget.NewLabel(file.Name())
					fileLabel.Alignment = fyne.TextAlignCenter

					thumbContainer := container.NewVBox(imageThumb, fileLabel)

					// Добавляем возможность выбора по двойному клику
					clickableThumb := NewThumbnailClickable(thumbContainer, func() {
						err := onSelect(imagePath)
						if err != nil {
							dialog.ShowError(err, p.window)
							return
						}

						// Сохраняем в БД
						p.saveToDatabase()

						// Обновляем UI профиля
						p.createView()

						// Закрываем диалог и показываем уведомление
						dialog.ShowInformation("Успех", fmt.Sprintf("%s успешно установлен", strings.ToLower(title)), p.window)
					})

					thumbnailsContainer.Add(clickableThumb)
					hasImages = true
				}
			}
		}

		if !hasImages {
			thumbnailsContainer.Add(widget.NewLabel(noImagesLabel))
		}
	}

	// Создаем кнопки
	loadButton := widget.NewButton(loadButtonLabel, func() {
		if title == "Фон" {
			p.selectBackgroundImage()
		} else {
			p.selectAvatarImage()
		}
	})

	// Создаем контейнер для кнопок
	var buttonsContainer fyne.CanvasObject
	if deleteButtonLabel != "" {
		deleteButton := widget.NewButton(deleteButtonLabel, func() {
			onDelete()
			p.saveToDatabase()

			// Обновляем UI профиля
			p.createView()
		})
		buttonsContainer = container.NewHBox(loadButton, deleteButton)
	} else {
		buttonsContainer = container.NewHBox(loadButton)
	}

	// Создаем основной контейнер для диалога
	dialogContent := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Выберите %s:", strings.ToLower(title))),
		thumbnailsContainer,
		container.NewCenter(buttonsContainer),
	)

	// Показываем диалог
	dialog.ShowCustom(title, "Закрыть", dialogContent, p.window)
}
