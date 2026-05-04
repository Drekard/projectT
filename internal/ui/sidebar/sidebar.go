package sidebar

import (
	"projectT/internal/services/favorites"
	"projectT/internal/services/p2p/protocols/transfer"
	"projectT/internal/storage/database/queries"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// CreateSidebar создает боковую панель с навигацией
func CreateSidebar(width float32, handler NavigationHandler, transferSvc *transfer.Service) *fyne.Container {
	// Навигационные кнопки с передачей обработчика
	navigation := CreateNavigation(handler)

	// Создаем область "Часто используемые" с кликабельным текстом
	frequentContainer := createFrequentlyUsedSection(handler)

	// Создаем виджет прогресса передачи файлов
	var transferProgress *TransferProgressWidget
	if transferSvc != nil {
		transferProgress = NewTransferProgressWidget(transferSvc)
	}

	// Основной контент sidebar
	mainContent := container.NewVBox(
		navigation,
		frequentContainer,
	)

	// Используем Border для добавления прогресса передачи вниз
	if transferProgress != nil {
		sidebarContainer := container.NewBorder(
			nil,                          // Top
			transferProgress.Container(), // Bottom - прогресс передачи
			nil,                          // Left
			nil,                          // Right
			mainContent,
		)
		sidebarContainer.Resize(fyne.NewSize(width, 0))
		return sidebarContainer
	}

	sidebarContainer := container.NewVBox(
		navigation,
		frequentContainer,
	)
	sidebarContainer.Resize(fyne.NewSize(width, 0))

	return sidebarContainer
}

// createFrequentlyUsedSection создает секцию "Часто используемые" (избранные элементы)
func createFrequentlyUsedSection(handler NavigationHandler) *fyne.Container {
	frequentContainer := container.NewVBox()

	updateContent := func() {
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

		frequentContainer.Objects = append([]fyne.CanvasObject{frequentLabel}, buttons...)
		frequentContainer.Refresh()
	}

	updateContent()

	eventChan := favorites.GetEventManager().Subscribe()
	go func() {
		for range eventChan {
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

			frequentContainer.Objects = append([]fyne.CanvasObject{frequentLabel}, buttons...)
			frequentContainer.Refresh()
		}
	}()

	return frequentContainer
}
