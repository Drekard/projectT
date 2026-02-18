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

	appWorkspace := workspace.CreateWorkspace(window)

	ml := &MainLayout{
		workspace:          appWorkspace,
		sidebarVisible:     true,
		widthHeaderSidebar: widthHeaderSidebar,
	}

	appHeader, breadcrumbManager, searchEntry := header.CreateHeader(&ml.sidebarVisible, ml.toggleSidebar, widthHeaderSidebar, appWorkspace)
	ml.searchEntry = searchEntry

	hover_preview.SetGlobalSearchEntry(searchEntry)

	handler := &workspaceNavigationHandler{
		workspace:   appWorkspace,
		searchEntry: searchEntry,
	}

	appSidebar := sidebar.CreateSidebar(widthHeaderSidebar, handler)

	borderColor := color.NRGBA{R: 144, G: 55, B: 255, A: 255}

	headerBorder := canvas.NewRectangle(borderColor)
	headerBorder.SetMinSize(fyne.NewSize(1, 1.5))

	sidebarBorder := canvas.NewRectangle(borderColor)
	sidebarBorder.SetMinSize(fyne.NewSize(1, 1))

	sidebarWithBorder := container.NewBorder(
		nil, nil, nil, sidebarBorder,
		appSidebar,
	)

	ml.workspace.GetNavigationManager().SetBreadcrumbUpdateCallback(breadcrumbManager.UpdateBreadcrumbs)

	breadcrumbManager.SetNavigationCallback(func(folderID int) {
		ml.workspace.NavigateToFolder(folderID)
	})

	breadcrumbManager.SetRefreshCallback(func() {
		ml.workspace.RefreshCurrentFolder()
	})

	headerBg := canvas.NewRectangle(color.RGBA{0, 0, 0, 255})
	headerWithBorder := container.NewStack(headerBg, container.NewBorder(
		nil, headerBorder, nil, nil,
		appHeader,
	))

	sidebarBg := canvas.NewRectangle(color.RGBA{0, 0, 0, 255})
	ml.sidebarContainer = container.NewStack(sidebarBg, sidebarWithBorder)

	mainBorderLayout := container.NewBorder(
		headerWithBorder,
		nil,
		ml.sidebarContainer,
		nil,
		appWorkspace.GetContainer(),
	)

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
