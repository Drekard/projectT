package workspace

import (
	"context"
	"fmt"
	"image/color"
	"log"
	"projectT/internal/services"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/workspace/chats"
	"projectT/internal/ui/workspace/contacts"
	"projectT/internal/ui/workspace/p2p"
	"projectT/internal/ui/workspace/profile"
	"projectT/internal/ui/workspace/saved"
	"projectT/internal/ui/workspace/saved/sorting"
	"projectT/internal/ui/workspace/tags"

	"github.com/libp2p/go-libp2p/core/peer"
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
)

// PreviewMode определяет режим отображения элементов
type PreviewMode string

const (
	PreviewModeSaved   PreviewMode = "saved"   // Показывать только saved элементы
	PreviewModePreview PreviewMode = "preview" // Показывать только preview элементы
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
	container          *fyne.Container
	gridManager        *saved.GridManager // Единый менеджер сетки
	currentType        ContentType
	currentPreviewMode PreviewMode // Текущий режим (saved/preview)
	contentCache       map[ContentType]fyne.CanvasObject
	navigationManager  *NavigationManager // Менеджер навигации
	profileUI          *profile.UI
	remoteProfileUI    *profile.RemoteProfileUI // Профиль удалённого пользователя
	tagsUI             *tags.UI
	chatsUI            *chats.UI
	contactsUI         *contacts.UI
	p2pUI              *p2p.UI
	window             fyne.Window
	p2pNetwork         *p2p_network.P2PNetwork // P2P сеть
	// Флаги для отслеживания, были ли UI-компоненты инициализированы
	tagsInitialized     bool
	chatsInitialized    bool
	contactsInitialized bool
	p2pInitialized      bool
	// Фоновое изображение
	background     *ScaledBackground // кастомный фон с масштабированием
	backgroundRect *canvas.Rectangle // прямоугольник фона по умолчанию
	// Режим отображения элементов
	showMode string // "current_folder" или "all_items"
	// Remote profile данные
	remoteProfilePeerID string
	remoteFolderUUID    string
	remoteFolderTitle   string
	remoteProfileName   string
	remoteFolderPath    []*models.Item // Путь по папкам для breadcrumbs
	// Callback для обновления breadcrumbs при смене remote режима
	onRemoteModeChanged func(isRemote bool, peerID, peerName string, path []*models.Item)
}

// CreateWorkspace создает и возвращает рабочую область
func CreateWorkspace(window fyne.Window, p2pNetwork *p2p_network.P2PNetwork) *Workspace {
	ws := &Workspace{
		container:          container.NewStack(),
		currentType:        ContentTypeSaved,
		currentPreviewMode: PreviewModeSaved, // По умолчанию показываем saved
		contentCache:       make(map[ContentType]fyne.CanvasObject),
		window:             window,
		p2pNetwork:         p2pNetwork,
	}

	// Инициализируем UI компоненты
	ws.profileUI = profile.New()
	// Не инициализируем tagsUI сразу - ленивая загрузка
	ws.tagsUI = nil
	// Не инициализируем chatsUI сразу - ленивая загрузка
	ws.chatsUI = nil

	// Устанавливаем окно для profile UI
	ws.profileUI.SetWindow(window)

	// Устанавливаем ссылку на workspace для обновления фона
	ws.profileUI.SetBackgroundUpdater(ws)

	// Инициализируем единый GridManager
	ws.gridManager = saved.NewGridManager()

	// Инициализируем NavigationManager
	ws.navigationManager = NewNavigationManager()

	// Устанавливаем навигацию для GridManager
	ws.gridManager.SetNavigationHandler(ws)

	// Создаем прямоугольник фона по умолчанию
	ws.backgroundRect = canvas.NewRectangle(color.Black)

	// Загружаем фоновое изображение из профиля
	ws.loadBackground()

	// Загружаем начальный контент (сохраненное)
	ws.loadSavedContent()

	return ws
}

// UpdateContent обновляет содержимое рабочей области
func (ws *Workspace) UpdateContent(contentType string, param ...interface{}) {
	ct := ContentType(contentType)
	ws.currentType = ct

	// Проверяем, есть ли дополнительные параметры для фильтрации
	var extraParam interface{}
	if len(param) > 0 {
		extraParam = param[0]
	}

	// Для вкладок tags и chats всегда инициализируем и обновляем данные
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
		ws.chatsUI.Refresh()
		ws.contentCache[ct] = ws.createChatsContent()
		ws.container.Objects = []fyne.CanvasObject{ws.contentCache[ct]}
		ws.container.Refresh()
		return
	case ContentTypeContacts:
		// Для вкладки "Контакты" всегда инициализируем и обновляем данные
		ws.initializeContactsUI()
		ws.contactsUI.Refresh()
		ws.contentCache[ct] = ws.createContactsContent()
		ws.container.Objects = []fyne.CanvasObject{ws.contentCache[ct]}
		ws.container.Refresh()
		return
	case ContentTypeP2P:
		// Для вкладки "P2P" всегда инициализируем и обновляем данные
		ws.initializeP2PUI()
		ws.p2pUI.Refresh()
		ws.contentCache[ct] = ws.createP2PContent()
		ws.container.Objects = []fyne.CanvasObject{ws.contentCache[ct]}
		ws.container.Refresh()
		return
	case ContentTypeRemoteProfile:
		// Remote profile — всегда пересоздаём
		ws.contentCache[ct] = ws.createRemoteProfileContent()
		ws.container.Objects = []fyne.CanvasObject{ws.contentCache[ct]}
		ws.container.Refresh()
		return
	case ContentTypeRemoteSaved:
		// Remote saved — всегда пересоздаём
		ws.contentCache[ct] = ws.createRemoteSavedContent()
		ws.container.Objects = []fyne.CanvasObject{ws.contentCache[ct]}
		ws.container.Refresh()
		return
	case ContentTypeSaved, ContentTypePreview:
		// Для вкладок saved/preview всегда обновляем данные, игнорируя кэш
		// Это необходимо, т.к. gridManager один на обе вкладки и хранит только последнее состояние
		var newContent fyne.CanvasObject
		switch ct {
		case ContentTypeSaved:
			ws.currentPreviewMode = PreviewModeSaved
			// Устанавливаем режим saved для gridManager
			if ws.gridManager != nil {
				ws.gridManager.SetItemMode("saved")
			}
			if extraParam != nil {
				// Если передан ID папки, переходим к этой папке
				if folderID, ok := extraParam.(int); ok {
					_ = ws.NavigateToFolder(folderID)
				}
			}
			newContent = ws.createSavedContent()
		case ContentTypePreview:
			ws.currentPreviewMode = PreviewModePreview
			// Устанавливаем режим preview для gridManager
			if ws.gridManager != nil {
				ws.gridManager.SetItemMode("preview")
			}
			if extraParam != nil {
				// Если передан ID папки, переходим к этой папке
				if folderID, ok := extraParam.(int); ok {
					_ = ws.NavigateToPreviewFolder(folderID)
				}
			}
			newContent = ws.createPreviewContent()
		}
		// Сохраняем в кэш для консистентности
		if ws.contentCache != nil {
			ws.contentCache[ct] = newContent
		}
		if ws.container != nil {
			ws.container.Objects = []fyne.CanvasObject{newContent}
			ws.container.Refresh()
		}
		return
	default:
		// Проверяем кэш для других типов контента (profile и т.д.)
		if content, exists := ws.contentCache[ct]; exists && extraParam == nil {
			ws.container.Objects = []fyne.CanvasObject{content}
			ws.container.Refresh()
			return
		}
	}

	// Создаем новый контент
	var newContent fyne.CanvasObject
	switch ct {
	case ContentTypeProfile:
		newContent = ws.createProfileContent()
	default:
		newContent = ws.createSavedContent()
	}

	// Сохраняем в кэш
	ws.contentCache[ct] = newContent
	ws.container.Objects = []fyne.CanvasObject{newContent}
	ws.container.Refresh()
}

// loadSavedContent загружает сохраненные элементы
func (ws *Workspace) loadSavedContent() {
	if ws.gridManager == nil {
		return
	}
	items, err := itemsService.GetSavedItemsByParent(0)
	if err != nil {
		items = []*models.Item{}
	}
	ws.gridManager.LoadItems(items)

	// Устанавливаем корневой элемент как текущий
	ws.gridManager.SetCurrentParentID(0)
}

// loadPreviewContent загружает preview элементы
func (ws *Workspace) loadPreviewContent() {
	items, err := itemsService.GetPreviewItemsByParent(0)
	if err != nil {
		items = []*models.Item{}
	}
	ws.gridManager.LoadItems(items)

	// Устанавливаем корневой элемент как текущий
	ws.gridManager.SetCurrentParentID(0)
}

// NavigateToPreviewFolder переходит в указанную папку для preview элементов
func (ws *Workspace) NavigateToPreviewFolder(folderID int) error {
	err := ws.navigationManager.GoToFolderInPath(folderID)
	if err != nil {
		return err
	}

	// Загружаем элементы текущей папки с учетом настроек сортировки
	currentParentID := ws.navigationManager.GetCurrentFolderID()
	err = ws.gridManager.LoadItemsByParentWithSort(currentParentID)
	if err != nil {
		return err
	}

	// Обновляем текущий тип контента на "папка"
	ws.currentType = ContentType("folder_" + fmt.Sprintf("%d", currentParentID))
	return nil
}

// NavigateToFolder переходит в указанную папку
func (ws *Workspace) NavigateToFolder(folderID int) error {
	// Если мы в remote режиме — переходим на remote навигацию
	if ws.remoteProfilePeerID != "" {
		item, err := queries.GetItemByID(folderID)
		if err == nil && item != nil && item.OwnerType == "remote" {
			ws.NavigateToRemoteFolder(item.ElementUUID)
			return nil
		}
	}

	err := ws.navigationManager.GoToFolderInPath(folderID)
	if err != nil {
		return err
	}

	// Загружаем элементы текущей папки с учетом настроек сортировки
	currentParentID := ws.navigationManager.GetCurrentFolderID()
	err = ws.gridManager.LoadItemsByParentWithSort(currentParentID)
	if err != nil {
		return err
	}

	// Обновляем текущий тип контента на "папка"
	ws.currentType = ContentType("folder_" + fmt.Sprintf("%d", currentParentID))
	return nil
}

// RefreshCurrentFolder обновляет содержимое текущей папки
func (ws *Workspace) RefreshCurrentFolder() error {
	currentParentID := ws.navigationManager.GetCurrentFolderID()
	err := ws.gridManager.LoadItemsByParentWithSort(currentParentID)
	if err != nil {
		return err
	}

	// Обновляем текущий тип контента на "папка"
	ws.currentType = ContentType("folder_" + fmt.Sprintf("%d", currentParentID))
	return nil
}

// SearchByTag выполняет поиск элементов по тегу
func (ws *Workspace) SearchByTag(tagName string) error {
	return ws.SearchItems(tagName)
}

// SetSearchQuery устанавливает значение в поисковую строку
func (ws *Workspace) SetSearchQuery(query string) error {
	// В данной реализации Workspace не имеет прямого доступа к поисковому полю
	// Это будет обрабатываться через main_layout, который содержит ссылку на searchEntry
	return nil
}

// SetupNavigation настраивает навигацию
func (ws *Workspace) SetupNavigation(scroll *container.Scroll) {
	// Настройка навигации - в данном случае просто устанавливаем обработчик скролла
	// Используем существующий метод onSizeChanged из GridManager
	scroll.OnScrolled = func(pos fyne.Position) {
		if ws.gridManager != nil {
			ws.gridManager.UpdateLayout()
		}
	}
}

// OnSizeChanged обработчик изменения размера
func (ws *Workspace) OnSizeChanged(pos interface{}) {
	// Обновляем макет сетки при изменении размера
	if ws.gridManager != nil {
		// Вызываем обновление макета
		go ws.gridManager.UpdateLayout()
	}
}

// SearchItems выполняет поиск элементов по запросу
func (ws *Workspace) SearchItems(query string) error {
	if query == "" {
		// Если запрос пустой, возвращаемся к обычному отображению
		currentParentID := ws.navigationManager.GetCurrentFolderID()
		return ws.gridManager.LoadItemsByParentWithSort(currentParentID)
	}

	// Загружаем элементы поисковому запросу с учетом настроек сортировки
	err := ws.gridManager.LoadItemsBySearchWithSort(query)
	if err != nil {
		return err
	}

	// Обновляем текущий тип контента на "поиск"
	ws.currentType = ContentType("search_" + query)
	return nil
}

// ClearSearch очищает результаты поиска и возвращает к нормальному отображению
func (ws *Workspace) ClearSearch() error {
	currentParentID := ws.navigationManager.GetCurrentFolderID()
	err := ws.gridManager.LoadItemsByParentWithSort(currentParentID)
	if err != nil {
		return err
	}

	// Возвращаемся к нормальному типу контента
	ws.currentType = ContentType("folder_" + fmt.Sprintf("%d", currentParentID))
	return nil
}

// GetGridManager возвращает менеджер сетки
func (ws *Workspace) GetGridManager() *saved.GridManager {
	return ws.gridManager
}

// GetNavigationManager возвращает менеджер навигации
func (ws *Workspace) GetNavigationManager() *NavigationManager {
	return ws.navigationManager
}

// createSavedContent создает контент для "Сохраненного"
func (ws *Workspace) createSavedContent() fyne.CanvasObject {
	if ws.gridManager == nil {
		return nil
	}
	// Загружаем актуальные данные
	ws.loadSavedContent()
	return ws.gridManager.GetContainer()
}

// createPreviewContent создает контент для "Загруженного"
func (ws *Workspace) createPreviewContent() fyne.CanvasObject {
	// Загружаем актуальные данные
	ws.loadPreviewContent()
	return ws.gridManager.GetContainer()
}

// createProfileContent создает контент для профиля
func (ws *Workspace) createProfileContent() fyne.CanvasObject {
	return ws.profileUI.CreateView()
}

// createTagsContent создает контент для тегов
func (ws *Workspace) createTagsContent() fyne.CanvasObject {
	ws.initializeTagsUI()
	// Обновляем содержимое при каждом открытии вкладки
	ws.tagsUI.Refresh()
	return ws.tagsUI.GetContent()
}

// createChatsContent создает контент для чатов
func (ws *Workspace) createChatsContent() fyne.CanvasObject {
	ws.initializeChatsUI()
	// Обновляем содержимое при каждом открытии вкладки (если метод Refresh доступен)
	if ws.chatsUI != nil {
		ws.chatsUI.Refresh()
	}
	return ws.chatsUI.CreateView()
}

// createContactsContent создает контент для вкладки "Контакты"
func (ws *Workspace) createContactsContent() fyne.CanvasObject {
	ws.initializeContactsUI()
	return ws.contactsUI.GetContent()
}

// createP2PContent создает контент для вкладки "P2P"
func (ws *Workspace) createP2PContent() fyne.CanvasObject {
	ws.initializeP2PUI()
	return ws.p2pUI.GetContent()
}

// createRemoteProfileContent создает контент для удалённого профиля
func (ws *Workspace) createRemoteProfileContent() fyne.CanvasObject {
	if ws.remoteProfileUI == nil || ws.remoteProfileUI.GetPeerID() != ws.remoteProfilePeerID {
		var p2pUI *p2p_ui.UIP2P
		if ws.p2pNetwork != nil {
			p2pUI = p2p_ui.NewUIP2P(ws.p2pNetwork)
		}
		ws.remoteProfileUI = profile.NewRemoteProfileUI(ws.remoteProfilePeerID, p2pUI)
		ws.remoteProfileUI.SetOnOpenElements(func() {
			ws.OpenRemoteSaved(ws.remoteProfilePeerID, ws.remoteProfileName, "")
		})
	}
	return ws.remoteProfileUI.Container()
}

// createRemoteSavedContent создает контент для удалённых элементов
func (ws *Workspace) createRemoteSavedContent() fyne.CanvasObject {
	// Загружаем элементы по parent_uuid из локальной БД
	var items []*models.Item
	remoteItems, err := queries.GetRemoteItemsByPeer(ws.remoteProfilePeerID)
	if err == nil {
		if ws.remoteFolderUUID == "" {
			// Корень — элементы с parent_uuid = nil или ""
			for _, item := range remoteItems {
				if item.ParentUUID == nil || *item.ParentUUID == "" {
					items = append(items, item)
				}
			}
		} else {
			// Конкретная папка
			for _, item := range remoteItems {
				if item.ParentUUID != nil && *item.ParentUUID == ws.remoteFolderUUID {
					items = append(items, item)
				}
			}
		}
	}

	ws.gridManager.LoadItemsWithoutCreateElement(items)

	// Если элементов нет в БД — запрашиваем у пира
	if len(items) == 0 && ws.p2pNetwork != nil {
		go ws.requestRemoteFolderFromPeer(ws.remoteProfilePeerID, ws.remoteFolderUUID)
	}

	return ws.gridManager.GetContainer()
}

// requestRemoteFolderFromPeer запрашивает элементы папки у пира и сохраняет в БД
func (ws *Workspace) requestRemoteFolderFromPeer(peerID, folderUUID string) {
	peerIDObj, err := peer.Decode(peerID)
	if err != nil {
		log.Printf("[Workspace] Ошибка декодирования peerID: %v", err)
		return
	}

	ctx := context.Background()
	items, err := ws.p2pNetwork.ItemSync().RequestFolder(ctx, peerIDObj, folderUUID)
	if err != nil {
		log.Printf("[Workspace] Ошибка запроса элементов у пира: %v", err)
		return
	}

	if len(items) == 0 {
		return
	}

	// Сохраняем элементы в БД как remote
	for _, item := range items {
		item.OwnerType = "remote"
		item.SourcePeerID = &peerID
		_ = queries.CreateRemoteItem(item)
	}

	// Обновляем отображение
	ws.gridManager.LoadItemsWithoutCreateElement(items)
}

// OpenRemoteProfile открывает профиль удалённого пользователя
func (ws *Workspace) OpenRemoteProfile(peerID string) {
	ws.remoteProfilePeerID = peerID
	ws.remoteFolderUUID = ""
	ws.remoteFolderTitle = ""
	ws.remoteFolderPath = nil

	// Получаем имя профиля (по peerID без фильтра owner_type)
	profile, err := queries.GetProfileByPeerID(peerID)
	if err == nil && profile != nil {
		ws.remoteProfileName = profile.Username
	}

	ws.notifyRemoteModeChanged(true)
	ws.UpdateContent(string(ContentTypeRemoteProfile))
}

// OpenRemoteSaved открывает элементы удалённого пользователя
func (ws *Workspace) OpenRemoteSaved(peerID, peerName, folderUUID string) {
	ws.remoteProfilePeerID = peerID
	ws.remoteProfileName = peerName
	ws.remoteFolderUUID = folderUUID
	ws.remoteFolderTitle = ""
	ws.remoteFolderPath = nil

	// Если folderUUID указан, получаем название папки
	if folderUUID != "" {
		item, err := queries.GetItemByElementUUID(folderUUID)
		if err == nil && item != nil {
			ws.remoteFolderTitle = item.Title
			ws.remoteFolderPath = []*models.Item{item}
		}
	}

	ws.notifyRemoteModeChanged(true)
	ws.UpdateContent(string(ContentTypeRemoteSaved))
}

// NavigateToRemoteFolder переходит в папку удалённого пользователя
func (ws *Workspace) NavigateToRemoteFolder(folderUUID string) {
	ws.remoteFolderUUID = folderUUID

	// Получаем название папки
	item, err := queries.GetItemByElementUUID(folderUUID)
	if err == nil && item != nil {
		ws.remoteFolderTitle = item.Title
		ws.remoteFolderPath = append(ws.remoteFolderPath, item)
	}

	ws.notifyRemoteModeChanged(true)
	ws.UpdateContent(string(ContentTypeRemoteSaved))
}

// OpenRemoteFolderFromChat открывает папку из чата — запрашивает у пира и переключает на remote saved view
func (ws *Workspace) OpenRemoteFolderFromChat(peerID, folderUUID string) {
	ws.remoteProfilePeerID = peerID
	ws.remoteFolderUUID = folderUUID
	ws.remoteFolderTitle = ""
	ws.remoteFolderPath = nil

	// Получаем имя профиля
	profile, err := queries.GetProfileByPeerID(peerID)
	if err == nil && profile != nil {
		ws.remoteProfileName = profile.Username
	}

	// Получаем название папки из БД если есть
	item, err := queries.GetItemByElementUUID(folderUUID)
	if err == nil && item != nil {
		ws.remoteFolderTitle = item.Title
	}

	// Запускаем запрос папки у пира в фоне
	go ws.requestRemoteFolderFromPeer(peerID, folderUUID)

	ws.notifyRemoteModeChanged(true)
	ws.UpdateContent(string(ContentTypeRemoteSaved))
}

// ResetToLocalSaved сбрасывает remote режим и возвращает к локальной сортировке
func (ws *Workspace) ResetToLocalSaved() {
	ws.remoteProfilePeerID = ""
	ws.remoteFolderUUID = ""
	ws.remoteFolderTitle = ""
	ws.remoteProfileName = ""
	ws.remoteFolderPath = nil
	ws.remoteProfileUI = nil
	ws.navigationManager.Reset()
	ws.notifyRemoteModeChanged(false)
	ws.UpdateContent(string(ContentTypeSaved))
}

// GetRemoteProfilePeerID возвращает peerID текущего remote профиля
func (ws *Workspace) GetRemoteProfilePeerID() string {
	return ws.remoteProfilePeerID
}

// GetRemoteProfileName возвращает имя текущего remote профиля
func (ws *Workspace) GetRemoteProfileName() string {
	return ws.remoteProfileName
}

// GetRemoteFolderUUID возвращает текущий remote folder UUID
func (ws *Workspace) GetRemoteFolderUUID() string {
	return ws.remoteFolderUUID
}

// GetRemoteFolderTitle возвращает название текущей remote папки
func (ws *Workspace) GetRemoteFolderTitle() string {
	return ws.remoteFolderTitle
}

// SetOnRemoteModeChanged устанавливает callback при смене remote режима
func (ws *Workspace) SetOnRemoteModeChanged(callback func(isRemote bool, peerID, peerName string, path []*models.Item)) {
	ws.onRemoteModeChanged = callback
}

func (ws *Workspace) notifyRemoteModeChanged(isRemote bool) {
	if ws.onRemoteModeChanged != nil {
		ws.onRemoteModeChanged(isRemote, ws.remoteProfilePeerID, ws.remoteProfileName, ws.remoteFolderPath)
	}
}

// initializeContactsUI инициализирует UI вкладки "Контакты" при первом обращении
func (ws *Workspace) initializeContactsUI() {
	if !ws.contactsInitialized {
		// Инициализируем chatsUI если еще не инициализирован
		if ws.chatsUI == nil {
			ws.initializeChatsUI()
		}
		ws.contactsUI = contacts.New(ws.chatsUI)
		ws.contactsInitialized = true

		// Устанавливаем окно для contacts UI
		ws.contactsUI.SetWindow(ws.window)

		// Устанавливаем P2P сервис если доступен
		if ws.p2pNetwork != nil {
			p2pUI := p2p_ui.NewUIP2P(ws.p2pNetwork)
			ws.contactsUI.SetP2PService(p2pUI)
		}
	}
}

// initializeP2PUI инициализирует UI вкладки "P2P" при первом обращении
func (ws *Workspace) initializeP2PUI() {
	if !ws.p2pInitialized {
		// Инициализируем chatsUI если еще не инициализирован
		if ws.chatsUI == nil {
			ws.initializeChatsUI()
		}
		// Передаём обёртку, которая игнорирует дополнительные параметры
		ws.p2pUI = p2p.New(ws.chatsUI, func(contentType string) {
			ws.UpdateContent(contentType)
		})
		ws.p2pInitialized = true

		// Устанавливаем окно для p2p UI
		ws.p2pUI.SetWindow(ws.window)

		// Устанавливаем P2P сервис если доступен
		if ws.p2pNetwork != nil {
			p2pUI := p2p_ui.NewUIP2P(ws.p2pNetwork)
			ws.p2pUI.SetP2PService(p2pUI)
		}
	}
}

// initializeTagsUI инициализирует UI тегов при первом обращении
func (ws *Workspace) initializeTagsUI() {
	if !ws.tagsInitialized {
		ws.tagsUI = tags.New()
		ws.tagsInitialized = true
	}
}

// initializeChatsUI инициализирует UI чатов при первом обращении
func (ws *Workspace) initializeChatsUI() {
	if !ws.chatsInitialized {
		ws.chatsUI = chats.New()
		ws.chatsInitialized = true

		// Устанавливаем окно для chats UI
		ws.chatsUI.SetWindow(ws.window)

		// Устанавливаем callback для открытия remote профиля
		ws.chatsUI.SetOnOpenRemoteProfile(func(peerID string) {
			ws.OpenRemoteProfile(peerID)
		})

		// Устанавливаем callback для открытия папки из чата
		ws.chatsUI.SetOnOpenFolderFromChat(func(peerID, folderUUID string) {
			ws.OpenRemoteFolderFromChat(peerID, folderUUID)
		})

		// Устанавливаем P2P сервис если доступен
		if ws.p2pNetwork != nil {
			p2pUI := p2p_ui.NewUIP2P(ws.p2pNetwork)
			// Устанавливаем callback для обновления UI после загрузки профиля
			p2pUI.SetOnProfileUpdated(func(peerID string) {
				// Обновляем правую панель с профилем в UI потоке
				ws.chatsUI.RefreshRightPanel(peerID)
			})
			ws.chatsUI.SetP2PService(p2pUI)

			// Устанавливаем UI API в profile exchange для уведомлений
			if ws.p2pNetwork.ProfileExchange() != nil {
				ws.p2pNetwork.ProfileExchange().SetUIP2P(p2pUI)
				// Устанавливаем callback для обновления витрины элементов после синхронизации
				ws.p2pNetwork.ProfileExchange().SetUIProfilePanel(ws.chatsUI)
			}

			// Устанавливаем глобальный P2P сервис для ChatService
			services.SetGlobalP2PNetwork(ws.p2pNetwork)
		}

		// Подписываемся на события сообщений
		ws.chatsUI.SubscribeToMessages()
	}
}

// loadBackground загружает фоновое изображение из профиля
func (ws *Workspace) loadBackground() {
	profile, err := queries.GetLocalProfile()
	if err == nil && profile.BackgroundPath != "" {
		// Создаем кастомный фон с масштабированием
		ws.background = NewScaledBackground(profile.BackgroundPath)
	} else {
		// Используем стандартный фон (черный прямоугольник)
		ws.background = nil
	}
}

// GetContainer возвращает контейнер рабочей области с учетом фона
func (ws *Workspace) GetContainer() *fyne.Container {
	// Если есть кастомный фон, используем его
	if ws.background != nil {
		return container.NewStack(ws.background, ws.container)
	}
	// Иначе используем стандартный фон
	return container.NewStack(ws.backgroundRect, ws.container)
}

// ApplyFilters применяет фильтры и обновляет сетку элементов
func (ws *Workspace) ApplyFilters(options services.FilterOptions) {
	// Обновляем настройки сортировки
	ws.gridManager.SetSortOptions(&options)

	// Определяем, какой режим отображения использовать
	if options.TabMode == "all_items" {
		// Режим "Все элементы"
		ws.showMode = "all_items"

		// Загружаем элементы в зависимости от текущего режима (saved/preview)
		var allItems []*models.Item
		var err error

		if ws.currentPreviewMode == PreviewModePreview {
			// Загружаем только preview элементы (status='preview')
			allItems, err = itemsService.GetPreviewItemsWithoutParentFilter()
		} else {
			// Загружаем только сохранённые элементы (status='saved')
			allItems, err = itemsService.GetSavedItemsWithoutParentFilter()
		}

		if err != nil {
			// В случае ошибки загружаем пустой список
			allItems = []*models.Item{}
		}

		// Применяем сортировку к полученным элементам
		sortedItems := ws.sortItems(allItems, &options)
		ws.gridManager.LoadItems(sortedItems)
	} else {
		// Режим "Эта папка" - отображаем элементы текущей папки
		ws.showMode = "current_folder"
		currentParentID := ws.navigationManager.GetCurrentFolderID()

		// Загружаем элементы в зависимости от текущего режима (saved/preview)
		var err error
		if ws.currentPreviewMode == PreviewModePreview {
			err = ws.gridManager.LoadItemsByParentWithSort(currentParentID)
		} else {
			err = ws.gridManager.LoadItemsByParentWithSort(currentParentID)
		}

		if err != nil {
			_ = err //nolint:staticcheck // В случае ошибки можно залогировать или обработать по-другому
		}
	}
}

// sortItems сортирует элементы по заданным настройкам
func (ws *Workspace) sortItems(items []*models.Item, options *services.FilterOptions) []*models.Item {
	// Используем сортировщик из пакета sorting
	itemSorter := sorting.NewItemSorter()
	return itemSorter.SortItems(items, options)
}

// SetBackgroundColor устанавливает цвет фона рабочей области
func (ws *Workspace) SetBackgroundColor(c color.Color) {
	ws.backgroundRect.FillColor = c
	ws.backgroundRect.Refresh()
}
