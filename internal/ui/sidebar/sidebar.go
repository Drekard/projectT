package sidebar

import (
	"fmt"
	"projectT/internal/services/favorites"
	"projectT/internal/storage/database/queries"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// CreateSidebar создает боковую панель с навигацией
func CreateSidebar(width float32, handler NavigationHandler) *fyne.Container {
	// Навигационные кнопки с передачей обработчика
	navigation := CreateNavigation(handler)

	// Создаем область "Часто используемые" с кликабельным текстом
	frequentContainer := createFrequentlyUsedSection(handler)

	// Общий контейнер для левой панели
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
		frequentLabel := widget.NewLabel("Избранные")
		frequentLabel.TextStyle = fyne.TextStyle{Bold: true}

		buttons := make([]fyne.CanvasObject, 0)

		// Получаем избранные папки
		favoriteFolders, err := queries.GetFavoriteFolders()
		if err != nil {
			fmt.Printf("Ошибка загрузки избранных папок: %v\n", err)
		} else {
			for _, folder := range favoriteFolders {
				buttonText := "📁 " + folder.Title
				btn := widget.NewButton(buttonText, func(folderID int) func() {
					return func() {
						// Переходим к выбранной папке
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

		// Получаем избранные теги
		favoriteTags, err := queries.GetFavoriteTags()
		if err != nil {
			fmt.Printf("Ошибка загрузки избранных тегов: %v\n", err)
		} else {
			for _, tag := range favoriteTags {
				buttonText := "# " + tag.Name
				btn := widget.NewButton(buttonText, func(tagName string) func() {
					return func() {
						// Устанавливаем тег в поисковую строку
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

		// Если нет избранных элементов, добавляем информационное сообщение
		if len(buttons) == 0 {
			infoLabel := widget.NewLabel("Нет избранных элементов")
			infoLabel.TextStyle = fyne.TextStyle{Italic: true}
			buttons = append(buttons, infoLabel)
		}

		// Обновляем содержимое контейнера
		frequentContainer.Objects = append([]fyne.CanvasObject{frequentLabel}, buttons...)
		frequentContainer.Refresh()
	}

	// Инициализируем содержимое
	updateContent()

	// Подписываемся на события изменения избранного
	eventChan := favorites.GetEventManager().Subscribe()
	go func() {
		for range eventChan {
			// Обновляем содержимое напрямую (в Fyne обновления через Refresh могут быть безопасными)
			// Обновляем содержимое контейнера
			frequentLabel := widget.NewLabel("Избранные")
			frequentLabel.TextStyle = fyne.TextStyle{Bold: true}

			buttons := make([]fyne.CanvasObject, 0)

			// Получаем избранные папки
			favoriteFolders, err := queries.GetFavoriteFolders()
			if err != nil {
				fmt.Printf("Ошибка загрузки избранных папок: %v\n", err)
			} else {
				for _, folder := range favoriteFolders {
					buttonText := "📁 " + folder.Title
					btn := widget.NewButton(buttonText, func(folderID int) func() {
						return func() {
							// Переходим к выбранной папке
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

			// Получаем избранные теги
			favoriteTags, err := queries.GetFavoriteTags()
			if err != nil {
				fmt.Printf("Ошибка загрузки избранных тегов: %v\n", err)
			} else {
				for _, tag := range favoriteTags {
					buttonText := "# " + tag.Name
					btn := widget.NewButton(buttonText, func(tagName string) func() {
						return func() {
							// Устанавливаем тег в поисковую строку
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

			// Если нет избранных элементов, добавляем информационное сообщение
			if len(buttons) == 0 {
				infoLabel := widget.NewLabel("Нет избранных элементов")
				infoLabel.TextStyle = fyne.TextStyle{Italic: true}
				buttons = append(buttons, infoLabel)
			}

			// Обновляем содержимое контейнера
			frequentContainer.Objects = append([]fyne.CanvasObject{frequentLabel}, buttons...)
			frequentContainer.Refresh()
		}
	}()

	return frequentContainer
}
