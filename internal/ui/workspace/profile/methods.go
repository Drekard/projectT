package profile

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
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
		p.customFields[index].valueLabel.SetText(value)
		p.customFields[index].valueEntry.SetText(value)
	}
}

func (p *UI) AddCustomField(title, value string) {
	row := &fieldRow{}

	row.titleLabel = widget.NewLabel(title + ":")
	row.titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	row.valueLabel = widget.NewLabel(value)

	row.valueEntry = widget.NewEntry()
	row.valueEntry.SetText(value)
	row.valueEntry.Hide()

	row.container = container.NewHBox(
		row.titleLabel,
		layout.NewSpacer(),
		row.valueLabel,
		row.valueEntry,
	)

	p.customFields = append(p.customFields, row)

	// Нужно обновить контейнер с полями
	// В реальном приложении нужно пересоздать вью
}
