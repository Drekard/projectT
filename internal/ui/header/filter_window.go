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
	// Создаем вкладки для "Эта папка" и "Все элементы"
	tabs := container.NewAppTabs()

	// Колонка 1: Типы элементов
	itemTypeGroup := widget.NewRadioGroup([]string{"Все", "Папки", "Картинки", "Файлы", "Ссылки", "Текст"}, func(value string) {
		// Преобразуем отображаемое значение в внутреннее представление
		switch value {
		case "Папки":
			fwm.currentOpts.ItemType = "folders"
		case "Картинки":
			fwm.currentOpts.ItemType = "images"
		case "Файлы":
			fwm.currentOpts.ItemType = "files"
		case "Ссылки":
			fwm.currentOpts.ItemType = "links"
		case "Текст":
			fwm.currentOpts.ItemType = "text"
		default:
			fwm.currentOpts.ItemType = "all"
		}
		// Обновляем настройки в глобальном сервисе, но НЕ вызываем onChange
		// Изменения будут применены только при нажатии кнопки "Применить"
	})

	// Устанавливаем начальное значение
	switch fwm.currentOpts.ItemType {
	case "folders":
		itemTypeGroup.SetSelected("Папки")
	case "images":
		itemTypeGroup.SetSelected("Картинки")
	case "files":
		itemTypeGroup.SetSelected("Файлы")
	case "links":
		itemTypeGroup.SetSelected("Ссылки")
	case "text":
		itemTypeGroup.SetSelected("Текст")
	default:
		itemTypeGroup.SetSelected("Все")
	}

	itemTypeColumn := container.NewVBox(
		widget.NewLabel("Только..."),
		itemTypeGroup,
	)

	// Колонка 2: Приоритет
	priorityGroup := widget.NewRadioGroup([]string{"Нет", "Сначала папки", "Сначала картинки", "Сначала файлы", "Сначала ссылки", "Сначала текст"}, func(value string) {
		// Преобразуем отображаемое значение в внутреннее представление
		switch value {
		case "Сначала папки":
			fwm.currentOpts.Priority = "folders_first"
		case "Сначала картинки":
			fwm.currentOpts.Priority = "images_first"
		case "Сначала файлы":
			fwm.currentOpts.Priority = "files_first"
		case "Сначала ссылки":
			fwm.currentOpts.Priority = "links_first"
		case "Сначала текст":
			fwm.currentOpts.Priority = "text_first"
		default:
			fwm.currentOpts.Priority = "none"
		}
		// Обновляем настройки в глобальном сервисе, но НЕ вызываем onChange
		// Изменения будут применены только при нажатии кнопки "Применить"
	})

	// Устанавливаем начальное значение
	switch fwm.currentOpts.Priority {
	case "folders_first":
		priorityGroup.SetSelected("Сначала папки")
	case "images_first":
		priorityGroup.SetSelected("Сначала картинки")
	case "files_first":
		priorityGroup.SetSelected("Сначала файлы")
	case "links_first":
		priorityGroup.SetSelected("Сначала ссылки")
	case "text_first":
		priorityGroup.SetSelected("Сначала текст")
	default:
		priorityGroup.SetSelected("Нет")
	}

	priorityColumn := container.NewVBox(
		widget.NewLabel("Приоритет:"),
		priorityGroup,
	)

	// Колонка 3: Сортировка
	sortByGroup := widget.NewRadioGroup([]string{"По имени", "По дате создания", "По дате изменения", "По объему ContentMeta"}, func(value string) {
		// Преобразуем отображаемое значение в внутреннее представление
		switch value {
		case "По имени":
			fwm.currentOpts.SortBy = "name"
		case "По дате создания":
			fwm.currentOpts.SortBy = "created_date"
		case "По дате изменения":
			fwm.currentOpts.SortBy = "modified_date"
		case "По объему ContentMeta":
			fwm.currentOpts.SortBy = "content_size"
		}
		// Обновляем настройки в глобальном сервисе, но НЕ вызываем onChange
		// Изменения будут применены только при нажатии кнопки "Применить"
	})

	// Устанавливаем начальное значение
	switch fwm.currentOpts.SortBy {
	case "name":
		sortByGroup.SetSelected("По имени")
	case "created_date":
		sortByGroup.SetSelected("По дате создания")
	case "modified_date":
		sortByGroup.SetSelected("По дате изменения")
	case "content_size":
		sortByGroup.SetSelected("По объему ContentMeta")
	default:
		sortByGroup.SetSelected("По имени")
	}

	sortByColumn := container.NewVBox(
		widget.NewLabel("Сортировать:"),
		sortByGroup,
	)

	// Колонка 4: Порядок
	orderGroup := widget.NewRadioGroup([]string{"По возрастанию", "По убыванию"}, func(value string) {
		// Преобразуем отображаемое значение в внутреннее представление
		switch value {
		case "По возрастанию":
			fwm.currentOpts.SortOrder = "asc"
		case "По убыванию":
			fwm.currentOpts.SortOrder = "desc"
		}
		// Обновляем настройки в глобальном сервисе, но НЕ вызываем onChange
		// Изменения будут применены только при нажатии кнопки "Применить"
	})

	// Устанавливаем начальное значение
	switch fwm.currentOpts.SortOrder {
	case "desc":
		orderGroup.SetSelected("По убыванию")
	default:
		orderGroup.SetSelected("По возрастанию")
	}

	orderColumn := container.NewVBox(
		widget.NewLabel("Порядок:"),
		orderGroup,
	)

	// Комбинируем колонки в сетку
	columnsContainer := container.NewGridWithColumns(4, itemTypeColumn, priorityColumn, sortByColumn, orderColumn)

	// Создаем контент для вкладки "Эта папка" - те же поля, но с другим значением TabMode
	thisFolderContent := container.NewVBox(columnsContainer)
	thisFolderTab := container.NewTabItem("Эта папка", thisFolderContent)

	// Создаем контент для вкладки "Все элементы" - те же поля, но с другим значением TabMode
	allItemsContent := container.NewVBox(columnsContainer)
	allItemsTab := container.NewTabItem("Все элементы", allItemsContent)

	// Обработчик смены вкладки
	tabs = container.NewAppTabs(thisFolderTab, allItemsTab)
	tabs.SetTabLocation(container.TabLocationTop)
	tabs.OnSelected = func(tab *container.TabItem) {
		if tab.Text == "Эта папка" {
			fwm.currentOpts.TabMode = "current_folder"
		} else if tab.Text == "Все элементы" {
			fwm.currentOpts.TabMode = "all_items"
		}
	}

	// Создаем кнопку "Применить"
	applyButton := widget.NewButton("Применить", func() {
		// Сохраняем изменения в глобальный сервис
		services.GlobalSortSettingsService.SetFilterOptions(fwm.currentOpts)

		// Вызываем callback применения фильтров
		if fwm.applyCallback != nil {
			fwm.applyCallback(*fwm.currentOpts)
		}

		// Закрываем окно после применения
		if fwm.popup != nil {
			fwm.popup.Hide()
		}
	})

	// Создаем контейнер для кнопки
	buttonContainer := container.NewHBox(container.NewPadded(applyButton))

	// Создаем вертикальный контейнер для всей формы
	formContainer := container.NewVBox(
		tabs,
		buttonContainer, // Перемещаем кнопку на один уровень с вкладками
	)

	// Оборачиваем в контейнер с отступами и фоном
	bgRect := canvas.NewRectangle(color.RGBA{R: 44, G: 44, B: 44, A: 255}) // Серый фон
	bgRect.CornerRadius = 8
	bgRect.StrokeColor = color.RGBA{R: 80, G: 80, B: 80, A: 255} // Темно-серая обводка
	bgRect.StrokeWidth = 1
	bgRect.SetMinSize(fyne.NewSize(600, 320)) // Увеличили размер для размещения вкладок и кнопки

	outerContainer := container.NewStack(bgRect, container.NewPadded(formContainer))

	return outerContainer
}
