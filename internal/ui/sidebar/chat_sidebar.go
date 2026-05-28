package sidebar

import (
	"image/color"
	"log"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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

	// Заголовок с иконками
	headerIcons := container.NewVBox(favoriteIcon)
	header := container.NewBorder(nil, refreshBtn, nil, nil,
		container.NewPadded(headerIcons),
	)

	// Список чатов
	chatsList := container.NewVBox()
	go loadChatsList(chatsList, state.ChatUI)

	// Вертикальная компоновка
	content := container.NewVBox(backBtn, header, chatsList)

	// Оборачиваем в скролл
	scroll := container.NewVScroll(content)

	sidebarContainer := container.NewStack(scroll)
	sidebarContainer.Resize(fyne.NewSize(state.Width, 0))

	return sidebarContainer
}

// loadChatsList загружает чаты с последними сообщениями в список чатов
func loadChatsList(chatsList *fyne.Container, chatUI ChatUIProvider) {
	if chatsList == nil {
		return
	}

	chatsData, err := queries.GetChatsWithLastMessages()
	if err != nil {
		fyne.Do(func() {
			emptyLabel := widget.NewLabel("Error loading chats")
			emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
			chatsList.Objects = []fyne.CanvasObject{emptyLabel}
			chatsList.Refresh()
		})
		return
	}

	if len(chatsData) == 0 {
		fyne.Do(func() {
			emptyLabel := widget.NewLabel("No chats")
			emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
			chatsList.Objects = []fyne.CanvasObject{emptyLabel}
			chatsList.Refresh()
		})
	} else {
		items := make([]fyne.CanvasObject, 0, len(chatsData))
		for _, chat := range chatsData {
			chatItem := createChatItem(chat, chatUI)
			items = append(items, chatItem)
		}
		fyne.Do(func() {
			chatsList.Objects = items
			chatsList.Refresh()
		})
	}
}

// createChatItem создает элемент чата с аватаром 50x50
func createChatItem(chat *models.ChatWithLastMessage, chatUI ChatUIProvider) *ChatItemWrapper {
	avatar := createChatAvatarIcon(chat, chatUI)

	return NewChatItemWrapper(avatar, chat.ID, chat.PeerID, chat.Username,
		func() {
			openChatByID(chat, chatUI)
		},
		func() {
			chatUI.ShowChatMenu(chat.ID, chat.PeerID, chat.Username, nil)
		},
	)
}

// openChatByID открывает чат по ID
func openChatByID(chat *models.ChatWithLastMessage, chatUI ChatUIProvider) {
	if chat.PeerID == models.LocalChatPeerID {
		chatUI.OpenLocalChat()
		return
	}

	profile, err := queries.GetProfileByPeerID(chat.PeerID)
	username := chat.PeerID[:8]
	if err == nil && profile != nil {
		username = profile.Username
	}

	if chat.PeerID != "" {
		chatUI.OpenPeerChat(chat.PeerID, username)
	}
}

// createChatAvatarIcon создает иконку чата с аватаром 50x50
func createChatAvatarIcon(chat *models.ChatWithLastMessage, chatUI ChatUIProvider) *avatarTappable {
	openAction := func() {
		openChatByID(chat, chatUI)
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
		icon = canvas.NewImageFromResource(theme.AccountIcon())
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
