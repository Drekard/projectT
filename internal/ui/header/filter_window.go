package header

import (
	"image/color"

	"projectT/internal/services"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// FilterWindowManager управляет окном фильтров
type FilterWindowManager struct {
	popup         *widget.PopUp
	currentOpts   *services.FilterOptions
	onChange      func(services.FilterOptions)
	applyCallback func(services.FilterOptions)
}

// NewFilterWindowManager создает новый менеджер окна фильтров
func NewFilterWindowManager(onChange func(services.FilterOptions), applyCallback func(services.FilterOptions)) *FilterWindowManager {
	// Получаем текущие сохраненные настройки или используем значения по умолчанию
	opts := services.GlobalSortSettingsService.GetFilterOptions()
	return &FilterWindowManager{
		currentOpts:   opts,
		onChange:      onChange,
		applyCallback: applyCallback,
	}
}

// ShowFilterWindow показывает окно фильтров под кнопкой
func (fwm *FilterWindowManager) ShowFilterWindow(trigger fyne.CanvasObject) {
	window := fyne.CurrentApp().Driver().CanvasForObject(trigger)
	if window == nil {
		return
	}

	// Создаем содержимое окна фильтров
	content := fwm.createFilterWindowContent()

	fwm.popup = widget.NewPopUp(content, window)

	// Позиция триггера (кнопки)
	triggerPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(trigger)

	// Показываем под триггером, по центру
	menuPos := fyne.NewPos(
		triggerPos.X,
		triggerPos.Y+trigger.Size().Height,
	)

	// Проверяем, не выходит ли за нижнюю границу окна
	popupSize := fwm.popup.MinSize()
	windowSize := window.Size()

	if menuPos.Y+popupSize.Height > windowSize.Height {
		// Если выходит, показываем над триггером
		menuPos.Y = triggerPos.Y - popupSize.Height - 5
	}

	// Центрируем по горизонтали относительно триггера
	menuPos.X += (trigger.Size().Width - popupSize.Width) / 2

	fwm.popup.ShowAtPosition(menuPos)
}

// createFilterWindowContent создает содержимое окна фильтров
func (fwm *FilterWindowManager) createFilterWindowContent() *fyne.Container {
	// Фоновый контейнер
	bgContainer := container.NewStack()

	// Создаем содержимое формы
	formContent := fwm.createFilterForm()

	// Добавляем форму на фоновый контейнер
	bgContainer.Objects = append(bgContainer.Objects, formContent)

	return bgContainer
}

// createFilterForm создает форму фильтров с 4 колонками и кнопкой "Применить"
func (fwm *FilterWindowManager) createFilterForm() fyne.CanvasObject {
	// Column 1: Item types
	itemTypeGroup := widget.NewRadioGroup([]string{"All", "Folders", "Images", "Files", "Links", "Text"}, func(value string) {
		// Convert display value to internal representation
		switch value {
		case "Folders":
			fwm.currentOpts.ItemType = "folders"
		case "Images":
			fwm.currentOpts.ItemType = "images"
		case "Files":
			fwm.currentOpts.ItemType = "files"
		case "Links":
			fwm.currentOpts.ItemType = "links"
		case "Text":
			fwm.currentOpts.ItemType = "text"
		default:
			fwm.currentOpts.ItemType = "all"
		}
		// Обновляем настройки в глобальном сервисе, но НЕ вызываем onChange
		// Изменения будут применены только при нажатии кнопки "Применить"
	})

	// Set initial value
	switch fwm.currentOpts.ItemType {
	case "folders":
		itemTypeGroup.SetSelected("Folders")
	case "images":
		itemTypeGroup.SetSelected("Images")
	case "files":
		itemTypeGroup.SetSelected("Files")
	case "links":
		itemTypeGroup.SetSelected("Links")
	case "text":
		itemTypeGroup.SetSelected("Text")
	default:
		itemTypeGroup.SetSelected("All")
	}

	itemTypeColumn := container.NewVBox(
		widget.NewLabel("Only..."),
		itemTypeGroup,
	)

	// Column 2: Priority
	priorityGroup := widget.NewRadioGroup([]string{"None", "Folders first", "Images first", "Files first", "Links first", "Text first"}, func(value string) {
		// Convert display value to internal representation
		switch value {
		case "Folders first":
			fwm.currentOpts.Priority = "folders_first"
		case "Images first":
			fwm.currentOpts.Priority = "images_first"
		case "Files first":
			fwm.currentOpts.Priority = "files_first"
		case "Links first":
			fwm.currentOpts.Priority = "links_first"
		case "Text first":
			fwm.currentOpts.Priority = "text_first"
		default:
			fwm.currentOpts.Priority = "none"
		}
		// Обновляем настройки в глобальном сервисе, но НЕ вызываем onChange
		// Изменения будут применены только при нажатии кнопки "Применить"
	})

	// Set initial value
	switch fwm.currentOpts.Priority {
	case "folders_first":
		priorityGroup.SetSelected("Folders first")
	case "images_first":
		priorityGroup.SetSelected("Images first")
	case "files_first":
		priorityGroup.SetSelected("Files first")
	case "links_first":
		priorityGroup.SetSelected("Links first")
	case "text_first":
		priorityGroup.SetSelected("Text first")
	default:
		priorityGroup.SetSelected("None")
	}

	priorityColumn := container.NewVBox(
		widget.NewLabel("Priority:"),
		priorityGroup,
	)

	// Column 3: Sort by
	sortByGroup := widget.NewRadioGroup([]string{"By name", "By creation date", "By modification date", "By ContentMeta size"}, func(value string) {
		// Convert display value to internal representation
		switch value {
		case "By name":
			fwm.currentOpts.SortBy = "name"
		case "By creation date":
			fwm.currentOpts.SortBy = "created_date"
		case "By modification date":
			fwm.currentOpts.SortBy = "modified_date"
		case "By ContentMeta size":
			fwm.currentOpts.SortBy = "content_size"
		}
		// Обновляем настройки в глобальном сервисе, но НЕ вызываем onChange
		// Изменения будут применены только при нажатии кнопки "Применить"
	})

	// Set initial value
	switch fwm.currentOpts.SortBy {
	case "name":
		sortByGroup.SetSelected("By name")
	case "created_date":
		sortByGroup.SetSelected("By creation date")
	case "modified_date":
		sortByGroup.SetSelected("By modification date")
	case "content_size":
		sortByGroup.SetSelected("By ContentMeta size")
	default:
		sortByGroup.SetSelected("By name")
	}

	sortByColumn := container.NewVBox(
		widget.NewLabel("Sort by:"),
		sortByGroup,
	)

	// Column 4: Order
	orderGroup := widget.NewRadioGroup([]string{"Ascending", "Descending"}, func(value string) {
		// Convert display value to internal representation
		switch value {
		case "Ascending":
			fwm.currentOpts.SortOrder = "asc"
		case "Descending":
			fwm.currentOpts.SortOrder = "desc"
		}
		// Обновляем настройки в глобальном сервисе, но НЕ вызываем onChange
		// Изменения будут применены только при нажатии кнопки "Применить"
	})

	// Set initial value
	switch fwm.currentOpts.SortOrder {
	case "desc":
		orderGroup.SetSelected("Descending")
	default:
		orderGroup.SetSelected("Ascending")
	}

	orderColumn := container.NewVBox(
		widget.NewLabel("Order:"),
		orderGroup,
	)

	// Combine columns into grid
	columnsContainer := container.NewGridWithColumns(4, itemTypeColumn, priorityColumn, sortByColumn, orderColumn)

	// Create content for "This folder" tab - same fields but with different TabMode value
	thisFolderContent := container.NewVBox(columnsContainer)
	thisFolderTab := container.NewTabItem("This Folder", thisFolderContent)

	// Create content for "All items" tab - same fields but with different TabMode value
	allItemsContent := container.NewVBox(columnsContainer)
	allItemsTab := container.NewTabItem("All Items", allItemsContent)

	// Tab change handler
	filterTabs := container.NewAppTabs(thisFolderTab, allItemsTab)
	filterTabs.SetTabLocation(container.TabLocationTop)
	filterTabs.OnSelected = func(tab *container.TabItem) {
		if tab.Text == "This Folder" {
			fwm.currentOpts.TabMode = "current_folder"
		} else if tab.Text == "All Items" {
			fwm.currentOpts.TabMode = "all_items"
		}
	}

	// Create "Apply" button
	applyButton := widget.NewButton("Apply", func() {
		// Save changes to global service
		services.GlobalSortSettingsService.SetFilterOptions(fwm.currentOpts)

		// Call filter apply callback
		if fwm.applyCallback != nil {
			fwm.applyCallback(*fwm.currentOpts)
		}

		// Close window after applying
		if fwm.popup != nil {
			fwm.popup.Hide()
		}
	})

	// Create button container
	buttonContainer := container.NewHBox(container.NewPadded(applyButton))

	// Create vertical container for entire form
	formContainer := container.NewVBox(
		filterTabs,
		buttonContainer, // Move button to same level as tabs
	)

	// Wrap in container with padding and background
	bgRect := canvas.NewRectangle(color.RGBA{R: 44, G: 44, B: 44, A: 255}) // Gray background
	bgRect.CornerRadius = 8
	bgRect.StrokeColor = color.RGBA{R: 80, G: 80, B: 80, A: 255} // Dark gray border
	bgRect.StrokeWidth = 1
	bgRect.SetMinSize(fyne.NewSize(600, 320)) // Increased size for tabs and button

	outerContainer := container.NewStack(bgRect, container.NewPadded(formContainer))

	return outerContainer
}
