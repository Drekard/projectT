// Package center содержит компоненты центральной панели чата
package center

import (
	"fmt"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// MessageMenuManager менеджер меню для сообщений
type MessageMenuManager struct {
	onMessageUpdated func(message *models.ChatMessage)
	onMessageDeleted func(messageID int)
}

// NewMessageMenuManager создает новый менеджер меню для сообщений
func NewMessageMenuManager(onMessageUpdated func(message *models.ChatMessage), onMessageDeleted func(messageID int)) *MessageMenuManager {
	return &MessageMenuManager{
		onMessageUpdated: onMessageUpdated,
		onMessageDeleted: onMessageDeleted,
	}
}

// ShowMessageMenu показывает контекстное меню действий для сообщения (аналог ShowSimpleMenu для карточек)
func (mmm *MessageMenuManager) ShowMessageMenu(message *models.ChatMessage, cont fyne.CanvasObject, isOutgoing bool) {
	window := fyne.CurrentApp().Driver().CanvasForObject(cont)
	if window == nil {
		return
	}

	// Получаем позицию и размер сообщения для центрирования попапов
	cardPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(cont)
	cardSize := cont.MinSize()

	// Создаем переменную для попапа, чтобы была возможность его закрыть из обработчика кнопки
	var popup *widget.PopUp

	var children []fyne.CanvasObject

	// Заголовок с информацией о сообщении
	headerText := fmt.Sprintf("**Сообщение** от %s", message.SentAt.Format("02.01.2006 15:04"))
	children = append(children,
		widget.NewRichTextFromMarkdown(headerText),
		widget.NewLabel("Содержимое:"),
	)

	// Поле с содержимым сообщения (только для чтения)
	contentEntry := widget.NewEntry()
	contentEntry.SetText(message.Content)
	contentEntry.Disable()
	contentEntry.MultiLine = true
	contentEntry.Wrapping = fyne.TextWrapBreak
	children = append(children, contentEntry)

	// Кнопки действий
	buttons := []fyne.CanvasObject{}

	// Кнопка редактирования (только для исходящих сообщений)
	if isOutgoing {
		editButton := widget.NewButton("✏️ Редактировать", func() {
			mmm.showEditMessageDialog(message, popup)
		})
		buttons = append(buttons, editButton)
	}

	// Кнопка удаления (для всех сообщений)
	deleteButton := widget.NewButton("🗑 Удалить", func() {
		mmm.showDeleteConfirmation(message, popup)
	})
	buttons = append(buttons, deleteButton)

	// Кнопка закрытия
	closeButton := widget.NewButton("Закрыть", func() {
		popup.Hide()
	})
	buttons = append(buttons, closeButton)

	buttonsContainer := container.NewHBox(buttons...)

	children = append(children, buttonsContainer)

	content := container.NewVBox(children...)

	popup = widget.NewPopUp(content, window)

	// Показываем прямо под сообщением
	menuPos := fyne.NewPos(
		cardPos.X,
		cardPos.Y+cardSize.Height+5,
	)

	// Проверяем, не выходит ли за нижнюю границу окна
	popupSize := popup.MinSize()
	windowSize := window.Size()

	if menuPos.Y+popupSize.Height > windowSize.Height {
		// Если выходит, показываем над сообщением
		menuPos.Y = cardPos.Y - popupSize.Height - 5
	}

	popup.ShowAtPosition(menuPos)
}

// showEditMessageDialog показывает диалог редактирования сообщения
func (mmm *MessageMenuManager) showEditMessageDialog(message *models.ChatMessage, parentPopup *widget.PopUp) {
	window := fyne.CurrentApp().Driver().AllWindows()[0]
	if window == nil {
		return
	}

	// Поле для редактирования содержимого
	editEntry := widget.NewMultiLineEntry()
	editEntry.SetText(message.Content)
	editEntry.SetMinRowsVisible(5)
	editEntry.Wrapping = fyne.TextWrapBreak

	content := container.NewVBox(
		widget.NewLabel("Редактировать сообщение:"),
		editEntry,
	)

	dialog.ShowCustomConfirm("Редактирование сообщения", "Сохранить", "Отмена", content, func(confirmed bool) {
		if confirmed {
			newContent := editEntry.Text
			if newContent == "" {
				dialog.ShowError(fmt.Errorf("Сообщение не может быть пустым"), window)
				return
			}

			// Обновляем сообщение в БД
			message.Content = newContent
			err := queries.UpdateChatMessage(message)
			if err != nil {
				dialog.ShowError(fmt.Errorf("Ошибка обновления сообщения: %v", err), window)
				return
			}

			// Вызываем колбэк обновления
			if mmm.onMessageUpdated != nil {
				mmm.onMessageUpdated(message)
			}

			// Закрываем родительский попап
			if parentPopup != nil {
				parentPopup.Hide()
			}
		}
	}, window)
}

// showDeleteConfirmation показывает диалог подтверждения удаления
func (mmm *MessageMenuManager) showDeleteConfirmation(message *models.ChatMessage, parentPopup *widget.PopUp) {
	window := fyne.CurrentApp().Driver().AllWindows()[0]
	if window == nil {
		return
	}

	dialog.ShowConfirm("Подтверждение удаления",
		"Вы уверены, что хотите удалить это сообщение?",
		func(confirmed bool) {
			if confirmed {
				err := queries.DeleteChatMessage(message.ID)
				if err != nil {
					dialog.ShowError(fmt.Errorf("Ошибка удаления сообщения: %v", err), window)
					return
				}

				// Вызываем колбэк удаления
				if mmm.onMessageDeleted != nil {
					mmm.onMessageDeleted(message.ID)
				}

				// Закрываем родительский попап
				if parentPopup != nil {
					parentPopup.Hide()
				}
			}
		}, window)
}

// ClickableMessageBubble оборачивает MessageBubble с возможностью клика (аналог ClickableCard)
type ClickableMessageBubble struct {
	widget.BaseWidget
	bubble         *MessageBubble
	message        *models.ChatMessage
	isOutgoing     bool
	onTapped       func()
	onDoubleTapped func()
}

// NewClickableMessageBubble создает кликабельный пузырёк сообщения
func NewClickableMessageBubble(
	message *models.ChatMessage,
	isOutgoing bool,
	onTapped func(),
	onDoubleTapped func(),
) *ClickableMessageBubble {
	cmb := &ClickableMessageBubble{
		message:        message,
		isOutgoing:     isOutgoing,
		onTapped:       onTapped,
		onDoubleTapped: onDoubleTapped,
	}
	cmb.bubble = NewMessageBubble(message, isOutgoing)
	cmb.ExtendBaseWidget(cmb)
	return cmb
}

// CreateRenderer создает рендерер для ClickableMessageBubble
func (cmb *ClickableMessageBubble) CreateRenderer() fyne.WidgetRenderer {
	// Создаем контейнер с пузырьком
	container := container.NewStack(cmb.bubble.Container())

	return &ClickableMessageBubbleRenderer{
		clickableBubble: cmb,
		container:       container,
	}
}

// MinSize возвращает минимальный размер
func (cmb *ClickableMessageBubble) MinSize() fyne.Size {
	return cmb.bubble.Container().MinSize()
}

// Tapped обрабатывает левый клик по сообщению
func (cmb *ClickableMessageBubble) Tapped(ev *fyne.PointEvent) {
	// Левый клик пока не используется
}

// TappedSecondary обрабатывает правый клик по сообщению (контекстное меню)
func (cmb *ClickableMessageBubble) TappedSecondary(ev *fyne.PointEvent) {
	if cmb.onTapped != nil {
		cmb.onTapped()
	}
}

// DoubleTapped обрабатывает двойной клик по сообщению
func (cmb *ClickableMessageBubble) DoubleTapped(ev *fyne.PointEvent) {
	if cmb.onDoubleTapped != nil {
		cmb.onDoubleTapped()
	}
}

// ClickableMessageBubbleRenderer рендерер для кликабельного пузырька
type ClickableMessageBubbleRenderer struct {
	clickableBubble *ClickableMessageBubble
	container       *fyne.Container
}

// Layout распологает элементы
func (r *ClickableMessageBubbleRenderer) Layout(size fyne.Size) {
	r.container.Resize(size)
}

// MinSize возвращает минимальный размер
func (r *ClickableMessageBubbleRenderer) MinSize() fyne.Size {
	return r.clickableBubble.MinSize()
}

// Refresh обновляет отображение
func (r *ClickableMessageBubbleRenderer) Refresh() {
	r.container.Refresh()
}

// Objects возвращает объекты для рендеринга
func (r *ClickableMessageBubbleRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.container}
}

// Destroy освобождает ресурсы
func (r *ClickableMessageBubbleRenderer) Destroy() {}
