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

// ChatHeaderState состояние шапки в режиме чата
type ChatHeaderState struct {
	ChatName            string
	OnBack              func()
	OnOpenProfile       func()
	OnAttach            func()
	OnToggleRight       func()
	OnOpenRemoteProfile func(peerID string)
	OnBackToNav         func()
	GetCurrentPeerID    func() string
	OnProfileClicked    func()
	IsLocalChat         func() bool
}

// HeaderState общее состояние шапки
type HeaderState struct {
	SidebarVisible  *bool
	OnToggleSidebar func()
	ChatMode        bool
	ChatHeader      *ChatHeaderState
}

// CreateHeader создает основную шапку приложения
func CreateHeader(state *HeaderState, width float32, searchHandler HeaderSearchHandler) (*fyne.Container, *BreadcrumbManager, *widget.Entry) {
	if state.ChatMode && state.ChatHeader != nil {
		return createChatHeader(state, width, searchHandler)
	}
	return createNormalHeader(state, width, searchHandler)
}

// createNormalHeader создает шапку для нормального режима
func createNormalHeader(state *HeaderState, width float32, searchHandler HeaderSearchHandler) (*fyne.Container, *BreadcrumbManager, *widget.Entry) {
	// Кнопка меню (бургер) - сворачивание sidebar
	menuButton := widget.NewButtonWithIcon("", theme.MenuIcon(), func() {
		state.OnToggleSidebar()
	})
	menuButton.Importance = widget.LowImportance

	// Иконка приложения
	appIcon := LoadAppIcon()

	// Хлебные крошки
	breadcrumbs, breadcrumbManager := CreateBreadcrumbs()

	// Кнопка фильтрации
	var filterButton *widget.Button
	filterButton = widget.NewButtonWithIcon("", theme.ListIcon(), func() {
		manager := NewFilterWindowManager(
			func(opts services.FilterOptions) {},
			func(opts services.FilterOptions) {
				if filterHandler, ok := searchHandler.(interface{ ApplyFilters(services.FilterOptions) }); ok {
					filterHandler.ApplyFilters(opts)
				} else {
					if workspaceHandler, ok := searchHandler.(interface{ RefreshCurrentFolder() error }); ok {
						_ = workspaceHandler.RefreshCurrentFolder()
					}
				}
			},
		)
		manager.ShowFilterWindow(filterButton)
	})
	filterButton.Importance = widget.LowImportance

	// Кнопка поиска
	var searchButton *widget.Button
	searchManager := NewSearchWindowManager(searchHandler)
	searchButton = widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
		searchManager.ShowSearchPopup(searchButton)
	})
	searchButton.Importance = widget.LowImportance

	searchEntry := searchManager.GetSearchEntry()

	// Кнопка [+]
	var addButton *widget.Button
	addButton = widget.NewButton("[+]", func() {
		manager := create_item.NewNewRectangleManager(breadcrumbManager)
		manager.ShowNewRectangle(addButton, func() {})
	})

	// Spacer между иконкой и лейблом
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(10, 10))
	iconLabelContainer := container.NewHBox(
		appIcon,
		spacer,
	)
	iconLabelContainer = container.NewPadded(iconLabelContainer)
	leftWrapper := canvas.NewRectangle(color.Black)
	leftWrapper.SetMinSize(fyne.NewSize(140, 40))
	iconLabelWrapper := container.NewStack(
		leftWrapper,
		container.NewCenter(addButton),
		iconLabelContainer,
	)

	// Левая часть: бургер + иконка/название + [+]
	leftContainer := container.NewHBox(
		menuButton,
		iconLabelWrapper,
	)

	fullWrapper := canvas.NewRectangle(color.Transparent)
	fullWrapper.SetMinSize(fyne.NewSize(140+40+40, 40))

	leftContainer = container.NewStack(
		fullWrapper,
		leftContainer,
	)

	centerContainer := container.NewPadded(breadcrumbs)
	searchWithFilter := container.NewHBox(filterButton, searchButton)

	header := container.NewBorder(nil, nil, leftContainer, searchWithFilter, centerContainer)

	return header, breadcrumbManager, searchEntry
}

// createChatHeader создает шапку для режима чата
func createChatHeader(state *HeaderState, width float32, searchHandler HeaderSearchHandler) (*fyne.Container, *BreadcrumbManager, *widget.Entry) {
	chatState := state.ChatHeader

	// Иконка приложения
	appIcon := LoadAppIcon()

	// Левая часть: иконка приложения
	leftContainer := container.NewHBox(appIcon)

	// Имя чата по центру
	chatName := chatState.ChatName
	if chatState.IsLocalChat != nil && chatState.IsLocalChat() {
		chatName = "Избранное"
	}
	chatNameLabel := widget.NewLabel(chatName)
	chatNameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Кнопка ℹ️ - открыть правую панель
	infoButton := widget.NewButtonWithIcon("", theme.InfoIcon(), func() {
		if chatState.OnToggleRight != nil {
			chatState.OnToggleRight()
		}
	})
	infoButton.Importance = widget.LowImportance

	// Кнопка аккаунта - перейти в normal mode и открыть профиль
	profilehButton := widget.NewButtonWithIcon("", theme.AccountIcon(), func() {
		if chatState.OnProfileClicked != nil {
			chatState.OnProfileClicked()
		}
	})
	profilehButton.Importance = widget.LowImportance

	// Кнопка поиска
	var searchButton *widget.Button
	searchManager := NewSearchWindowManager(searchHandler)
	searchButton = widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
		searchManager.ShowSearchPopup(searchButton)
	})
	searchButton.Importance = widget.LowImportance

	searchEntry := searchManager.GetSearchEntry()

	// Правая часть: 📎 + ℹ️ (поиск скрыт в chat mod)
	rightContainer := container.NewHBox(profilehButton, infoButton)

	// Центральная часть: имя чата
	centerContainer := container.NewCenter(chatNameLabel)

	// Компоновка шапки: слева иконка, справа кнопки, центр имя
	header := container.NewBorder(nil, nil, leftContainer, rightContainer, centerContainer)

	return header, nil, searchEntry
}
