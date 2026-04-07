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
		log.Printf("[Chat] ❌ Ошибка загрузки чатов: %v", err)
		// Показываем сообщение об ошибке
		emptyLabel := widget.NewLabel("Error loading chats")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		p.chatsList.Add(emptyLabel)
		p.chatsList.Refresh()
		return
	}

	log.Printf("[Chat] 📚 Загружено чатов из БД: %d", len(chatsData))
	for i, chat := range chatsData {
		log.Printf("[Chat] 📋 Чат #%d: ID=%d, peer=%s, username=%q, lastMsg=%q, contactID=%v",
			i, chat.ID, chat.PeerID[:8], chat.Username, chat.LastMessage, chat.ContactID)
	}

	if len(chatsData) == 0 {
		emptyLabel := widget.NewLabel("No chats")
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
	log.Printf("[Chat] 🎨 Создание элемента чата: ID=%d, peer=%s, lastMsg=%q",
		chat.ID, chat.PeerID[:8], chat.LastMessage)

	// Аватар 50x50
	avatarContainer := p.createChatAvatarIcon(chat)

	// Компонуем: только аватар
	content := container.NewStack(avatarContainer)

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
	log.Printf("[Avatar] 🎨 Создание аватара для чата ID=%d, peer_id=%s...", chat.ID, chat.PeerID[:8])
	log.Printf("[Avatar] 📋 AvatarPath из БД: %q", chat.AvatarPath)

	if chat.AvatarPath != "" {
		// Пробуем загрузить аватар из файла
		log.Printf("[Avatar] 🔍 Попытка загрузки аватара из файла: %s", chat.AvatarPath)
		avatarRes, err := fyne.LoadResourceFromPath(chat.AvatarPath)
		if err != nil {
			log.Printf("[Avatar] ❌ Ошибка загрузки ресурса из пути %q: %v", chat.AvatarPath, err)
		} else if avatarRes == nil {
			log.Printf("[Avatar] ❌ Ресурс аватара пуст (nil) для пути: %s", chat.AvatarPath)
		} else {
			// Аватар успешно загружен
			log.Printf("[Avatar] ✅ Аватар успешно загружен: %s (Content: %d байт)", chat.AvatarPath, len(avatarRes.Content()))

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

			log.Printf("[Avatar] 🖼️ Возврат контейнера с изображением аватара")
			// Ставим изображение поверх кнопки
			return container.NewStack(btnWrapper, btn, img)
		}

		// Если дошли сюда - произошла ошибка загрузки
		log.Printf("[Avatar] ⚠️ Не удалось загрузить аватар, будет использована иконка по умолчанию")
	} else {
		log.Printf("[Avatar] ℹ️ AvatarPath пустой, будет использована иконка по умолчанию")
	}

	// Аватара нет или ошибка загрузки - используем иконку по умолчанию
	log.Printf("[Avatar] 🔲 Использование системной иконки (theme.AccountIcon)")
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

	log.Printf("[Avatar] ⚪ Возврат контейнера с системной иконкой")
	return container.NewStack(btnWrapper, btn, icon)
}

// openChatByID открывает чат по ID
func (p *Panel) openChatByID(chatID int) {
	log.Printf("[Chat] 🚪 Открытие чата по ID=%d", chatID)

	// Сначала пробуем получить чат по ID
	chat, err := queries.GetChat(chatID)
	if err != nil {
		log.Printf("Ошибка получения чата %d: %v", chatID, err)
		return
	}

	// ✅ ПРОВЕРКА: если это локальный чат - открываем через OpenLocalChat()
	if chat.PeerID == models.LocalChatPeerID {
		log.Printf("[Chat] 🏠 Обнаружен локальный чат (Избранное), открываем через OpenLocalChat()")
		p.chatsUI.OpenLocalChat()
		return
	}

	// Получаем профиль пира для отображения имени
	profile, err := queries.GetProfileByPeerID(chat.PeerID)
	username := chat.PeerID[:8] // По умолчанию используем сокращённый PeerID
	if err == nil && profile != nil {
		username = profile.Username
		log.Printf("[Chat] ℹ️ Профиль найден: %s", username)
	}

	log.Printf("[Chat] 🗨️ Открытие чата с пиром: %s (%s)", username, chat.PeerID[:8])

	// Открываем чат через публичный метод
	if chat.PeerID != "" {
		p.chatsUI.OpenPeerChat(chat.PeerID, username)
	}
}
