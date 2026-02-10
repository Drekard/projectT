package header

import (
	"image/color"
	"projectT/internal/services"
	"projectT/internal/ui/header/create_item"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// HeaderSearchHandler интерфейс для обработки поиска
type HeaderSearchHandler interface {
	SearchItems(query string) error
	ClearSearch() error
}

// CreateHeader создает основную шапку приложения
func CreateHeader(sidebarVisible *bool, onToggle func(), width float32, searchHandler HeaderSearchHandler) (*fyne.Container, *BreadcrumbManager, *widget.Entry) {
	// Кнопка меню (бургер)
	menuButton := widget.NewButtonWithIcon("", theme.MenuIcon(), func() {
		*sidebarVisible = !*sidebarVisible
		onToggle()
	})
	menuButton.Importance = widget.LowImportance

	// Иконка приложения
	appIcon := LoadAppIcon()

	// Текст "ProjectT"
	appLabel := widget.NewLabel("ProjectT")
	// Увеличиваем размер шрифта
	appLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Хлебные крошки
	breadcrumbs, breadcrumbManager := CreateBreadcrumbs()

	// Кнопка фильтрации
	var filterButton *widget.Button
	filterButton = widget.NewButtonWithIcon("", theme.ListIcon(), func() {
		manager := NewFilterWindowManager(
			func(opts services.FilterOptions) {
				// Обработка изменений фильтров (оставляем для совместимости)
			},
			func(opts services.FilterOptions) {
				// Обработка применения фильтров - здесь будет вызов обновления сетки
				// Обновляем сетку элементов в рабочей области
				if searchHandler, ok := searchHandler.(interface{ ApplyFilters(services.FilterOptions) }); ok {
					// Если интерфейс поддерживает применение фильтров, вызываем его
					searchHandler.ApplyFilters(opts)
				} else {
					// Если не поддерживает, просто обновляем текущую папку
					if workspaceHandler, ok := searchHandler.(interface{ RefreshCurrentFolder() error }); ok {
						workspaceHandler.RefreshCurrentFolder()
					}
				}
			},
		)

		manager.ShowFilterWindow(filterButton)
	})
	filterButton.Importance = widget.LowImportance

	// Поле поиска
	searchBarContainer, searchEntry := CreateSearchBar(searchHandler)

	// Создаем прямоугольник для задания минимального размера
	searchWrapper := canvas.NewRectangle(color.Transparent)
	searchWrapper.SetMinSize(fyne.NewSize(300, 40))

	// Используем Stack, чтобы прямоугольник был фоном
	// и контейнер поиска занимал всю доступную ширину
	searchContainer := container.NewStack(
		searchWrapper,
		container.NewPadded(searchBarContainer), // Padded чтобы был небольшой отступ от краев
	)

	// Кнопка [+] - объявляем здесь, чтобы избежать проблемы с областью видимости
	var addButton *widget.Button
	addButton = widget.NewButton("[+]", func() {
		// Создаем экземпляр NewRectangleManager
		manager := create_item.NewNewRectangleManager(breadcrumbManager)

		// Отображаем NewRectangle под кнопкой
		manager.ShowNewRectangle(addButton, func() {
			// Функция обратного вызова при закрытии
		})
	})

	// Spacer между иконкой и лейблом
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(10, 10))
	iconLabelContainer := container.NewHBox(
		appIcon,
		spacer,
		appLabel,
	)
	// Фиксируем ширину контейнера с иконкой и названием
	iconLabelContainer = container.NewPadded(iconLabelContainer)
	leftWrapper := canvas.NewRectangle(color.Black)
	leftWrapper.SetMinSize(fyne.NewSize(140, 40))
	iconLabelWrapper := container.NewStack(
		leftWrapper,
		iconLabelContainer,
	)

	// Теперь создаем основной контейнер с двумя частями
	leftContainer := container.NewHBox(
		iconLabelWrapper,               // Контейнер с иконкой и названием (ширина 165)
		container.NewCenter(addButton), // Квадратная кнопка рядом
	)

	// Обертка для всего блока
	fullWrapper := canvas.NewRectangle(color.Transparent)
	fullWrapper.SetMinSize(fyne.NewSize(140+40, 40)) // 165 + ширина кнопки

	// Стек для фона и содержимого
	leftContainer = container.NewStack(
		fullWrapper,
		leftContainer,
	)

	// Центральная часть (хлебные крошки и кнопка фильтра)
	centerContainer := container.NewPadded(breadcrumbs)
	searchWithFilter := container.NewHBox(filterButton, searchContainer)

	// Компоновка шапки
	header := container.NewBorder(nil, nil, leftContainer, searchWithFilter, centerContainer)

	return header, breadcrumbManager, searchEntry
}
