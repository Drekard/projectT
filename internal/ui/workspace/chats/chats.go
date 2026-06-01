package chats

import (
	"projectT/internal/config"
	"projectT/internal/controllers"
	network "projectT/internal/services/p2p/ui"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/workspace/chats/center"
	"projectT/internal/ui/workspace/chats/dialogs"
	"projectT/internal/ui/workspace/chats/right"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// UI представляет интерфейс чатов
type UI struct {
	content        fyne.CanvasObject
	window         fyne.Window
	p2pUI          *network.UIP2P
	chatController *controllers.ChatController
	currentChatID  int
	currentContact *models.Contact
	chatPanel      *center.ChatPanel

	// Компоненты панелей
	rightPanel    *right.Panel
	rightVisible  bool
	chatAreaObj   *fyne.Container
	mainContainer *fyne.Container

	// Менеджеры
	chatMenuManager *dialogs.ChatMenuManager

	// Callback для открытия remote профиля
	onOpenRemoteProfile func(peerID string)
	// Callback для открытия папки из чата
	onOpenFolderFromChat func(peerID, folderUUID string)
	// Callback для переключения режима чата
	onChatModeChanged func(isChatMode bool, chatName string, onBack, onOpenProfile, onAttach, onToggleRight, onProfileClicked func())

	// Конфиг и сохранение
	config *config.Config
	onSave func()
}

// SetOnOpenRemoteProfile устанавливает callback для открытия remote профиля
func (ui *UI) SetOnOpenRemoteProfile(callback func(peerID string)) {
	ui.onOpenRemoteProfile = callback
}

// GetCurrentContactPeerID возвращает peerID текущего открытого контакта
func (ui *UI) GetCurrentContactPeerID() string {
	if ui.currentContact != nil {
		return ui.currentContact.PeerID
	}
	// Если контакт не выбран, берём из правой панели (локальный профиль)
	if ui.rightPanel != nil && ui.rightPanel.GetCurrentContactPeerID() != "" {
		return ui.rightPanel.GetCurrentContactPeerID()
	}
	return ""
}

// IsCurrentChatLocal возвращает true, если текущий чат локальный
func (ui *UI) IsCurrentChatLocal() bool {
	return ui.currentContact != nil && ui.currentContact.IsLocalChat()
}

// OnBackToNormalMode переключает из режима чата в нормальный режим
func (ui *UI) OnBackToNormalMode() {
	if ui.onChatModeChanged != nil {
		ui.onChatModeChanged(false, "", nil, nil, nil, nil, nil)
	}
}

func (ui *UI) OpenRemoteProfile(peerID string) {
	if ui.onOpenRemoteProfile != nil {
		ui.onOpenRemoteProfile(peerID)
	}
}

// SetOnOpenFolderFromChat устанавливает callback для открытия папки из чата
func (ui *UI) SetOnOpenFolderFromChat(callback func(peerID, folderUUID string)) {
	ui.onOpenFolderFromChat = callback
}

// SetOnChatModeChanged устанавливает callback для переключения режима чата
func (ui *UI) SetOnChatModeChanged(callback func(isChatMode bool, chatName string, onBack, onOpenProfile, onAttach, onToggleRight, onProfileClicked func())) {
	ui.onChatModeChanged = callback
}

// SetConfig устанавливает конфиг для сохранения настроек
func (ui *UI) SetConfig(cfg *config.Config) {
	ui.config = cfg
}

// SetOnSave устанавливает callback для сохранения конфига
func (ui *UI) SetOnSave(onSave func()) {
	ui.onSave = onSave
}

// RestoreRightPanelState восстанавливает состояние правой панели из конфига
func (ui *UI) RestoreRightPanelState() {
	if ui.config != nil && ui.config.UISettings.RightPanelVisible {
		ui.rightVisible = true
		if ui.rightPanel != nil && ui.rightPanel.Container() != nil {
			ui.rightPanel.Container().Show()
			if ui.mainContainer != nil {
				ui.mainContainer.Refresh()
			}
		}
	}
}

// New создает и возвращает новый UI чатов
func New() *UI {
	ui := &UI{
		rightVisible: false,
	}
	ui.content = ui.createViewContent()
	return ui
}

// SetWindow устанавливает окно
func (ui *UI) SetWindow(window fyne.Window) {
	ui.window = window
}

// createViewContent создает основное представление UI чатов
func (ui *UI) createViewContent() fyne.CanvasObject {
	// Центральная область с чатом (пустая по умолчанию)
	ui.chatAreaObj = ui.createChatArea()

	// Правая панель с профилем (скрыта по умолчанию)
	ui.rightPanel = right.New(ui)
	rightPanelContainer := ui.rightPanel.Container()
	rightPanelContainer.Hide()

	// Основная компоновка: чат | правая панель
	ui.mainContainer = container.NewBorder(
		nil, nil,
		nil,
		rightPanelContainer,
		ui.chatAreaObj,
	)

	return ui.mainContainer
}

// Refresh обновляет UI
func (ui *UI) Refresh() {
	if ui.content != nil {
		ui.content.Refresh()
	}

	// Обновляем правую панель
	if ui.rightPanel != nil {
		ui.rightPanel.Refresh()
	}
}

// CreateView возвращает canvas object для UI чатов
func (ui *UI) CreateView() fyne.CanvasObject {
	return ui.content
}

// SetP2PService устанавливает P2P сервис и создаёт контроллер
func (ui *UI) SetP2PService(p2pUI *network.UIP2P) {
	ui.p2pUI = p2pUI

	// Создаём контроллер чатов
	ui.chatController = controllers.NewChatController()
	ui.chatController.SetP2PService(p2pUI)

	// Настраиваем callback-функции контроллера
	ui.setupControllerCallbacks()
}

// setupControllerCallbacks настраивает callback-функции контроллера для связи с UI
func (ui *UI) setupControllerCallbacks() {
	if ui.chatController == nil {
		return
	}

	// Callback при открытии чата
	ui.chatController.SetOnChatOpened(func(contact *models.Contact) {
		// Обновляем текущий контакт
		ui.currentContact = contact
		ui.currentChatID = contact.ID

		// Создаём панель чата
		chatPanel := ui.createChatPanel(contact)
		ui.chatAreaObj.Objects = []fyne.CanvasObject{chatPanel}
		ui.chatAreaObj.Refresh()

		// Обновляем профиль
		if ui.rightPanel != nil {
			ui.rightPanel.UpdateProfile(contact)
		}

		// Переключаем шапку в режим чата
		chatName := contact.Username
		if chatName == "" {
			profile, err := queries.GetProfileByPeerID(contact.PeerID)
			if err == nil && profile != nil && profile.Username != "" {
				chatName = profile.Username
			} else {
				chatName = contact.PeerID
			}
		}
		// Callback для кнопки профиля в шапке - открывает remote профиль собеседника
		onProfileClicked := func() {
			if contact.PeerID != "" && !contact.IsLocalChat() {
				ui.OnBackToNormalMode()
				ui.OpenRemoteProfile(contact.PeerID)
			}
		}
		ui.notifyChatModeChanged(true, chatName,
			ui.OnBackToNormalMode,
			nil, // onOpenProfile - not used in chat mode
			nil, // onAttach - not wired yet
			ui.ToggleRightPanel,
			onProfileClicked,
		)
	})

	// Callback при закрытии чата
	ui.chatController.SetOnChatClosed(func() {
		// Показываем пустую панель
		emptyPanel := ui.createEmptyPanel()
		ui.chatAreaObj.Objects = []fyne.CanvasObject{emptyPanel}
		ui.chatAreaObj.Refresh()

		// Возвращаем шапку в нормальный режим
		ui.notifyChatModeChanged(false, "", nil, nil, nil, nil, nil)
	})

	// Callback при обновлении контактов
	ui.chatController.SetOnContactsRefreshed(func() {
		// Список чатов теперь в sidebar, уведомляем workspace
	})

	// Callback при получении сообщения
	ui.chatController.SetOnMessageReceived(func(event *controllers.ChatMessageEvent) {
		chatIsOpen := false
		if ui.currentContact != nil {
			if ui.currentContact.ID == event.ContactID {
				chatIsOpen = true
			} else if ui.currentContact.ID == 0 && event.ContactID == 0 {
				if ui.currentContact.PeerID == event.Message.FromPeerID {
					chatIsOpen = true
				}
			}
		}

		if chatIsOpen {
			if ui.chatPanel != nil {
				ui.chatPanel.AddMessage(event.Message, event.IsOutgoing)
			}
		}
	})

	// Callback при загрузке закреплённых элементов пира
	ui.chatController.SetOnPinnedElementsLoaded(func(peerID string) {
		if ui.rightPanel != nil {
			ui.rightPanel.RefreshDemoElementsAfterSync(peerID)
		}
	})
}

// notifyChatModeChanged уведомляет о смене режима чата
func (ui *UI) notifyChatModeChanged(isChatMode bool, chatName string, onBack, onOpenProfile, onAttach, onToggleRight, onProfileClicked func()) {
	if ui.onChatModeChanged != nil {
		ui.onChatModeChanged(isChatMode, chatName, onBack, onOpenProfile, onAttach, onToggleRight, onProfileClicked)
	}
}

// selectChat выбирает чат с пиром (устарел, использовать через контроллер)
func (ui *UI) selectChat(contact *models.Contact) {
	ui.currentContact = contact
	ui.currentChatID = contact.ID

	chatPanel := ui.createChatPanel(contact)
	ui.chatAreaObj.Objects = []fyne.CanvasObject{chatPanel}
	ui.chatAreaObj.Refresh()

	if ui.rightPanel != nil {
		ui.rightPanel.UpdateProfile(contact)
	}
}

// OpenPeerChat открывает чат с пиром (публичный метод для вызова из p2p_panel)
func (ui *UI) OpenPeerChat(peerID, username string) {
	if ui.chatController != nil {
		_ = ui.chatController.OpenPeerChat(peerID, username)
	} else {
		ui.openPeerChatLegacy(peerID, username)
	}
}

// openPeerChatLegacy открывает чат по-старому (для обратной совместимости)
func (ui *UI) openPeerChatLegacy(peerID, username string) {
	profile, err := queries.GetProfileByPeerID(peerID)
	if err == nil && profile != nil {
		username = profile.Username
	}

	contactUsername := username
	contactAvatarPath := ""
	if profile != nil {
		contactUsername = profile.Username
		contactAvatarPath = profile.AvatarPath
	}

	tempContact := &models.Contact{
		PeerID:     peerID,
		Username:   contactUsername,
		AvatarPath: contactAvatarPath,
		ID:         0,
	}

	ui.selectChat(tempContact)
	ui.loadMessagesForPeer(peerID)

	if ui.p2pUI != nil {
		go func() {
			_ = ui.p2pUI.RequestProfile(peerID)
		}()
	}

	if ui.rightPanel != nil {
		ui.rightPanel.UpdateProfile(tempContact)
	}

	if ui.chatAreaObj != nil {
		ui.chatAreaObj.Refresh()
	}
}

// SubscribeToMessages подписывается на события сообщений (устарел, контроллер сам подписывается)
func (ui *UI) SubscribeToMessages() {
}

// createChatArea создает центральную область чата
func (ui *UI) createChatArea() *fyne.Container {
	emptyPanel := ui.createEmptyPanel()
	ui.chatAreaObj = container.NewStack(emptyPanel)
	return ui.chatAreaObj
}

// createEmptyPanel создает пустую панель с подсказкой
func (ui *UI) createEmptyPanel() *fyne.Container {
	icon := fyne.CurrentApp().Icon()

	title := widget.NewLabel("Chats")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	subtitle := widget.NewLabel("Select a chat from the sidebar")
	subtitle.Alignment = fyne.TextAlignCenter
	subtitle.TextStyle = fyne.TextStyle{Italic: true}

	content := container.NewVBox(
		widget.NewIcon(icon),
		title,
		subtitle,
	)

	centered := container.NewCenter(content)

	return centered
}

// createChatPanel создает панель чата с сообщениями и полем ввода
func (ui *UI) createChatPanel(contact *models.Contact) fyne.CanvasObject {
	localPeerID := ""
	if ui.p2pUI != nil {
		status := ui.p2pUI.GetStatus()
		if status != nil && status.PeerID != "" {
			localPeerID = status.PeerID
		}
	}

	if localPeerID == "" {
		localProfile, err := queries.GetLocalProfile()
		if err == nil && localProfile != nil {
			localPeerID = localProfile.PeerID
		}
	}

	ui.chatPanel = center.NewChatPanel(
		contact,
		ui.sendMessage,
		ui.closeChat,
		localPeerID,
		ui.openFolderFromChat,
	)

	return ui.chatPanel.Container()
}

// sendMessage отправляет сообщение (вызывается из chatPanel)
func (ui *UI) sendMessage() {
	if ui.chatPanel == nil {
		return
	}

	text := ui.chatPanel.MessageInput().Text()
	if text == "" {
		return
	}

	ui.chatPanel.MessageInput().Clear()

	if ui.chatController != nil {
		_ = ui.chatController.SendMessage(text)
	}
}

// loadMessagesForPeer загружает сообщения для пира по peer_id
func (ui *UI) loadMessagesForPeer(peerID string) {
	if ui.chatPanel == nil {
		return
	}

	ui.chatPanel.Clear()

	messages, err := ui.chatController.LoadMessages()
	if err != nil {
		return
	}

	ui.chatPanel.LoadMessages(messages, ui.chatController.GetLocalPeerID())
}

// closeChat закрывает текущий чат (вызывается из chatPanel)
func (ui *UI) closeChat() {
	if ui.chatController != nil {
		_ = ui.chatController.CloseChat()
	}
}

// ShowChatMenu показывает меню действий для чата
func (ui *UI) ShowChatMenu(chatID int, peerID string, username string, cont fyne.CanvasObject) {
	if ui.chatMenuManager == nil {
		ui.chatMenuManager = dialogs.NewChatMenuManager(ui, ui.OnChatDeleted)
	}
	ui.chatMenuManager.ShowChatMenu(chatID, peerID, username, cont)
}

// OnChatDeleted обработчик удаления чата
func (ui *UI) OnChatDeleted(chatID int, peerID string) {
	if ui.chatController != nil {
		_ = ui.chatController.DeleteContact(chatID, peerID)
	}
}

// OpenLocalChat открывает локальный чат с самим собой
func (ui *UI) OpenLocalChat() {
	if ui.chatController != nil {
		_ = ui.chatController.OpenLocalChat()
	} else {
		ui.openLocalChatLegacy()
	}
}

// openLocalChatLegacy открывает локальный чат по-старому (для обратной совместимости)
func (ui *UI) openLocalChatLegacy() {
	localProfile, err := queries.GetLocalProfile()
	if err != nil {
		return
	}

	localChat, err := queries.GetOrCreateLocalChat(localProfile.PeerID)
	if err != nil {
		return
	}

	localContact := models.NewLocalContact(
		localProfile.Username,
		localProfile.Title,
		localProfile.AvatarPath,
	)

	ui.selectChat(localContact)
	ui.loadMessagesForChat(localChat.ID)

	if ui.rightPanel != nil {
		ui.rightPanel.UpdateProfile(localContact)
	}

	if ui.chatAreaObj != nil {
		ui.chatAreaObj.Refresh()
	}
}

// loadMessagesForChat загружает сообщения для чата по ID
func (ui *UI) loadMessagesForChat(chatID int) {
	if ui.chatPanel == nil {
		return
	}

	ui.chatPanel.Clear()

	messages, err := ui.chatController.LoadMessagesForChat(chatID)
	if err != nil {
		return
	}

	ui.chatPanel.LoadMessages(messages, ui.chatController.GetLocalPeerID())
}

// RefreshContactsList обновляет список чатов
func (ui *UI) RefreshContactsList() {
	// Список чатов теперь в sidebar
}

// ToggleRightPanel переключает видимость правой панели
func (ui *UI) ToggleRightPanel() {
	ui.rightVisible = !ui.rightVisible

	if ui.rightPanel != nil {
		if ui.rightVisible {
			ui.rightPanel.Container().Show()
		} else {
			ui.rightPanel.Container().Hide()
		}
	}

	// Сохраняем состояние правой панели в конфиг
	if ui.config != nil {
		ui.config.UISettings.RightPanelVisible = ui.rightVisible
		if ui.onSave != nil {
			ui.onSave()
		}
	}

	if ui.mainContainer != nil {
		ui.mainContainer.Refresh()
	}
}

// RefreshRightPanel обновляет правую панель с профилем пира
func (ui *UI) RefreshRightPanel(peerID string) {
	fyne.Do(func() {
		if ui.chatController != nil {
			contact, err := ui.chatController.GetContactByPeerID(peerID)
			if err == nil && contact != nil {
				if ui.rightPanel != nil {
					ui.rightPanel.UpdateProfile(contact)
				}
				return
			}
		}

		profile, err := queries.GetProfileByPeerID(peerID)
		if err != nil || profile == nil {
			return
		}

		contact := &models.Contact{
			PeerID:     peerID,
			Username:   profile.Username,
			Title:      profile.Title,
			AvatarPath: profile.AvatarPath,
		}

		if ui.rightPanel != nil {
			ui.rightPanel.UpdateProfile(contact)
		}
	})
}

// RefreshDemoElementsAfterSync обновляет витрину элементов правой панели после синхронизации элементов
func (ui *UI) RefreshDemoElementsAfterSync(peerID string) {
	fyne.Do(func() {
		if ui.rightPanel != nil {
			ui.rightPanel.RefreshDemoElementsAfterSync(peerID)
		}
	})
}

// GetP2PService возвращает P2P сервис
func (ui *UI) GetP2PService() *network.UIP2P {
	return ui.p2pUI
}

// GetWindow возвращает окно
func (ui *UI) GetWindow() fyne.Window {
	return ui.window
}

// openFolderFromChat открывает папку из чата — запрашивает у пира и переключает workspace
func (ui *UI) openFolderFromChat(folderUUID, peerID string) {
	if ui.onOpenFolderFromChat != nil {
		ui.onOpenFolderFromChat(peerID, folderUUID)
	}
}
