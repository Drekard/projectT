// Package left содержит компоненты левой панели чатов
package left

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

// Panel представляет левую панель со списком чатов
type Panel struct {
	container *fyne.Container
	chatsList *fyne.Container
	chatsUI   UIProvider
}

// UIProvider интерфейс для доступа к функциям UI чатов
type UIProvider interface {
	OpenPeerChat(peerID, username string)
	RefreshContactsList()
	ShowChatMenu(chatID int, peerID string, username string, cont fyne.CanvasObject)
	OnChatDeleted(chatID int, peerID string)
	OpenLocalChat()
	ShowContactsPanel()
}

// New создает новую левую панель
func New(chatsUI UIProvider) *Panel {
	p := &Panel{
		chatsUI: chatsUI,
	}
	p.container = p.createLeftPanel()
	return p
}

// Container возвращает контейнер панели
func (p *Panel) Container() *fyne.Container {
	return p.container
}

// Refresh обновляет список чатов
func (p *Panel) Refresh() {
	p.loadChatsList()
}

// createLeftPanel создает левую панель со списком чатов
func (p *Panel) createLeftPanel() *fyne.Container {
	// Заголовок с иконками
	header := p.createLeftPanelHeader()

	// Список чатов
	p.chatsList = container.NewVBox()

	// Загружаем чаты из БД
	p.loadChatsList()

	// Вертикальная компоновка
	content := container.NewVBox(header, p.chatsList)

	// Оборачиваем в скролл
	scroll := container.NewVScroll(content)

	return container.NewStack(scroll)
}

// loadChatsList загружает чаты с последними сообщениями в список чатов
func (p *Panel) loadChatsList() {
	if p.chatsList == nil {
		return
	}

	p.chatsList.Objects = nil

	chatsData, err := queries.GetChatsWithLastMessages()
	if err != nil {
		log.Printf("Ошибка загрузки чатов: %v", err)
		// Показываем сообщение об ошибке
		emptyLabel := widget.NewLabel("Ошибка загрузки чатов")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		p.chatsList.Add(emptyLabel)
		p.chatsList.Refresh()
		return
	}

	if len(chatsData) == 0 {
		emptyLabel := widget.NewLabel("Нет чатов")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		p.chatsList.Add(emptyLabel)
	} else {
		for _, chat := range chatsData {
			chatItem := p.createChatItem(chat)
			p.chatsList.Add(chatItem)
		}
	}

	p.chatsList.Refresh()
}

// createChatItem создает элемент чата с аватаром 50x50
func (p *Panel) createChatItem(chat *models.ChatWithLastMessage) *ChatItemWrapper {
	// Аватар 50x50
	avatarContainer := p.createChatAvatarIcon(chat)

	// Основная компоновка: только аватар
	content := container.NewHBox(
		avatarContainer,
		widget.NewSeparator(),
	)

	// Создаём обёртку с обработчиками
	return NewChatItemWrapper(content, chat.ID, chat.PeerID, chat.Username,
		func() {
			p.openChatByID(chat.ID)
		},
		func() {
			p.chatsUI.ShowChatMenu(chat.ID, chat.PeerID, chat.Username, p.chatsList)
		},
	)
}

// createChatAvatarIcon создает иконку чата с аватаром 50x50
func (p *Panel) createChatAvatarIcon(chat *models.ChatWithLastMessage) *fyne.Container {
	if chat.AvatarPath != "" {
		// Пробуем загрузить аватар из файла
		avatarRes, err := fyne.LoadResourceFromPath(chat.AvatarPath)
		if err == nil && avatarRes != nil {
			// Создаём изображение аватара
			img := canvas.NewImageFromResource(avatarRes)
			img.FillMode = canvas.ImageFillContain
			img.SetMinSize(fyne.NewSize(50, 50))

			// Создаём кнопку с изображением
			btn := widget.NewButton("", func() {
				p.openChatByID(chat.ID)
			})
			btn.Importance = widget.LowImportance

			// Оборачиваем в контейнер с фиксированным размером
			btnWrapper := canvas.NewRectangle(color.Transparent)
			btnWrapper.SetMinSize(fyne.NewSize(50, 50))

			// Ставим изображение поверх кнопки
			return container.NewStack(btnWrapper, btn, img)
		}
	}

	// Аватара нет - используем иконку по умолчанию
	icon := canvas.NewImageFromResource(theme.AccountIcon())
	icon.FillMode = canvas.ImageFillContain
	icon.SetMinSize(fyne.NewSize(50, 50))

	// Создаём кнопку с иконкой
	btn := widget.NewButton("", func() {
		p.openChatByID(chat.ID)
	})
	btn.Importance = widget.LowImportance

	// Оборачиваем в контейнер с фиксированным размером
	btnWrapper := canvas.NewRectangle(color.Transparent)
	btnWrapper.SetMinSize(fyne.NewSize(50, 50))

	return container.NewStack(btnWrapper, btn, icon)
}

// openChatByID открывает чат по ID контакта
func (p *Panel) openChatByID(contactID int) {
	// Получаем контакт по ID
	contact, err := queries.GetContact(contactID)
	if err != nil {
		log.Printf("Ошибка получения контакта %d: %v", contactID, err)
		return
	}

	// Открываем чат через публичный метод
	if contact.PeerID != "" {
		p.chatsUI.OpenPeerChat(contact.PeerID, contact.Username)
	}
}
