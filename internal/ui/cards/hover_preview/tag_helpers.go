package hover_preview

import (
	"context"
	"image/color"
	"time"

	"projectT/internal/services"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// getTagsContainer возвращает контейнер с цветными кнопками тегов для элемента
func getTagsContainer(item *models.Item, handler SearchHandler, cardPos fyne.Position, cardSize fyne.Size) fyne.CanvasObject {
	tags, err := queries.GetTagsForItem(context.Background(), item.ID)
	if err != nil || len(tags) == 0 {
		return nil
	}

	var tagButtons []fyne.CanvasObject
	for _, tag := range tags {
		hexColor := tag.Color
		if hexColor == "" {
			hexColor = "#808080"
		}

		rgba, err := parseHexColor(hexColor)
		if err != nil {
			rgba = color.RGBA{R: 128, G: 128, B: 128, A: 255}
		}

		textColor := getContrastColor(rgba)

		tagBtn := NewTagButton(
			tag.Name,
			rgba,
			textColor,
			func(tagName, tagDescription string) func() {
				return func() {
					if tagDescription != "" {
						showTagDescriptionMenu(tagName, tagDescription, cardPos, cardSize)
					}
				}
			}(tag.Name, tag.Description),
		)

		tagBtn.OnTapped = func(tagName string) func() {
			return func() {
				if handler != nil {
					handler.SetSearchQuery(tagName)
				}
			}
		}(tag.Name)

		tagButtons = append(tagButtons, tagBtn)
	}

	return container.NewHBox(tagButtons...)
}

// showTagDescriptionMenu показывает меню с описанием тега
func showTagDescriptionMenu(tagName, tagDescription string, cardPos fyne.Position, cardSize fyne.Size) {
	window := fyne.CurrentApp().Driver().AllWindows()[0]
	if window == nil {
		return
	}

	canvas := window.Canvas()

	var children []fyne.CanvasObject

	children = append(children,
		widget.NewLabel(tagDescription),
	)

	content := container.NewVBox(children...)

	popup := widget.NewPopUp(content, canvas)

	popupSize := popup.MinSize()

	popup.ShowAtPosition(fyne.NewPos(
		cardPos.X+(cardSize.Width-popupSize.Width)/2,
		cardPos.Y+(cardSize.Height-popupSize.Height)/2,
	))

	go func() {
		for popup.Visible() {
			time.Sleep(100 * time.Millisecond)
		}
	}()
}

// showMoveFolderSelection показывает список папок для перемещения элемента
func showMoveFolderSelection(parentPopup *widget.PopUp, item *models.Item) {
	window := fyne.CurrentApp().Driver().AllWindows()[0]
	if window == nil {
		return
	}

	folderButtonsContainer := container.NewVBox()

	allItems, err := queries.GetSavedItems()
	if err != nil {
		errorLabel := widget.NewLabel("Error loading folders")
		folderButtonsContainer.Add(errorLabel)
	} else {
		savedButton := widget.NewButton("Saved", func() {
			menuManager := &MenuManager{}
			err := menuManager.MoveItemToFolder(item.ID, nil)
			if err != nil {
				appWindow := fyne.CurrentApp().Driver().AllWindows()[0]
				dialog.ShowError(err, appWindow)
			} else {
				parentPopup.Hide()
			}
		})
		savedButton.Importance = widget.LowImportance
		folderButtonsContainer.Add(savedButton)

		for _, folderItem := range allItems {
			if folderItem.Type == models.ItemTypeFolder && folderItem.ID != item.ID {
				folderCopy := *folderItem
				folderButton := widget.NewButton(folderCopy.Title, func(selectedFolder models.Item) func() {
					return func() {
						folderID := selectedFolder.ID
						menuManager := &MenuManager{}
						err := menuManager.MoveItemToFolder(item.ID, &folderID)
						if err != nil {
							appWindow := fyne.CurrentApp().Driver().AllWindows()[0]
							dialog.ShowError(err, appWindow)
						} else {
							parentPopup.Hide()
						}
					}
				}(folderCopy))
				folderButton.Importance = widget.LowImportance
				folderButtonsContainer.Add(folderButton)
			}
		}
	}

	scrollContainer := container.NewVScroll(folderButtonsContainer)
	scrollContainer.SetMinSize(fyne.NewSize(200, 150))

	content := container.NewVBox(
		widget.NewLabel("Select a folder to move to:"),
		scrollContainer,
	)

	dialog.ShowCustom("Move to folder", "Cancel", content, window)
}

// sendItemToChat отправляет элемент или папку в выбранный чат
func sendItemToChat(chat *models.ChatWithLastMessage, item *models.Item, window fyne.Window) {
	localProfile, err := queries.GetLocalProfile()
	if err != nil {
		dialog.ShowError(err, window)
		return
	}

	contactID := 0
	if chat.PeerID != models.LocalChatPeerID && chat.ContactID != nil {
		contactID = *chat.ContactID
	}

	recipientPeerID := chat.PeerID

	if item.Type == models.ItemTypeFolder {
		_, err = services.GetChatService().SendFolderMessage(contactID, recipientPeerID, localProfile.PeerID, item)
		if err != nil {
			dialog.ShowError(err, window)
			return
		}
		dialog.ShowInformation("Success", "Folder '"+item.Title+"' sent to chat", window)
	} else {
		_, err = services.GetChatService().SendElementMessage(contactID, recipientPeerID, localProfile.PeerID, item)
		if err != nil {
			dialog.ShowError(err, window)
			return
		}
		dialog.ShowInformation("Success", "Element sent to chat", window)
	}
}
