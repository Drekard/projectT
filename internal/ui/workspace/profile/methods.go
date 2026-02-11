package profile

import (
	"encoding/json"
	"fmt"
	"image/color"
	"projectT/internal/storage/database/queries"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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
	fmt.Println("DEBUG: Вызван метод AddCharacteristic (создание пустого элемента)")
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
	fmt.Printf("DEBUG: Добавлен новый элемент в customFields. ID: %d, Всего элементов: %d\n", row.id, len(p.customFields))
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
	fmt.Printf("DEBUG: Автосохранение поля - ID: %d, Title: '%s', Value: '%s'\n", row.id, row.titleEntry.Text, row.valueEntry.Text)

	// Получаем текущий профиль
	profile, err := queries.GetProfile()
	if err != nil {
		fmt.Printf("DEBUG: Ошибка получения профиля для автосохранения: %v\n", err)
		return
	}

	// Обновляем основные поля профиля
	profile.Username = p.userNameEntry.Text
	profile.Status = p.userStatusEntry.Text

	// Обновляем характеристики в JSON-формате
	characteristicsJSON, err := p.SaveCharacteristicsToJSON()
	if err != nil {
		fmt.Printf("DEBUG: Ошибка сохранения характеристик: %v\n", err)
		return
	}
	profile.ContentCharacteristic = characteristicsJSON

	// Сохраняем изменения в базу данных
	err = queries.UpdateProfile(profile)
	if err != nil {
		fmt.Printf("DEBUG: Ошибка сохранения профиля в базу данных: %v\n", err)
		return
	}

	fmt.Println("DEBUG: Поле успешно автосохранено в базу данных")
}

// RemoveCharacteristic удаляет характеристику из интерфейса
func (p *UI) RemoveCharacteristic(row *fieldRow) {
	fmt.Println("DEBUG: Вызван метод RemoveCharacteristic")

	// Отменяем таймер, если он существует
	if row.timer != nil {
		row.timer.Stop()
	}

	p.characteristicsContainer.Remove(row.container)

	// Удаляем из списка, если нужно сохранить ссылки
	for i, r := range p.customFields {
		if r == row {
			p.customFields = append(p.customFields[:i], p.customFields[i+1:]...)
			fmt.Printf("DEBUG: Удален элемент из customFields. Осталось элементов: %d\n", len(p.customFields))
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
		fmt.Printf("DEBUG: Загружена характеристика - ID: %d, Title: '%s', Value: '%s'\n", item.ID, item.Title, item.Value)
	}
	fmt.Printf("DEBUG: Всего загружено характеристик: %d\n", len(characteristics))
}

// SaveCharacteristicsToJSON сохраняет характеристики в JSON-строку
func (p *UI) SaveCharacteristicsToJSON() (string, error) {
	fmt.Println("DEBUG: Начало сохранения характеристик в JSON")
	var characteristics []ContentCharacteristicItem

	// Собираем все характеристики из интерфейса
	for i, row := range p.customFields {
		fmt.Printf("DEBUG: Обработка элемента %d из %d\n", i+1, len(p.customFields))

		// Получаем название из поля ввода
		title := row.titleEntry.Text
		fmt.Printf("DEBUG: Название из поля ввода: '%s'\n", title)

		// Получаем значение из поля ввода
		value := row.valueEntry.Text
		fmt.Printf("DEBUG: Значение из поля ввода: '%s'\n", value)

		characteristics = append(characteristics, ContentCharacteristicItem{
			ID:    row.id,
			Title: title,
			Value: value,
		})
	}

	fmt.Printf("DEBUG: Всего собрано характеристик для сохранения: %d\n", len(characteristics))

	jsonData, err := json.Marshal(characteristics)
	if err != nil {
		fmt.Printf("DEBUG: Ошибка при маршалинге JSON: %v\n", err)
		return "", err
	}

	jsonStr := string(jsonData)
	fmt.Printf("DEBUG: Сохраняемый JSON: %s\n", jsonStr)

	return jsonStr, nil
}
