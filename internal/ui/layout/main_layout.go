package layout

import (
	"fmt"
	"projectT/internal/ui/cards/hover_preview"
	"projectT/internal/ui/header"
	"projectT/internal/ui/sidebar"
	"projectT/internal/ui/workspace"

	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// MainLayout управляет основным макетом приложения
type MainLayout struct {
	mainContainer      *fyne.Container
	sidebarContainer   *fyne.Container
	workspace          *workspace.Workspace
	searchEntry        *widget.Entry
	sidebarVisible     bool
	widthHeaderSidebar float32
}

// CreateMainLayout создает основной макет приложения
func CreateMainLayout(window fyne.Window) *fyne.Container {
	widthHeaderSidebar := float32(180)

	// Создаем рабочую область
	appWorkspace := workspace.CreateWorkspace(window)

	// Создаем заголовок с функцией переключения боковой панели
	ml := &MainLayout{
		workspace:          appWorkspace,
		sidebarVisible:     true,
		widthHeaderSidebar: widthHeaderSidebar,
	}

	appHeader, breadcrumbManager, searchEntry := header.CreateHeader(&ml.sidebarVisible, ml.toggleSidebar, widthHeaderSidebar, appWorkspace)
	ml.searchEntry = searchEntry

	// Устанавливаем глобальную ссылку на поисковую строку для MenuManager
	hover_preview.SetGlobalSearchEntry(searchEntry)

	// Создаем обработчик навигации
	handler := &workspaceNavigationHandler{
		workspace:   appWorkspace,
		searchEntry: searchEntry,
	}

	// Создаем боковую панель
	appSidebar := sidebar.CreateSidebar(widthHeaderSidebar, handler)

	// Создаем границы
	borderColor := color.NRGBA{R: 144, G: 55, B: 255, A: 255}

	headerBorder := canvas.NewRectangle(borderColor)
	headerBorder.SetMinSize(fyne.NewSize(1, 1.5))

	sidebarBorder := canvas.NewRectangle(borderColor)
	sidebarBorder.SetMinSize(fyne.NewSize(1, 1))

	// Оборачиваем боковую панель с границей
	sidebarWithBorder := container.NewBorder(
		nil, nil, nil, sidebarBorder,
		appSidebar,
	)

	// Устанавливаем callback для обновления хлебных крошек
	ml.workspace.GetNavigationManager().SetBreadcrumbUpdateCallback(breadcrumbManager.UpdateBreadcrumbs)

	// Устанавливаем callback для навигации из хлебных крошек
	breadcrumbManager.SetNavigationCallback(func(folderID int) {
		ml.workspace.NavigateToFolder(folderID)
	})

	// Устанавливаем callback для обновления текущей папки
	breadcrumbManager.SetRefreshCallback(func() {
		/*ml.workspace.UpdateContent("favorites")
		ml.workspace.UpdateContent("tags")*/
		ml.workspace.RefreshCurrentFolder()

		// Обновляем вкладки "Избранное" и "Теги", если они уже инициализированы
		// Вызываем обновление контента для этих вкладок, чтобы они получили свежие данные из базы
	})

	// Создаем черный фон для заголовка
	headerBg := canvas.NewRectangle(color.RGBA{0, 0, 0, 255})
	// Создаем контейнер для шапки с границей
	headerWithBorder := container.NewStack(headerBg, container.NewBorder(
		nil, headerBorder, nil, nil,
		appHeader,
	))

	// Создаем черный фон для боковой панели
	sidebarBg := canvas.NewRectangle(color.RGBA{0, 0, 0, 255})
	// Создаем контейнер для боковой панели и границы
	ml.sidebarContainer = container.NewStack(sidebarBg, sidebarWithBorder)

	// Создаем основной макет с Border
	mainBorderLayout := container.NewBorder(
		headerWithBorder,
		nil,
		ml.sidebarContainer,
		nil,
		appWorkspace.GetContainer(),
	)

	// Черный фон
	bgRect := canvas.NewRectangle(color.RGBA{0, 0, 0, 255})
	ml.mainContainer = container.NewStack(bgRect, mainBorderLayout)

	return ml.mainContainer
}

// toggleSidebar переключает видимость боковой панели
func (ml *MainLayout) toggleSidebar() {
	ml.sidebarVisible = !ml.sidebarVisible

	if ml.sidebarVisible {
		ml.sidebarContainer.Show()
	} else {
		ml.sidebarContainer.Hide()
	}

	// Обновляем контейнер
	ml.mainContainer.Refresh()
}

// workspaceNavigationHandler реализует интерфейс NavigationHandler
type workspaceNavigationHandler struct {
	workspace   *workspace.Workspace
	searchEntry *widget.Entry
}

func (h *workspaceNavigationHandler) OnNavigationChanged(contentType string, param ...interface{}) {
	if h.workspace != nil {
		h.workspace.UpdateContent(contentType)
	}
}

func (h *workspaceNavigationHandler) NavigateToFolder(folderID int) error {
	if h.workspace != nil {
		return h.workspace.NavigateToFolder(folderID)
	}
	return nil
}

func (h *workspaceNavigationHandler) SearchByTag(tagName string) error {
	if h.workspace != nil {
		return h.workspace.SearchItems(tagName)
	}
	return nil
}

func (h *workspaceNavigationHandler) SetSearchQuery(query string) error {
	if h.searchEntry != nil {
		h.searchEntry.SetText(query)
		return nil
	}
	return fmt.Errorf("search entry is not initialized")
}
