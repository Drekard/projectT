package workspace

import (
	"image/color"
	"projectT/internal/services"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/workspace/chats"
	"projectT/internal/ui/workspace/compilation"
	"projectT/internal/ui/workspace/contacts"
	"projectT/internal/ui/workspace/p2p"
	"projectT/internal/ui/workspace/profile"
	"projectT/internal/ui/workspace/saved"
	"projectT/internal/ui/workspace/tags"

	p2p_network "projectT/internal/services/p2p/core"
	p2p_ui "projectT/internal/services/p2p/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// itemsService - глобальный экземпляр сервиса элементов
var itemsService = services.NewItemsService()

// ContentType определяет тип отображаемого контента
type ContentType string

const (
	ContentTypeSaved         ContentType = "saved"
	ContentTypePreview       ContentType = "preview"
	ContentTypeProfile       ContentType = "profile"
	ContentTypeTags          ContentType = "tags"
	ContentTypeChats         ContentType = "chats"
	ContentTypeContacts      ContentType = "contacts"
	ContentTypeP2P           ContentType = "p2p"
	ContentTypeRemoteProfile ContentType = "remote_profile"
	ContentTypeRemoteSaved   ContentType = "remote_saved"
	ContentTypeCompilation   ContentType = "compilation"
)

// PreviewMode определяет режим отображения элементов
type PreviewMode string

const (
	PreviewModeSaved   PreviewMode = "saved"
	PreviewModePreview PreviewMode = "preview"
)

// NavigationHandler интерфейс для обработки навигации
type NavigationHandler interface {
	NavigateToFolder(folderID int) error
	UpdateContent(contentType string, param ...interface{})
	SearchByTag(tagName string) error
	SetSearchQuery(query string) error
	ApplyFilters(options services.FilterOptions)
}

// Workspace управляет рабочей областью
type Workspace struct {
	container           *fyne.Container
	gridManager         *saved.GridManager
	currentType         ContentType
	currentPreviewMode  PreviewMode
	contentCache        map[ContentType]fyne.CanvasObject
	navigationManager   *NavigationManager
	profileUI           *profile.UI
	remoteProfileUI     *profile.RemoteProfileUI
	tagsUI              *tags.UI
	chatsUI             *chats.UI
	contactsUI          *contacts.UI
	p2pUI               *p2p.UI
	compilationUI       *compilation.CompilationUI
	window              fyne.Window
	p2pNetwork          *p2p_network.P2PNetwork
	p2pUIShared         *p2p_ui.UIP2P // Общий экземпляр UIP2P для всех вкладок
	tagsInitialized     bool
	chatsInitialized    bool
	contactsInitialized bool
	p2pInitialized      bool
	background          *ScaledBackground
	backgroundRect      *canvas.Rectangle
	showMode            string
	remoteProfilePeerID string
	remoteFolderUUID    string
	remoteFolderTitle   string
	remoteProfileName   string
	remoteFolderPath    []*models.Item
	onRemoteModeChanged func(isRemote bool, peerID, peerName string, path []*models.Item)
}

// CreateWorkspace создает и возвращает рабочую область
func CreateWorkspace(window fyne.Window, p2pNetwork *p2p_network.P2PNetwork) *Workspace {
	ws := &Workspace{
		container:          container.NewStack(),
		currentType:        ContentTypeSaved,
		currentPreviewMode: PreviewModeSaved,
		contentCache:       make(map[ContentType]fyne.CanvasObject),
		window:             window,
		p2pNetwork:         p2pNetwork,
	}

	ws.profileUI = profile.New()
	ws.tagsUI = nil
	ws.chatsUI = nil

	ws.profileUI.SetWindow(window)
	ws.profileUI.SetBackgroundUpdater(ws)

	ws.gridManager = saved.NewGridManager()
	ws.navigationManager = NewNavigationManager()
	ws.gridManager.SetNavigationHandler(ws)

	ws.backgroundRect = canvas.NewRectangle(color.Black)
	ws.loadBackground()
	ws.loadSavedContent()

	return ws
}

// UpdateContent обновляет содержимое рабочей области
func (ws *Workspace) UpdateContent(contentType string, param ...interface{}) {
	ct := ContentType(contentType)
	ws.currentType = ct

	var extraParam interface{}
	if len(param) > 0 {
		extraParam = param[0]
	}

	switch ct {
	case ContentTypeTags:
		ws.initializeTagsUI()
		ws.tagsUI.Refresh()
		ws.contentCache[ct] = ws.createTagsContent()
		ws.container.Objects = []fyne.CanvasObject{ws.contentCache[ct]}
		ws.container.Refresh()
		return
	case ContentTypeChats:
		ws.initializeChatsUI()
		// Используем кэш если контент уже создан
		if cached, exists := ws.contentCache[ct]; exists {
			ws.container.Objects = []fyne.CanvasObject{cached}
			ws.container.Refresh()
			// Обновляем список чатов асинхронно
			go func() {
				if ws.chatsUI != nil {
					ws.chatsUI.Refresh()
				}
			}()
		} else {
			// При первом создании контент уже загружается асинхронно в initializeChatsUI
			ws.contentCache[ct] = ws.createChatsContent()
			ws.container.Objects = []fyne.CanvasObject{ws.contentCache[ct]}
			ws.container.Refresh()
		}
		return
	case ContentTypeContacts:
		ws.initializeContactsUI()
		// Используем кэш если контент уже создан
		if cached, exists := ws.contentCache[ct]; exists {
			ws.container.Objects = []fyne.CanvasObject{cached}
			ws.container.Refresh()
		} else {
			ws.contactsUI.Refresh()
			ws.contentCache[ct] = ws.createContactsContent()
			ws.container.Objects = []fyne.CanvasObject{ws.contentCache[ct]}
			ws.container.Refresh()
		}
		return
	case ContentTypeP2P:
		ws.initializeP2PUI()
		// Используем кэш если контент уже создан
		if cached, exists := ws.contentCache[ct]; exists {
			ws.container.Objects = []fyne.CanvasObject{cached}
			ws.container.Refresh()
			// Обновляем P2P данные асинхронно
			go func() {
				if ws.p2pUI != nil {
					ws.p2pUI.Refresh()
				}
			}()
		} else {
			// При первом создании настройки загружаются асинхронно в SetP2PService
			ws.contentCache[ct] = ws.createP2PContent()
			ws.container.Objects = []fyne.CanvasObject{ws.contentCache[ct]}
			ws.container.Refresh()
		}
		return
	case ContentTypeCompilation:
		ws.contentCache[ct] = ws.createCompilationContent()
		ws.container.Objects = []fyne.CanvasObject{ws.contentCache[ct]}
		ws.container.Refresh()
		return
	case ContentTypeRemoteProfile:
		ws.contentCache[ct] = ws.createRemoteProfileContent()
		ws.container.Objects = []fyne.CanvasObject{ws.contentCache[ct]}
		ws.container.Refresh()
		return
	case ContentTypeRemoteSaved:
		ws.contentCache[ct] = ws.createRemoteSavedContent()
		ws.container.Objects = []fyne.CanvasObject{ws.contentCache[ct]}
		ws.container.Refresh()
		return
	case ContentTypeSaved, ContentTypePreview:
		var newContent fyne.CanvasObject
		switch ct {
		case ContentTypeSaved:
			ws.currentPreviewMode = PreviewModeSaved
			if ws.gridManager != nil {
				ws.gridManager.SetItemMode("saved")
			}
			if extraParam != nil {
				if folderID, ok := extraParam.(int); ok {
					_ = ws.NavigateToFolder(folderID)
				}
			}
			newContent = ws.createSavedContent()
		case ContentTypePreview:
			ws.currentPreviewMode = PreviewModePreview
			if ws.gridManager != nil {
				ws.gridManager.SetItemMode("preview")
			}
			if extraParam != nil {
				if folderID, ok := extraParam.(int); ok {
					_ = ws.NavigateToPreviewFolder(folderID)
				}
			}
			newContent = ws.createPreviewContent()
		}
		if ws.contentCache != nil {
			ws.contentCache[ct] = newContent
		}
		if ws.container != nil {
			ws.container.Objects = []fyne.CanvasObject{newContent}
			ws.container.Refresh()
		}
		return
	default:
		if content, exists := ws.contentCache[ct]; exists && extraParam == nil {
			ws.container.Objects = []fyne.CanvasObject{content}
			ws.container.Refresh()
			return
		}
	}

	var newContent fyne.CanvasObject
	switch ct {
	case ContentTypeProfile:
		newContent = ws.createProfileContent()
	default:
		newContent = ws.createSavedContent()
	}

	ws.contentCache[ct] = newContent
	ws.container.Objects = []fyne.CanvasObject{newContent}
	ws.container.Refresh()
}

// GetContainer возвращает контейнер рабочей области с учетом фона
func (ws *Workspace) GetContainer() *fyne.Container {
	if ws.background != nil {
		return container.NewStack(ws.background, ws.container)
	}
	return container.NewStack(ws.backgroundRect, ws.container)
}

// GetGridManager возвращает менеджер сетки
func (ws *Workspace) GetGridManager() *saved.GridManager {
	return ws.gridManager
}

// GetNavigationManager возвращает менеджер навигации
func (ws *Workspace) GetNavigationManager() *NavigationManager {
	return ws.navigationManager
}

// loadBackground загружает фоновое изображение из профиля
func (ws *Workspace) loadBackground() {
	profile, err := queries.GetLocalProfile()
	if err == nil && profile.BackgroundPath != "" {
		ws.background = NewScaledBackground(profile.BackgroundPath)
	} else {
		ws.background = nil
	}
}

// SetBackgroundColor устанавливает цвет фона рабочей области
func (ws *Workspace) SetBackgroundColor(c color.Color) {
	ws.backgroundRect.FillColor = c
	ws.backgroundRect.Refresh()
}
