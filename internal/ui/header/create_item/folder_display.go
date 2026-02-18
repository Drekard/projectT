package create_item

import (
	"database/sql"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ResetButtonImportance сбрасывает важность всех кнопок в контейнере
func ResetButtonImportance(container *fyne.Container) {
	for _, obj := range container.Objects {
		if btn, ok := obj.(*widget.Button); ok {
			btn.Importance = widget.LowImportance
			btn.Refresh()
		}
	}
}

// GetAllItems возвращает все элементы из базы данных
func GetAllItems() ([]*models.Item, error) {
	// Выполняем SQL-запрос для получения всех элементов
	query := `SELECT id, type, title, description, content_meta, parent_id, created_at, updated_at FROM items ORDER BY updated_at DESC`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		var item models.Item
		var parentID sql.NullInt64
		err := rows.Scan(
			&item.ID, &item.Type, &item.Title, &item.Description, &item.ContentMeta, &parentID, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if parentID.Valid {
			parentIDValue := int(parentID.Int64)
			item.ParentID = &parentIDValue
		}

		items = append(items, &item)
	}

	return items, nil
}

// CreateCustomFolderButton создает кнопку для выбора папки
func CreateCustomFolderButton(label string, onClick func()) *widget.Button {
	button := widget.NewButton(label, onClick)
	button.Importance = widget.LowImportance
	return button
}

// CreateFolderSelection создает контейнер для выбора папки
func CreateFolderSelection(breadcrumbManager BreadcrumbManagerInterface) *fyne.Container {
	// Контейнер для кнопок папок
	folderButtonsContainer := container.NewVBox()

	// Получаем ID текущей папки из хлебных крошек
	currentFolderID := breadcrumbManager.GetCurrentFolderID()

	// Для получения всех папок мы должны получить все элементы и отфильтровать по типу
	allItems, err := GetAllItems()
	if err != nil {
		// В случае ошибки добавим хотя бы сообщение об этом
		errorLabel := widget.NewLabel("Ошибка загрузки папок")
		folderButtonsContainer.Add(errorLabel)
	} else {
		// Сначала добавляем текущую папку в начало списка (если она не root)
		if currentFolderID != nil && *currentFolderID != 0 {
			// Находим текущую папку в списке всех папок
			var currentFolder *models.Item
			var otherFolders []*models.Item

			for _, item := range allItems {
				if item.Type == models.ItemTypeFolder {
					if item.ID == *currentFolderID {
						currentFolder = item
					} else {
						otherFolders = append(otherFolders, item)
					}
				}
			}

			// Если текущая папка найдена, добавляем её первой с пометкой "(текущая)"
			if currentFolder != nil {
				// Устанавливаем текущую папку как выбранную по умолчанию
				setCurrentFolder(&currentFolder.ID, currentFolder.Title)
				
				currentFolderTitle := currentFolder.Title + " (текущая)"
				currentFolderButton := CreateCustomFolderButton(currentFolderTitle, func() {
					ResetButtonImportance(folderButtonsContainer)
					for _, obj := range folderButtonsContainer.Objects {
						if btn, ok := obj.(*widget.Button); ok && btn.Text == currentFolderTitle {
							btn.Importance = widget.MediumImportance
							btn.Refresh()
							break
						}
					}
					setCurrentFolder(&currentFolder.ID, currentFolder.Title)
				})
				currentFolderButton.Importance = widget.MediumImportance
				folderButtonsContainer.Add(currentFolderButton)
			}

			savedButton := CreateCustomFolderButton("Сохраненное", func() {
				ResetButtonImportance(folderButtonsContainer)
				for _, obj := range folderButtonsContainer.Objects {
					if btn, ok := obj.(*widget.Button); ok && btn.Text == "Сохраненное" {
						btn.Importance = widget.MediumImportance
						btn.Refresh()
						break
					}
				}
				setCurrentFolder(nil, "Сохраненное")
			})
			folderButtonsContainer.Add(savedButton)

			// Добавляем остальные папки
			for _, item := range otherFolders {
				// Создаем замыкание для захвата переменных
				itemCopy := *item // Разыменовываем указатель
				folderButton := CreateCustomFolderButton(itemCopy.Title, func(selectedItem models.Item) func() {
					return func() {
						// Обработка нажатия на папку - здесь можно передать ID папки
						// Сначала сбросим выделение с других кнопок
						ResetButtonImportance(folderButtonsContainer)
						// Установим выделение для текущей кнопки
						// Найдем кнопку, соответствующую выбранному элементу
						for _, obj := range folderButtonsContainer.Objects {
							if btn, ok := obj.(*widget.Button); ok && btn.Text == selectedItem.Title {
								btn.Importance = widget.MediumImportance
								btn.Refresh()
								break
							}
						}
						setCurrentFolder(&selectedItem.ID, selectedItem.Title)
					}
				}(itemCopy))
				folderButtonsContainer.Add(folderButton)
			}
		} else {
			// Если текущая папка - root (ID = 0), просто добавляем "Сохраненное" и остальные папки
			// Кнопка "Сохраненное" с ID = NULL
			savedButton := CreateCustomFolderButton("Сохраненное", func() {
				ResetButtonImportance(folderButtonsContainer)
				for _, obj := range folderButtonsContainer.Objects {
					if btn, ok := obj.(*widget.Button); ok && btn.Text == "Сохраненное" {
						btn.Importance = widget.MediumImportance
						btn.Refresh()
						break
					}
				}
				setCurrentFolder(nil, "Сохраненное")
			})
			savedButton.Importance = widget.MediumImportance // Устанавливаем как выбранную по умолчанию
			folderButtonsContainer.Add(savedButton)

			// Добавляем остальные папки
			for _, item := range allItems {
				if item.Type == models.ItemTypeFolder {
					// Создаем замыкание для захвата переменных
					itemCopy := *item // Разыменовываем указатель
					folderButton := CreateCustomFolderButton(itemCopy.Title, func(selectedItem models.Item) func() {
						return func() {
							// Обработка нажатия на папку - здесь можно передать ID папки
							// Сначала сбросим выделение с других кнопок
							ResetButtonImportance(folderButtonsContainer)
							// Установим выделение для текущей кнопки
							// Найдем кнопку, соответствующую выбранному элементу
							for _, obj := range folderButtonsContainer.Objects {
								if btn, ok := obj.(*widget.Button); ok && btn.Text == selectedItem.Title {
									btn.Importance = widget.MediumImportance
									btn.Refresh()
									break
								}
							}
							setCurrentFolder(&selectedItem.ID, selectedItem.Title)
						}
					}(itemCopy))
					folderButtonsContainer.Add(folderButton)
				}
			}
		}
	}

	// Добавим прокрутку, если папок много
	scrollContainer := container.NewVScroll(folderButtonsContainer)
	scrollContainer.SetMinSize(fyne.NewSize(200, 150))

	return container.NewVBox(scrollContainer)
}
