package hover_preview

import (
	"time"

	"projectT/internal/services/favorites"
	"projectT/internal/services/pinned"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/edit_item"
	"projectT/internal/ui/workspace/chats/dialogs"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// SearchHandler определяет интерфейс для обработки поисковых запросов
type SearchHandler interface {
	SetSearchQuery(query string)
}

// favoritesService - глобальный экземпляр сервиса избранного
var favoritesService = favorites.NewService()

// pinnedService - глобальный экземпляр сервиса закрепленных элементов
var pinnedService = pinned.NewService()

// globalSearchEntry глобальная ссылка на поисковую строку
var globalSearchEntry *widget.Entry

// MenuManager менеджер меню действий
type MenuManager struct {
	searchEntry *widget.Entry
}

// SetGlobalSearchEntry устанавливает глобальную ссылку на поисковую строку
func SetGlobalSearchEntry(entry *widget.Entry) {
	globalSearchEntry = entry
}

// NewMenuManager создает новый менеджер меню
func NewMenuManager() *MenuManager {
	return &MenuManager{}
}

// SetSearchEntry устанавливает ссылку на поисковую строку
func (mm *MenuManager) SetSearchEntry(entry *widget.Entry) {
	mm.searchEntry = entry
}

// ShowSimpleMenu показывает простое меню действий
func (mm *MenuManager) ShowSimpleMenu(item *models.Item, cont fyne.CanvasObject, onClose func(), noButtons ...bool) {
	window := fyne.CurrentApp().Driver().CanvasForObject(cont)
	if window == nil {
		return
	}

	cardPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(cont)
	cardSize := cont.MinSize()

	hideButtons := len(noButtons) > 0 && noButtons[0]

	var popup *widget.PopUp

	var children []fyne.CanvasObject

	if item.Title != "" {
		children = append(children,
			widget.NewRichTextFromMarkdown(getTitleForItem(item)),
		)
	}

	if item.Type == models.ItemTypeElement && item.ContentMeta != "" && item.Description != "" {
		descLabel := widget.NewLabel(getDescriptionForItem(item))
		descLabel.Wrapping = fyne.TextWrapWord
		children = append(children, descLabel)
	}

	tagsContainer := getTagsContainer(item, mm, cardPos, cardSize)
	if tagsContainer != nil {
		children = append(children, tagsContainer)
	}

	children = append(children,
		widget.NewLabel("Created: "+item.CreatedAt.Format("02.01.2006 15:04")),
		widget.NewLabel("Modified: "+item.UpdatedAt.Format("02.01.2006 15:04")),
	)

	if item.IsRemote() && item.SourcePeerID != nil {
		// TODO: по нажатию — переход на просмотр профиля человека
		ownerLabel := widget.NewLabel("Owner: " + formatPeerID(*item.SourcePeerID))
		ownerLabel.TextStyle = fyne.TextStyle{Italic: true}
		children = append(children, ownerLabel)
	}

	children = append(children,
		container.NewBorder(
			nil, nil, nil,
			func() fyne.CanvasObject {
				buttons := []fyne.CanvasObject{}

				if item.IsPreview() && item.IsRemote() {
					saveButton := widget.NewButton("Save to collection", func() {
						if err := mm.saveItemToCollection(item); err != nil {
							appWindow := fyne.CurrentApp().Driver().AllWindows()[0]
							dialog.ShowError(err, appWindow)
						} else {
							popup.Hide()
							if onClose != nil {
								onClose()
							}
						}
					})
					saveButton.Importance = widget.HighImportance
					buttons = append(buttons, saveButton)
				}

				if item.IsSaved() && item.IsRemote() {
					removeButton := widget.NewButton("Remove from collection", func() {
						appWindow := fyne.CurrentApp().Driver().AllWindows()[0]
						dialog.ShowConfirm("Remove from collection",
							"Element \""+item.Title+"\" will be returned to chat (status will change to 'preview'). Continue?",
							func(confirmed bool) {
								if confirmed {
									if err := mm.removeItemFromCollection(item); err != nil {
										dialog.ShowError(err, appWindow)
									} else {
										popup.Hide()
										if onClose != nil {
											onClose()
										}
									}
								}
							}, appWindow)
					})
					removeButton.Importance = widget.DangerImportance
					buttons = append(buttons, removeButton)
				}

				if !hideButtons {
					if item.IsLocal() {
						buttons = append(buttons,
							widget.NewButton("Edit", func() {
								appWindows := fyne.CurrentApp().Driver().AllWindows()
								if len(appWindows) > 0 {
									edit_item.ShowCreateItemModalForEdit(appWindows[0], item.ID)
								}
							}),
						)
					}

					buttons = append(buttons,
						widget.NewButton("Delete", func() {
							appWindow := fyne.CurrentApp().Driver().AllWindows()[0]
							dialog.ShowConfirm("Confirm deletion",
								"Are you sure you want to delete element \""+item.Title+"\"?",
								func(confirmed bool) {
									if confirmed {
										if err := mm.deleteItem(item); err != nil {
											dialog.ShowError(err, appWindow)
										} else {
											popup.Hide()
											if onClose != nil {
												onClose()
											}
										}
									}
								}, appWindow)
						}),
					)

					if item.Type == models.ItemTypeFolder {
						isFavorite, err := favoritesService.IsFavorite("folder", item.ElementUUID)
						if err != nil {
							isFavorite = false
						}

						var favButton *widget.Button
						var createFavHandler func(currentState bool) func()

						createFavHandler = func(currentState bool) func() {
							if currentState {
								return func() {
									err := favoritesService.RemoveFromFavorites("folder", item.ElementUUID)
									if err != nil {
										return
									}
									favButton.SetText("⭐️")
									favButton.OnTapped = createFavHandler(false)
								}
							} else {
								return func() {
									err := favoritesService.AddToFavorites("folder", item.ElementUUID)
									if err != nil {
										return
									}
									favButton.SetText("✨")
									favButton.OnTapped = createFavHandler(true)
								}
							}
						}

						if isFavorite {
							favButton = widget.NewButton("✨", createFavHandler(true))
						} else {
							favButton = widget.NewButton("⭐️", createFavHandler(false))
						}

						buttons = append([]fyne.CanvasObject{favButton}, buttons...)
					}

					// Visibility toggle button (only for local items)
					if item.IsLocal() {
						var visButton *widget.Button
						var createVisHandler func(isPublic bool) func()

						createVisHandler = func(isPublic bool) func() {
							if isPublic {
								return func() {
									newVis, err := queries.ToggleItemVisibility(item.ID)
									if err != nil {
										return
									}
									item.Visibility = newVis
									if newVis == models.ItemVisibilityPrivate {
										visButton.SetText("🔒")
										visButton.OnTapped = createVisHandler(false)
									} else {
										visButton.SetText("🌐")
										visButton.OnTapped = createVisHandler(true)
									}
								}
							} else {
								return func() {
									newVis, err := queries.ToggleItemVisibility(item.ID)
									if err != nil {
										return
									}
									item.Visibility = newVis
									if newVis == models.ItemVisibilityPrivate {
										visButton.SetText("🔒")
										visButton.OnTapped = createVisHandler(false)
									} else {
										visButton.SetText("🌐")
										visButton.OnTapped = createVisHandler(true)
									}
								}
							}
						}

						if item.IsPublic() {
							visButton = widget.NewButton("🌐", createVisHandler(true))
						} else {
							visButton = widget.NewButton("🔒", createVisHandler(false))
						}

						buttons = append([]fyne.CanvasObject{visButton}, buttons...)
					}

					sendButton := widget.NewButton("Send", func() {
						dialogs.ShowSendToChatDialog(item, func(chat *models.ChatWithLastMessage) {
							if popup != nil {
								popup.Hide()
							}
							appWindow := fyne.CurrentApp().Driver().AllWindows()[0]
							sendItemToChat(chat, item, appWindow)
						})
					})
					buttons = append([]fyne.CanvasObject{sendButton}, buttons...)

					moveButton := widget.NewButton("Move", func() {
						showMoveFolderSelection(popup, item)
					})
					buttons = append([]fyne.CanvasObject{moveButton}, buttons...)

					isPinned, err := pinnedService.IsItemPinned(item.ID)
					if err != nil {
						isPinned = false
					}

					var pinButton *widget.Button
					var createPinHandler func(currentState bool) func()

					createPinHandler = func(currentState bool) func() {
						if currentState {
							return func() {
								err := pinnedService.UnpinItem(item.ID)
								if err != nil {
									return
								}
								pinButton.SetText("📌")
								pinButton.OnTapped = createPinHandler(false)
							}
						} else {
							return func() {
								err := pinnedService.PinItem(item.ID)
								if err != nil {
									return
								}
								pinButton.SetText("✅📌")
								pinButton.OnTapped = createPinHandler(true)
							}
						}
					}

					if isPinned {
						pinButton = widget.NewButton("✅📌", createPinHandler(true))
					} else {
						pinButton = widget.NewButton("📌", createPinHandler(false))
					}

					buttons = append([]fyne.CanvasObject{pinButton}, buttons...)
				}

				return container.NewHBox(buttons...)
			}(),
		),
	)

	content := container.NewVBox(children...)

	popup = widget.NewPopUp(content, window)

	menuPos := fyne.NewPos(
		cardPos.X,
		cardPos.Y+50,
	)

	popupSize := popup.MinSize()
	windowSize := window.Size()

	if menuPos.Y+popupSize.Height > windowSize.Height {
		menuPos.Y = cardPos.Y - popupSize.Height - 5
	}

	popup.ShowAtPosition(menuPos)

	if onClose != nil {
		go func() {
			for {
				visible := false
				fyne.CurrentApp().Driver().DoFromGoroutine(func() {
					visible = popup.Visible()
				}, true)
				if !visible {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
			onClose()
		}()
	}
}

// SetSearchQuery устанавливает поисковый запрос
func (mm *MenuManager) SetSearchQuery(query string) {
	if mm.searchEntry != nil {
		mm.searchEntry.SetText(query)
		return
	}

	if globalSearchEntry != nil {
		globalSearchEntry.SetText(query)
		return
	}
}
