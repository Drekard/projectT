package layout

import (
	"fmt"
	"projectT/internal/config"
	"projectT/internal/storage/database/models"
	"projectT/internal/ui/cards/hover_preview"
	"projectT/internal/ui/header"
	"projectT/internal/ui/sidebar"
	"projectT/internal/ui/theme"
	"projectT/internal/ui/workspace"

	"image/color"
	p2p_network "projectT/internal/services/p2p/core"
	"projectT/internal/services/p2p/protocols/transfer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// MainLayout управляет основным макетом приложения
type MainLayout struct {
	mainContainer     *fyne.Container
	sidebarContainer  *fyne.Container
	workspace         *workspace.Workspace
	searchEntry       *widget.Entry
	sidebarState      *sidebar.SidebarState
	headerState       *header.HeaderState
	appHeader         *fyne.Container
	breadcrumbManager *header.BreadcrumbManager
	widthExpanded     float32
	widthCollapsed    float32
	window            fyne.Window
	currentBreadcrumb *header.BreadcrumbManager
	config            *config.Config
	onSave            func()
	sidebarBeforeChat bool // состояние sidebar до входа в чат
}

// CreateMainLayout создает основной макет приложения
func CreateMainLayout(window fyne.Window, p2pNetwork *p2p_network.P2PNetwork, cfg *config.Config, onSave func()) *fyne.Container {
	ml := &MainLayout{
		window: window,
		sidebarState: &sidebar.SidebarState{
			Collapsed: cfg.UISettings.SidebarCollapsed,
			Width:     180,
			ChatMode:  false,
		},
		widthExpanded:     180,
		widthCollapsed:    50,
		config:            cfg,
		onSave:            onSave,
		sidebarBeforeChat: cfg.UISettings.SidebarCollapsed,
	}

	if ml.sidebarState.Collapsed {
		ml.sidebarState.Width = ml.widthCollapsed
	}

	appWorkspace := workspace.CreateWorkspace(window, p2pNetwork, cfg, onSave)
	ml.workspace = appWorkspace

	// Состояние шапки
	ml.headerState = &header.HeaderState{
		SidebarVisible:  &ml.sidebarState.Collapsed,
		OnToggleSidebar: ml.toggleSidebar,
		ChatMode:        false,
	}

	appHeader, breadcrumbManager, searchEntry := header.CreateHeader(ml.headerState, ml.sidebarState.Width, appWorkspace)
	ml.appHeader = appHeader
	ml.breadcrumbManager = breadcrumbManager
	ml.searchEntry = searchEntry

	hover_preview.SetGlobalSearchEntry(searchEntry)

	handler := &workspaceNavigationHandler{
		workspace:   appWorkspace,
		searchEntry: searchEntry,
		mainLayout:  ml,
	}
	// Получаем сервис передачи файлов из P2P сети
	var transferSvc *transfer.Service = nil
	if p2pNetwork != nil {
		transferSvc = p2pNetwork.Transfer()
	}

	appSidebar := sidebar.CreateSidebar(ml.sidebarState, handler, transferSvc)

	// Инициализируем начальное содержимое - Saved
	ml.workspace.UpdateContent("saved")

	borderColor := theme.GetTheme().BorderColor

	headerBorder := canvas.NewRectangle(borderColor)
	headerBorder.SetMinSize(fyne.NewSize(1, 1.5))

	sidebarBorder := canvas.NewRectangle(borderColor)
	sidebarBorder.SetMinSize(fyne.NewSize(1, 1))

	theme.OnThemeChange(func() {
		headerBorder.FillColor = theme.GetTheme().BorderColor
		sidebarBorder.FillColor = theme.GetTheme().BorderColor
		headerBorder.Refresh()
		sidebarBorder.Refresh()
	})

	headerBg := canvas.NewRectangle(color.RGBA{0, 0, 0, 255})
	headerWithBorder := container.NewStack(headerBg, container.NewBorder(
		nil, headerBorder, nil, nil,
		appHeader,
	))

	sidebarWithBorder := container.NewBorder(
		nil, nil, nil, sidebarBorder,
		appSidebar,
	)

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

	ml.workspace.GetNavigationManager().SetBreadcrumbUpdateCallback(breadcrumbManager.UpdateBreadcrumbs)
	ml.currentBreadcrumb = breadcrumbManager

	appWorkspace.SetOnRemoteModeChanged(func(isRemote bool, peerID, peerName string, path []*models.Item) {
		if ml.currentBreadcrumb != nil {
			if isRemote {
				ml.currentBreadcrumb.UpdateRemoteBreadcrumbs(peerName, peerID, path)
			} else {
				ml.currentBreadcrumb.ResetToLocalMode()
				ml.currentBreadcrumb.UpdateBreadcrumbs(nil)
			}
		}
	})

	ml.setupBreadcrumbCallbacks(breadcrumbManager)

	appWorkspace.SetOnChatModeChanged(func(isChatMode bool, chatName string, onBack, onOpenProfile, onAttach, onToggleRight func()) {
		ml.setChatMode(isChatMode, chatName, onToggleRight)
	})

	return ml.mainContainer
}

// setChatMode переключает режим чата
func (ml *MainLayout) setChatMode(isChatMode bool, chatName string, onToggleRight func()) {
	ml.headerState.ChatMode = isChatMode
	ml.sidebarState.ChatMode = isChatMode

	if isChatMode {
		// Сохраняем состояние sidebar до входа в чат
		ml.sidebarBeforeChat = ml.sidebarState.Collapsed

		// Получаем chats UI из workspace
		if chatsUI := ml.workspace.GetChatsUI(); chatsUI != nil {
			ml.sidebarState.ChatUI = chatsUI
		}

		// Устанавливаем callback для кнопки "назад"
		ml.sidebarState.OnBackToNav = ml.backToNormalMode

		ml.headerState.ChatHeader = &header.ChatHeaderState{
			ChatName: chatName,
			OnToggleRight: func() {
				if chatsUI := ml.workspace.GetChatsUI(); chatsUI != nil {
					chatsUI.ToggleRightPanel()
				}
			},
			OnOpenRemoteProfile: func(peerID string) {
				if peerID != "" {
					ml.workspace.OpenRemoteProfile(peerID)
				}
			},
			OnBackToNav: ml.backToNormalMode,
			GetCurrentPeerID: func() string {
				if chatsUI := ml.workspace.GetChatsUI(); chatsUI != nil {
					return chatsUI.GetCurrentContactPeerID()
				}
				return ""
			},
			OnProfileClicked: func() {
				// Захватываем peerID пока ещё в chat mode
				peerID := ""
				if chatsUI := ml.workspace.GetChatsUI(); chatsUI != nil {
					peerID = chatsUI.GetCurrentContactPeerID()
				}
				if peerID != "" {
					if chatsUI := ml.workspace.GetChatsUI(); chatsUI != nil {
						chatsUI.OnBackToNormalMode()
						chatsUI.OpenRemoteProfile(peerID)
					} else {
						ml.backToNormalMode()
					}
				} else {
					ml.backToNormalMode()
				}
			},
			IsLocalChat: func() bool {
				if chatsUI := ml.workspace.GetChatsUI(); chatsUI != nil {
					return chatsUI.IsCurrentChatLocal()
				}
				return false
			},
		}
		ml.sidebarState.Collapsed = false
		ml.sidebarState.Width = ml.widthExpanded
	} else {
		ml.headerState.ChatHeader = nil
		ml.sidebarState.ChatUI = nil
		// Возвращаемся к предыдущему разделу или Saved
		if ml.sidebarState.ActiveSection == "" || ml.sidebarState.ActiveSection == "chats" {
			ml.sidebarState.ActiveSection = "saved"
		}
		// Восстанавливаем состояние sidebar которое было до входа в чат
		ml.sidebarState.Collapsed = ml.sidebarBeforeChat
		if ml.sidebarState.Collapsed {
			ml.sidebarState.Width = ml.widthCollapsed
		} else {
			ml.sidebarState.Width = ml.widthExpanded
		}
		ml.workspace.UpdateContent(ml.sidebarState.ActiveSection)
	}

	ml.rebuildAll()
}

// setupBreadcrumbCallbacks устанавливает callback'и для хлебных крошек
func (ml *MainLayout) setupBreadcrumbCallbacks(bm *header.BreadcrumbManager) {
	bm.SetNavigationCallback(func(folderID int) {
		ml.workspace.UpdateContent("saved")
		_ = ml.workspace.NavigateToFolder(folderID)
	})

	bm.SetRefreshCallback(func() {
		_ = ml.workspace.RefreshCurrentFolder()
	})

	bm.SetRemoteNavigationCallback(func(folderUUID string) {
		ml.workspace.NavigateToRemoteFolder(folderUUID)
	})

	bm.SetOpenRemoteProfileCallback(func(peerID string) {
		ml.workspace.OpenRemoteProfile(peerID)
	})
}

// rebuildAll перестраивает весь layout (для смены режима)
func (ml *MainLayout) rebuildAll() {
	handler := &workspaceNavigationHandler{
		workspace:   ml.workspace,
		searchEntry: ml.searchEntry,
		mainLayout:  ml,
	}

	var transferSvc *transfer.Service = nil
	if ml.workspace.GetP2PNetwork() != nil {
		transferSvc = ml.workspace.GetP2PNetwork().Transfer()
	}

	// Создаём новый sidebar
	newSidebar := sidebar.CreateSidebar(ml.sidebarState, handler, transferSvc)

	newHeader, newBreadcrumbManager, newSearchEntry := header.CreateHeader(ml.headerState, ml.sidebarState.Width, ml.workspace)
	ml.searchEntry = newSearchEntry
	hover_preview.SetGlobalSearchEntry(newSearchEntry)

	// Обновляем breadcrumb manager и устанавливаем callback'и
	if !ml.headerState.ChatMode && newBreadcrumbManager != nil {
		ml.currentBreadcrumb = newBreadcrumbManager
		ml.workspace.GetNavigationManager().SetBreadcrumbUpdateCallback(newBreadcrumbManager.UpdateBreadcrumbs)
		ml.setupBreadcrumbCallbacks(newBreadcrumbManager)
	}

	borderColor := theme.GetTheme().BorderColor

	headerBorder := canvas.NewRectangle(borderColor)
	headerBorder.SetMinSize(fyne.NewSize(0, 1.5))

	sidebarBorder := canvas.NewRectangle(borderColor)
	sidebarBorder.SetMinSize(fyne.NewSize(1, 1))

	sidebarWithBorder := container.NewBorder(
		nil, nil, nil, sidebarBorder,
		newSidebar,
	)

	headerBg := canvas.NewRectangle(color.RGBA{0, 0, 0, 255})
	headerWithBorder := container.NewStack(headerBg, container.NewBorder(
		nil, headerBorder, nil, nil,
		newHeader,
	))

	sidebarBg := canvas.NewRectangle(color.RGBA{0, 0, 0, 255})
	newSidebarContainer := container.NewStack(sidebarBg, sidebarWithBorder)

	mainBorderLayout := container.NewBorder(
		headerWithBorder,
		nil,
		newSidebarContainer,
		nil,
		ml.workspace.GetContainer(),
	)

	bgRect := canvas.NewRectangle(color.RGBA{0, 0, 0, 255})
	newMainContainer := container.NewStack(bgRect, mainBorderLayout)

	ml.mainContainer = newMainContainer
	ml.sidebarContainer = newSidebarContainer
	ml.window.SetContent(ml.mainContainer)
	ml.window.Resize(ml.window.Canvas().Size())
	ml.mainContainer.Refresh()
}

// toggleSidebar переключает свёрнутое состояние боковой панели
func (ml *MainLayout) toggleSidebar() {
	if ml.headerState.ChatMode {
		return
	}

	ml.sidebarState.Collapsed = !ml.sidebarState.Collapsed

	if ml.sidebarState.Collapsed {
		ml.sidebarState.Width = ml.widthCollapsed
	} else {
		ml.sidebarState.Width = ml.widthExpanded
	}

	// Сохраняем состояние sidebar в конфиг
	if ml.config != nil {
		ml.config.UISettings.SidebarCollapsed = ml.sidebarState.Collapsed
		if ml.onSave != nil {
			ml.onSave()
		}
	}

	ml.rebuildAll()
}

// backToNormalMode возвращает из режима чата в нормальный режим
func (ml *MainLayout) backToNormalMode() {
	ml.sidebarState.ChatMode = false
	ml.headerState.ChatMode = false
	ml.sidebarState.ChatUI = nil
	ml.headerState.ChatHeader = nil
	ml.sidebarState.ActiveSection = "saved"

	// Восстанавливаем состояние sidebar которое было до входа в чат
	ml.sidebarState.Collapsed = ml.sidebarBeforeChat
	if ml.sidebarState.Collapsed {
		ml.sidebarState.Width = ml.widthCollapsed
	} else {
		ml.sidebarState.Width = ml.widthExpanded
	}

	ml.workspace.UpdateContent("saved")
	ml.rebuildAll()
}

// workspaceNavigationHandler реализует интерфейс NavigationHandler
type workspaceNavigationHandler struct {
	workspace   *workspace.Workspace
	searchEntry *widget.Entry
	mainLayout  *MainLayout
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
	if h.mainLayout != nil && h.mainLayout.searchEntry != nil {
		h.mainLayout.searchEntry.SetText(query)
		return nil
	}
	return fmt.Errorf("search entry is not initialized")
}

func (h *workspaceNavigationHandler) ResetToSaved() {
	if h.workspace != nil {
		h.workspace.ResetToLocalSaved()
	}
}

func (h *workspaceNavigationHandler) OnSidebarStateChanged() {
	if h.mainLayout.sidebarState.ChatMode && !h.mainLayout.headerState.ChatMode {
		h.workspace.UpdateContent("chats")
		if chatsUI := h.workspace.GetChatsUI(); chatsUI != nil {
			h.mainLayout.sidebarState.ChatUI = chatsUI
			h.mainLayout.setChatMode(true, "Chats", nil)
		} else {
			if chatsUI := h.workspace.GetChatsUI(); chatsUI != nil {
				h.mainLayout.sidebarState.ChatUI = chatsUI
				h.mainLayout.setChatMode(true, "Chats", nil)
			}
		}
		return
	}

	// Если переключились с чатов на другую вкладку
	if !h.mainLayout.sidebarState.ChatMode && h.mainLayout.headerState.ChatMode {
		h.mainLayout.setChatMode(false, "", nil)
		return
	}

	h.mainLayout.rebuildAll()
}
