package center

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// MessageInput поле ввода сообщения
type MessageInput struct {
	entry          *widget.Entry
	entryContainer *fyne.Container
	button         *widget.Button
}

// NewMessageInput создаёт новое поле ввода сообщения
func NewMessageInput(onSend func()) *MessageInput {
	mi := &MessageInput{}

	entryWidget := widget.NewMultiLineEntry()
	entryWidget.SetPlaceHolder("Введите сообщение...")
	entryWidget.Wrapping = fyne.TextWrapBreak

	mi.entryContainer = container.NewStack(entryWidget)
	mi.entry = entryWidget

	mi.button = widget.NewButtonWithIcon("", theme.MailSendIcon(), func() {
		if onSend != nil {
			onSend()
		}
	})
	mi.button.Importance = widget.HighImportance

	mi.entry.OnSubmitted = func(s string) {
		if onSend != nil {
			onSend()
		}
	}

	return mi
}

// Container возвращает контейнер поля ввода
func (mi *MessageInput) Container() *fyne.Container {
	return mi.entryContainer
}

// Text возвращает текст сообщения
func (mi *MessageInput) Text() string {
	return mi.entry.Text
}

// SetText устанавливает текст сообщения
func (mi *MessageInput) SetText(text string) {
	mi.entry.SetText(text)
}

// Clear очищает поле ввода
func (mi *MessageInput) Clear() {
	mi.entry.SetText("")
}

// SetEnabled устанавливает доступность поля ввода
func (mi *MessageInput) SetEnabled(enabled bool) {
	mi.entry.Disable()
	if enabled {
		mi.entry.Enable()
	}
}
