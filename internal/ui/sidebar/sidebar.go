package sidebar

import (
	"fmt"
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
	// Создаем контейнер для избранных элементов
	frequentContainer := container.NewVBox()

	// Функция для обновления содержимого
	updateContent := func() {
		frequentLabel := widget.NewLabel("Favorites")
		frequentLabel.TextStyle = fyne.TextStyle{Bold: true}

		buttons := make([]fyne.CanvasObject, 0)

		// Get favorite folders
		favoriteFolders, err := queries.GetFavoriteFolders()
		if err != nil {
			fmt.Printf("Error loading favorite folders: %v\n", err)
		} else {
			for _, folder := range favoriteFolders {
				buttonText := "📁 " + folder.Title
				btn := widget.NewButton(buttonText, func(folderID int) func() {
					return func() {
						// Navigate to selected folder
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

		// Get favorite tags
		favoriteTags, err := queries.GetFavoriteTags()
		if err != nil {
			fmt.Printf("Error loading favorite tags: %v\n", err)
		} else {
			for _, tag := range favoriteTags {
				buttonText := "# " + tag.Name
				btn := widget.NewButton(buttonText, func(tagName string) func() {
					return func() {
						// Set tag in search field
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

		// If no favorite items, add info message
		if len(buttons) == 0 {
			infoLabel := widget.NewLabel("No favorite items")
			infoLabel.TextStyle = fyne.TextStyle{Italic: true}
			buttons = append(buttons, infoLabel)
		}

		// Update container content
		frequentContainer.Objects = append([]fyne.CanvasObject{frequentLabel}, buttons...)
		frequentContainer.Refresh()
	}

	// Initialize content
	updateContent()

	// Subscribe to favorite change events
	eventChan := favorites.GetEventManager().Subscribe()
	go func() {
		for range eventChan {
			// Update container directly (in Fyne, Refresh updates may be safe)
			// Update container content
			frequentLabel := widget.NewLabel("Favorites")
			frequentLabel.TextStyle = fyne.TextStyle{Bold: true}

			buttons := make([]fyne.CanvasObject, 0)

			// Get favorite folders
			favoriteFolders, err := queries.GetFavoriteFolders()
			if err != nil {
				fmt.Printf("Error loading favorite folders: %v\n", err)
			} else {
				for _, folder := range favoriteFolders {
					buttonText := "📁 " + folder.Title
					btn := widget.NewButton(buttonText, func(folderID int) func() {
						return func() {
							// Navigate to selected folder
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

			// Get favorite tags
			favoriteTags, err := queries.GetFavoriteTags()
			if err != nil {
				fmt.Printf("Error loading favorite tags: %v\n", err)
			} else {
				for _, tag := range favoriteTags {
					buttonText := "# " + tag.Name
					btn := widget.NewButton(buttonText, func(tagName string) func() {
						return func() {
							// Set tag in search field
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

			// If no favorite items, add info message
			if len(buttons) == 0 {
				infoLabel := widget.NewLabel("No favorite items")
				infoLabel.TextStyle = fyne.TextStyle{Italic: true}
				buttons = append(buttons, infoLabel)
			}

			// Update container content
			frequentContainer.Objects = append([]fyne.CanvasObject{frequentLabel}, buttons...)
			frequentContainer.Refresh()
		}
	}()

	return frequentContainer
}
