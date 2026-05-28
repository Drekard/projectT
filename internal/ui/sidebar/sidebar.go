package sidebar

import (
	"projectT/internal/services/favorites"
	"projectT/internal/services/p2p/protocols/transfer"
	"projectT/internal/storage/database/queries"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ChatUIProvider интерфейс для доступа к функциям UI чатов
type ChatUIProvider interface {
	OpenPeerChat(peerID, username string)
	RefreshContactsList()
	ShowChatMenu(chatID int, peerID string, username string, cont fyne.CanvasObject)
	OnChatDeleted(chatID int, peerID string)
	OpenLocalChat()
	IsCurrentChatLocal() bool
}

// SidebarState управляет состоянием боковой панели
type SidebarState struct {
	Collapsed     bool
	Width         float32
	ChatMode      bool
	ActiveSection string
	ChatUI        ChatUIProvider
	OnBackToNav   func()
	NavButtons    *NavigationButtons
}

// CreateSidebar создает боковую панель с навигацией
func CreateSidebar(state *SidebarState, handler NavigationHandler, transferSvc *transfer.Service) *fyne.Container {
	if state.ChatMode && state.ChatUI != nil {
		return createChatSidebar(state)
	}

	navigation, navButtons := CreateNavigation(state, handler)
	state.NavButtons = navButtons

	// Создаем виджет прогресса передачи файлов
	var transferProgress *TransferProgressWidget
	if transferSvc != nil {
		transferProgress = NewTransferProgressWidget(transferSvc)
	}

	// Основной контент sidebar
	var mainContent *fyne.Container
	if state.Collapsed {
		favoritesContainer := createCollapsedFavoritesSection(state, handler)
		mainContent = container.NewVBox(navigation, favoritesContainer)
	} else {
		frequentContainer := createFrequentlyUsedSection(state, handler)

		mainContent = container.NewVBox(
			navigation,
			frequentContainer,
		)
	}

	if transferProgress != nil {
		sidebarContainer := container.NewBorder(
			nil,                          // Top
			transferProgress.Container(), // Bottom - прогресс передачи
			nil,                          // Left
			nil,                          // Right
			mainContent,
		)
		sidebarContainer.Resize(fyne.NewSize(state.Width, 0))
		return sidebarContainer
	}

	mainContent.Resize(fyne.NewSize(state.Width, 0))
	return mainContent
}

// createFrequentlyUsedSection создает секцию "Часто используемые" (избранные элементы)
func createFrequentlyUsedSection(state *SidebarState, handler NavigationHandler) *fyne.Container {
	buildObjects := func() []fyne.CanvasObject {
		frequentLabel := widget.NewLabel("Favorites")
		frequentLabel.TextStyle = fyne.TextStyle{Bold: true}

		buttons := make([]fyne.CanvasObject, 0)

		favoriteFolders, err := queries.GetFavoriteFolders()
		if err == nil {
			for _, folder := range favoriteFolders {
				buttonText := "📁 " + folder.Title
				btn := widget.NewButton(buttonText, func(folderID int) func() {
					return func() {
						if handler != nil {
							_ = handler.NavigateToFolder(folderID)
						}
					}
				}(folder.ID))

				btn.Alignment = widget.ButtonAlignLeading
				btn.Importance = widget.LowImportance
				buttons = append(buttons, btn)
			}
		}

		favoriteTags, err := queries.GetFavoriteTags()
		if err == nil {
			for _, tag := range favoriteTags {
				buttonText := "# " + tag.Name
				btn := widget.NewButton(buttonText, func(tagName string) func() {
					return func() {
						if handler != nil {
							_ = handler.SetSearchQuery(tagName)
						}
					}
				}(tag.Name))

				btn.Alignment = widget.ButtonAlignLeading
				btn.Importance = widget.LowImportance
				buttons = append(buttons, btn)
			}
		}

		if len(buttons) == 0 {
			infoLabel := widget.NewLabel("No favorite items")
			infoLabel.TextStyle = fyne.TextStyle{Italic: true}
			buttons = append(buttons, infoLabel)
		}

		return append([]fyne.CanvasObject{frequentLabel}, buttons...)
	}

	frequentContainer := container.NewVBox(buildObjects()...)

	eventChan := favorites.GetEventManager().Subscribe()
	go func() {
		for range eventChan {
			if state.Collapsed {
				continue
			}

			fyne.Do(func() {
				frequentContainer.Objects = buildObjects()
				frequentContainer.Refresh()
			})
		}
	}()

	return frequentContainer
}

// createCollapsedFavoritesSection создает секцию избранных для свернутого sidebar (только иконки)
func createCollapsedFavoritesSection(state *SidebarState, handler NavigationHandler) *fyne.Container {
	separator := widget.NewSeparator()

	buildObjects := func() []fyne.CanvasObject {
		buttons := make([]fyne.CanvasObject, 0)

		favoriteFolders, err := queries.GetFavoriteFolders()
		if err == nil {
			for _, folder := range favoriteFolders {
				btn := widget.NewButton("📁", func(folderID int) func() {
					return func() {
						if handler != nil {
							_ = handler.NavigateToFolder(folderID)
						}
					}
				}(folder.ID))
				btn.Importance = widget.LowImportance
				buttons = append(buttons, btn)
			}
		}

		favoriteTags, err := queries.GetFavoriteTags()
		if err == nil {
			for _, tag := range favoriteTags {
				btn := widget.NewButton("#", func(tagName string) func() {
					return func() {
						if handler != nil {
							_ = handler.SetSearchQuery(tagName)
						}
					}
				}(tag.Name))
				btn.Importance = widget.LowImportance
				buttons = append(buttons, btn)
			}
		}

		if len(buttons) > 0 {
			return append([]fyne.CanvasObject{separator}, buttons...)
		}
		return []fyne.CanvasObject{separator}
	}

	favoritesContainer := container.NewVBox(buildObjects()...)

	eventChan := favorites.GetEventManager().Subscribe()
	go func() {
		for range eventChan {
			if state.Collapsed {
				fyne.Do(func() {
					favoritesContainer.Objects = buildObjects()
					favoritesContainer.Refresh()
				})
			}
		}
	}()

	return favoritesContainer
}
