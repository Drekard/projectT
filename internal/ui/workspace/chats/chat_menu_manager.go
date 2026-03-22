// Package chats содержит компоненты UI для чатов
package chats

import (
	"fmt"

	"projectT/internal/storage/database/queries"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ChatMenuManager менеджер меню для чатов
type ChatMenuManager struct {
	onChatDeleted func(chatID int, peerID string)
	chatsUI       *UI
}

// NewChatMenuManager создает новый менеджер меню для чатов
func NewChatMenuManager(chatsUI *UI, onChatDeleted func(chatID int, peerID string)) *ChatMenuManager {
	return &ChatMenuManager{
		onChatDeleted: onChatDeleted,
		chatsUI:       chatsUI,
	}
}

// ShowChatMenu показывает меню действий для чата при двойном клике
func (cmm *ChatMenuManager) ShowChatMenu(chatID int, peerID string, username string, cont fyne.CanvasObject) {
	window := fyne.CurrentApp().Driver().CanvasForObject(cont)
	if window == nil {
		return
	}

	// Получаем позицию и размер элемента чата для центрирования попапа
	cardPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(cont)
	cardSize := cont.MinSize()

	// Создаем переменную для попапа, чтобы была возможность его закрыть из обработчика кнопки
	var popup *widget.PopUp

	var children []fyne.CanvasObject

	// Заголовок с информацией о чате
	headerText := fmt.Sprintf("**Чат**\n%s", username)
	children = append(children,
		widget.NewRichTextFromMarkdown(headerText),
	)

	// Кнопки действий
	buttons := []fyne.CanvasObject{}

	// Кнопка удаления чата
	deleteButton := widget.NewButton("🗑 Удалить чат", func() {
		cmm.showDeleteConfirmation(chatID, peerID, popup)
	})
	buttons = append(buttons, deleteButton)

	buttonsContainer := container.NewHBox(buttons...)

	children = append(children, buttonsContainer)

	content := container.NewVBox(children...)

	popup = widget.NewPopUp(content, window)

	// Показываем прямо под элементом чата
	menuPos := fyne.NewPos(
		cardPos.X,
		cardPos.Y+cardSize.Height,
	)

	// Проверяем, не выходит ли за нижнюю границу окна
	popupSize := popup.MinSize()
	windowSize := window.Size()

	if menuPos.Y+popupSize.Height > windowSize.Height {
		// Если выходит, показываем над элементом
		menuPos.Y = cardPos.Y - popupSize.Height - 5
	}

	popup.ShowAtPosition(menuPos)
}

// showDeleteConfirmation показывает диалог подтверждения удаления чата
func (cmm *ChatMenuManager) showDeleteConfirmation(chatID int, peerID string, parentPopup *widget.PopUp) {
	window := fyne.CurrentApp().Driver().AllWindows()[0]
	if window == nil {
		return
	}

	dialog.ShowConfirm("Подтверждение удаления",
		fmt.Sprintf("Вы уверены, что хотите удалить чат с %s?\n\nЭто действие нельзя отменить.", peerID[:8]),
		func(confirmed bool) {
			if confirmed {
				// Удаляем сообщения чата
				err := queries.DeleteMessagesForChat(chatID)
				if err != nil {
					dialog.ShowError(fmt.Errorf("Ошибка удаления сообщений: %v", err), window)
					return
				}

				// Удаляем чат
				err = queries.DeleteChat(chatID)
				if err != nil {
					dialog.ShowError(fmt.Errorf("Ошибка удаления чата: %v", err), window)
					return
				}

				// Вызываем колбэк удаления
				if cmm.onChatDeleted != nil {
					cmm.onChatDeleted(chatID, peerID)
				}

				// Закрываем родительский попап
				if parentPopup != nil {
					parentPopup.Hide()
				}
			}
		}, window)
}

// HandleChatDoubleClick обрабатывает двойной клик по элементу чата
func (cmm *ChatMenuManager) HandleChatDoubleClick(chatID int, peerID string, username string, cont fyne.CanvasObject) {
	cmm.ShowChatMenu(chatID, peerID, username, cont)
}
