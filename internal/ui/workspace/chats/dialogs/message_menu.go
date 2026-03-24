// Package dialogs содержит диалоги для чатов
package dialogs

import (
	"fmt"
	"time"

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
	)

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

	buttonsContainer := container.NewHBox(buttons...)

	children = append(children, buttonsContainer)

	content := container.NewVBox(children...)

	popup = widget.NewPopUp(content, window)

	// Показываем прямо под сообщением
	menuPos := fyne.NewPos(
		cardPos.X*2,
		cardPos.Y+cardSize.Height,
	)

	// Проверяем, не выходит ли за нижнюю границу окна
	popupSize := popup.MinSize()
	windowSize := window.Size()

	if menuPos.Y+popupSize.Height > windowSize.Height {
		// Если выходит, показываем над сообщением
		menuPos.Y = cardPos.Y - popupSize.Height - 5
	}

	popup.ShowAtPosition(menuPos)

	// Вызываем колбэк при закрытии
	go func() {
		// Периодически проверяем, закрыт ли попап, чтобы не нагружать CPU
		for popup.Visible() {
			time.Sleep(100 * time.Millisecond)
		}
	}()
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
