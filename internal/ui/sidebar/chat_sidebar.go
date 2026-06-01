package sidebar

import (
	"image/color"
	"log"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createChatSidebar создает sidebar в режиме чата со списком чатов
func createChatSidebar(state *SidebarState) *fyne.Container {
	backBtn := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		if state.OnBackToNav != nil {
			state.OnBackToNav()
		}
	})
	backBtn.Importance = widget.LowImportance

	// Иконка чата с собой (Избранное)
	favoriteIcon := createChatFavoriteIcon(state.ChatUI)

	// Кнопка обновления
	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		if state.ChatUI != nil {
			state.ChatUI.RefreshContactsList()
		}
	})
	refreshBtn.Importance = widget.LowImportance

	// Верхняя панель с кнопками (статичная)
	topButtons := container.NewVBox(
		backBtn,
		container.NewBorder(nil, nil, nil, nil, favoriteIcon),
		container.NewBorder(nil, nil, nil, nil, refreshBtn),
	)

	// Цвет для разделителей
	grayColor := color.RGBA{R: 50, G: 50, B: 50, A: 255}

	// Список чатов (прокручиваемый, растянут на всю высоту)
	chatsList := container.NewVBox()
	chatsScroll := container.NewVScroll(chatsList)

	// Чёрный фон под списком чатов
	chatsBg := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 255})
	chatsWithBg := container.NewStack(chatsBg, chatsScroll)

	go loadChatsList(chatsList, state.ChatUI)

	// Кнопка + внизу (статичная)
	addBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		showNewChatDialog(state.ChatUI)
	})
	addBtn.Importance = widget.LowImportance

	// Нижний разделитель (серый)
	bottomSeparator := canvas.NewRectangle(grayColor)
	bottomSeparator.SetMinSize(fyne.NewSize(0, 1))

	// Верхний разделитель (серый) над списком чатов
	topSeparator := canvas.NewRectangle(grayColor)
	topSeparator.SetMinSize(fyne.NewSize(0, 1))

	// Вертикальная компоновка: статичные кнопки + разделитель + прокручиваемый список + разделитель + кнопка +
	scrollSection := container.NewBorder(topSeparator, nil, nil, nil, chatsWithBg)
	content := container.NewBorder(topButtons, container.NewVBox(bottomSeparator, addBtn), nil, nil, scrollSection)

	sidebarContainer := container.NewStack(content)
	sidebarContainer.Resize(fyne.NewSize(state.Width, 0))

	return sidebarContainer
}

// showNewChatDialog показывает диалог для создания нового чата/группы
func showNewChatDialog(chatUI ChatUIProvider) {
	if chatUI == nil {
		return
	}

	// Поле для ввода инвайта
	inviteEntry := widget.NewEntry()
	inviteEntry.SetPlaceHolder("Enter group/channel invite token")

	// Заголовок секции инвайта
	inviteLabel := widget.NewLabel("Join Group/Channel")
	inviteLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Кнопка подключения по инвайту
	joinBtn := widget.NewButton("Join", func() {
		token := inviteEntry.Text
		if token == "" {
			return
		}
		if p2pUI := chatUI.GetP2PService(); p2pUI != nil {
			if network := p2pUI.GetNetwork(); network != nil {
				if groupChat := network.GroupChat(); groupChat != nil {
					_, err := groupChat.JoinGroupViaInvite(token)
					if err != nil {
						dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
					} else {
						dialog.ShowInformation("Success", "Joined group/channel", fyne.CurrentApp().Driver().AllWindows()[0])
						chatUI.RefreshContactsList()
					}
				}
			}
		}
	})
	joinBtn.Importance = widget.HighImportance

	// Список подключенных людей для создания чата
	contactsLabel := widget.NewLabel("Start chat with:")
	contactsLabel.TextStyle = fyne.TextStyle{Bold: true}

	contactsList := container.NewVBox()
	if p2pUI := chatUI.GetP2PService(); p2pUI != nil {
		connectedPeers := p2pUI.GetConnectedPeers()
		for _, peer := range connectedPeers {
			username := peer.Username
			if username == "" {
				profile, err := queries.GetProfileByPeerID(peer.PeerID)
				if err == nil && profile != nil {
					username = profile.Username
				}
				if username == "" {
					username = peer.PeerID[:min(8, len(peer.PeerID))]
				}
			}
			peerBtn := widget.NewButton(username, func() {
				chatUI.OpenPeerChat(peer.PeerID, username)
			})
			peerBtn.Importance = widget.LowImportance
			contactsList.Add(peerBtn)
		}
	}

	if len(contactsList.Objects) == 0 {
		emptyLabel := widget.NewLabel("No connected peers")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		contactsList.Add(emptyLabel)
	}

	// Компоновка диалога
	DialogContent := container.NewVBox(
		inviteLabel,
		container.NewHBox(inviteEntry, joinBtn),
		widget.NewSeparator(),
		contactsLabel,
		container.NewScroll(contactsList),
	)

	dialog.ShowCustom("New Chat", "Close", DialogContent, fyne.CurrentApp().Driver().AllWindows()[0])
}

// loadChatsList загружает чаты, группы и каналы в список чатов
func loadChatsList(chatsList *fyne.Container, chatUI ChatUIProvider) {
	if chatsList == nil {
		return
	}

	localPeerID := ""
	if chatUI != nil {
		if p2pUI := chatUI.GetP2PService(); p2pUI != nil {
			if host := p2pUI.GetNetwork().Host(); host != nil {
				localPeerID = host.ID().String()
			}
		}
	}

	unifiedChats, err := queries.GetUnifiedChatList(localPeerID)
	if err != nil {
		fyne.Do(func() {
			emptyLabel := widget.NewLabel("Error loading chats")
			emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
			chatsList.Objects = []fyne.CanvasObject{emptyLabel}
			chatsList.Refresh()
		})
		return
	}

	if len(unifiedChats) == 0 {
		fyne.Do(func() {
			emptyLabel := widget.NewLabel("No chats")
			emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
			chatsList.Objects = []fyne.CanvasObject{emptyLabel}
			chatsList.Refresh()
		})
	} else {
		items := make([]fyne.CanvasObject, 0, len(unifiedChats))
		for _, chat := range unifiedChats {
			chatItem := createUnifiedChatItem(chat, chatUI)
			items = append(items, chatItem)
		}
		fyne.Do(func() {
			chatsList.Objects = items
			chatsList.Refresh()
		})
	}
}

// createUnifiedChatItem создаёт элемент чата для unified chat item
func createUnifiedChatItem(chat *models.UnifiedChatItem, chatUI ChatUIProvider) *ChatItemWrapper {
	avatar := createUnifiedChatAvatarIcon(chat, chatUI)

	return NewChatItemWrapper(avatar, chat.ID, chat.PeerID, chat.Username,
		func() {
			openUnifiedChatByID(chat, chatUI)
		},
		func() {
			chatUI.ShowChatMenu(chat.ID, chat.PeerID, chat.Username, nil)
		},
	)
}

// openUnifiedChatByID открывает чат по ID
func openUnifiedChatByID(chat *models.UnifiedChatItem, chatUI ChatUIProvider) {
	if chat.ChatType == "group" || chat.ChatType == "channel" {
		// TODO: открыть групповой чат
		log.Printf("[Chat] Opening group/channel: %s (%s)", chat.Username, chat.GroupUUID)
		return
	}

	if chat.PeerID == models.LocalChatPeerID {
		chatUI.OpenLocalChat()
		return
	}

	profile, err := queries.GetProfileByPeerID(chat.PeerID)
	username := chat.PeerID
	if err == nil && profile != nil {
		username = profile.Username
	}

	if chat.PeerID != "" {
		chatUI.OpenPeerChat(chat.PeerID, username)
	}
}

// createUnifiedChatAvatarIcon создаёт иконку чата с учётом типа
func createUnifiedChatAvatarIcon(chat *models.UnifiedChatItem, chatUI ChatUIProvider) *avatarTappable {
	openAction := func() {
		openUnifiedChatByID(chat, chatUI)
	}
	doubleTapAction := func() {
		chatUI.ShowChatMenu(chat.ID, chat.PeerID, chat.Username, nil)
	}

	bg := canvas.NewRectangle(color.Transparent)
	bg.SetMinSize(fyne.NewSize(50, 50))

	var icon *canvas.Image

	if chat.AvatarPath != "" {
		avatarRes, err := fyne.LoadResourceFromPath(chat.AvatarPath)
		if err == nil && avatarRes != nil {
			icon = canvas.NewImageFromResource(avatarRes)
			icon.FillMode = canvas.ImageFillContain
		}
	}

	if icon == nil {
		switch chat.ChatType {
		case "group":
			icon = canvas.NewImageFromResource(theme.AccountIcon())
		case "channel":
			icon = canvas.NewImageFromResource(theme.MediaRecordIcon())
		default:
			icon = canvas.NewImageFromResource(theme.AccountIcon())
		}
		icon.FillMode = canvas.ImageFillContain
	}
	icon.SetMinSize(fyne.NewSize(50, 50))

	result := newAvatarTappable(chat.ID, chat.PeerID, chat.Username, openAction, doubleTapAction, bg, icon)
	return result
}

// createChatFavoriteIcon создает иконку для чата Избранного (локальный чат с самим собой)
func createChatFavoriteIcon(chatUI ChatUIProvider) *fyne.Container {
	avatar := canvas.NewRectangle(color.RGBA{R: 158, G: 158, B: 158, A: 0})
	avatar.CornerRadius = 15
	avatar.StrokeColor = color.RGBA{R: 255, G: 255, B: 255, A: 100}
	avatar.StrokeWidth = 1
	avatar.SetMinSize(fyne.NewSize(50, 50))

	btn := widget.NewButtonWithIcon("", theme.ContentRedoIcon(), func() {
		log.Printf("[Chat] 🗨️ Нажата кнопка 'Избранное' - открытие локального чата")
		chatUI.OpenLocalChat()
	})
	btn.Importance = widget.LowImportance

	btnWrapper := canvas.NewRectangle(color.Transparent)
	btnWrapper.SetMinSize(fyne.NewSize(50, 50))
	btnContainer := container.NewStack(btnWrapper, btn)

	result := container.NewStack(avatar, btnContainer)
	return result
}

// ChatItemWrapper обёртка для элемента чата с поддержкой двойного клика
type ChatItemWrapper struct {
	widget.BaseWidget
	content     fyne.CanvasObject
	chatID      int
	peerID      string
	username    string
	onSelect    func()
	onDoubleTap func()
}

// NewChatItemWrapper создаёт новую обёртку для элемента чата
func NewChatItemWrapper(content fyne.CanvasObject, chatID int, peerID string, username string, onSelect, onDoubleTap func()) *ChatItemWrapper {
	w := &ChatItemWrapper{
		content:     content,
		chatID:      chatID,
		peerID:      peerID,
		username:    username,
		onSelect:    onSelect,
		onDoubleTap: onDoubleTap,
	}
	w.ExtendBaseWidget(w)
	return w
}

// DoubleTapped обрабатывает двойной клик
func (c *ChatItemWrapper) DoubleTapped(_ *fyne.PointEvent) {
	if c.onDoubleTap != nil {
		c.onDoubleTap()
	}
}

// Tapped обрабатывает одинарный клик
func (c *ChatItemWrapper) Tapped(_ *fyne.PointEvent) {
	if c.onSelect != nil {
		c.onSelect()
	}
}

// CreateRenderer реализует интерфейс fyne.Widget
func (c *ChatItemWrapper) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.content)
}

// MinSize возвращает минимальный размер виджета
func (c *ChatItemWrapper) MinSize() fyne.Size {
	return fyne.NewSize(50, 50)
}

// avatarTappable обёртка для аватара с поддержкой двойного клика
type avatarTappable struct {
	widget.BaseWidget
	chatID      int
	peerID      string
	username    string
	onSelect    func()
	onDoubleTap func()
	bg          *canvas.Rectangle
	icon        *canvas.Image
}

// newAvatarTappable создаёт новую обёрку для аватара
func newAvatarTappable(chatID int, peerID string, username string, onSelect, onDoubleTap func(), bg *canvas.Rectangle, icon *canvas.Image) *avatarTappable {
	w := &avatarTappable{
		chatID:      chatID,
		peerID:      peerID,
		username:    username,
		onSelect:    onSelect,
		onDoubleTap: onDoubleTap,
		bg:          bg,
		icon:        icon,
	}
	w.ExtendBaseWidget(w)
	return w
}

// DoubleTapped обрабатывает двойной клик
func (a *avatarTappable) DoubleTapped(_ *fyne.PointEvent) {
	if a.onDoubleTap != nil {
		a.onDoubleTap()
	}
}

// Tapped обрабатывает одинарный клик
func (a *avatarTappable) Tapped(_ *fyne.PointEvent) {
	if a.onSelect != nil {
		a.onSelect()
	}
}

// CreateRenderer реализует интерфейс fyne.Widget
func (a *avatarTappable) CreateRenderer() fyne.WidgetRenderer {
	objects := []fyne.CanvasObject{a.bg}
	if a.icon != nil {
		objects = append(objects, a.icon)
	}
	return widget.NewSimpleRenderer(container.NewStack(objects...))
}

// MinSize возвращает минимальный размер виджета
func (a *avatarTappable) MinSize() fyne.Size {
	return fyne.NewSize(50, 50)
}
