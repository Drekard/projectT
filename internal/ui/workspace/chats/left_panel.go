package chats

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

// createLeftPanel создает левую панель со списком чатов
func (ui *UI) createLeftPanel() *fyne.Container {
	// Заголовок с иконками
	header := ui.createLeftPanelHeader()

	// Список чатов
	ui.chatsList = container.NewVBox()

	// Загружаем контакты из БД
	ui.loadContactsToChatsList()

	// Вертикальная компоновка
	content := container.NewVBox(header, ui.chatsList)

	// Оборачиваем в скролл
	scroll := container.NewVScroll(content)

	return container.NewStack(scroll)
}

// loadContactsToChatsList загружает контакты из БД в список чатов
func (ui *UI) loadContactsToChatsList() {
	if ui.chatsList == nil {
		return
	}

	ui.chatsList.Objects = nil

	contacts, err := queries.GetAllContacts()
	if err != nil {
		log.Printf("Ошибка загрузки контактов: %v", err)
		// Показываем сообщение об ошибке
		emptyLabel := widget.NewLabel("Ошибка загрузки контактов")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.chatsList.Add(emptyLabel)
		return
	}

	if len(contacts) == 0 {
		emptyLabel := widget.NewLabel("Нет контактов")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.chatsList.Add(emptyLabel)
	} else {
		for _, contact := range contacts {
			peerItem := ui.createPeerItem(contact)
			ui.chatsList.Add(peerItem)
		}
	}

	ui.chatsList.Refresh()
}

// createPeerItem создает элемент пира в списке
func (ui *UI) createPeerItem(contact *models.Contact) *fyne.Container {
	// Создаём иконку пира с аватаром
	peerIcon := ui.createPeerAvatarIcon(contact)

	// Основная компоновка: иконка + разделитель
	content := container.NewBorder(
		nil, nil,
		nil, nil,
		peerIcon,
		widget.NewSeparator(),
	)

	return content
}

// createPeerAvatarIcon создает иконку пира с аватаром 50x50
func (ui *UI) createPeerAvatarIcon(contact *models.Contact) *fyne.Container {
	// Создаём фон с закругленными углами
	avatarBg := canvas.NewRectangle(color.RGBA{R: 50, G: 50, B: 50, A: 255})
	avatarBg.CornerRadius = 10
	avatarBg.StrokeColor = color.RGBA{R: 255, G: 255, B: 255, A: 100}
	avatarBg.StrokeWidth = 1
	avatarBg.SetMinSize(fyne.NewSize(50, 50))

	// Создаём кнопку с иконкой поверх фона
	peerBtn := widget.NewButtonWithIcon("", theme.AccountIcon(), func() {
		ui.openPeerChat(contact)
	})
	peerBtn.Importance = widget.LowImportance

	// Оборачиваем кнопку в контейнер с фиксированным размером
	btnWrapper := canvas.NewRectangle(color.Transparent)
	btnWrapper.SetMinSize(fyne.NewSize(50, 50))
	btnContainer := container.NewStack(btnWrapper, peerBtn)

	// Оборачиваем в контейнер с фоном
	return container.NewStack(avatarBg, btnContainer)
}

// openPeerChat открывает чат с пиром
func (ui *UI) openPeerChat(contact *models.Contact) {
	if ui.window == nil {
		log.Printf("Окно не инициализировано")
		return
	}

	// Выбираем чат
	ui.selectChat(contact)

	// Загружаем сообщения для контакта
	ui.loadMessagesForContact(contact.ID)

	// Запрашиваем профиль у пира если P2P инициализирован
	if ui.p2pUI != nil {
		go func() {
			err := ui.p2pUI.RequestProfile(contact.PeerID)
			if err != nil {
				log.Printf("Не удалось запросить профиль у пира %s: %v", contact.PeerID, err)
			}
		}()
	}

	// Обновляем правую панель с профилем
	ui.updateProfile(contact)

	// Обновляем UI
	if ui.chatArea != nil {
		ui.chatArea.Refresh()
	}
}

// createLeftPanelHeader создает заголовок левой панели с иконками
func (ui *UI) createLeftPanelHeader() *fyne.Container {
	// Иконка контактов
	contactsIcon := ui.createContactsIcon()

	// Иконка чата с собой
	faworiteIcon := ui.createFaworiteIcon()

	// Вертикальная компоновка иконок
	icons := container.NewVBox(
		contactsIcon,
		faworiteIcon,
	)

	return container.NewPadded(icons)
}

// createContactsIcon создает иконку для панели контактов
func (ui *UI) createContactsIcon() *fyne.Container {
	// Создаем фон с закругленными углами
	avatar := canvas.NewRectangle(color.RGBA{R: 158, G: 158, B: 158, A: 0})
	avatar.CornerRadius = 15
	avatar.StrokeColor = color.RGBA{R: 255, G: 255, B: 255, A: 100}
	avatar.StrokeWidth = 1
	avatar.SetMinSize(fyne.NewSize(50, 50))

	// Создаем кнопку с иконкой поверх графики
	btn := widget.NewButtonWithIcon("", theme.AccountIcon(), func() {
		ui.showContactsPanel()
	})
	btn.Importance = widget.LowImportance

	// Оборачиваем кнопку в контейнер с фиксированным размером
	btnWrapper := canvas.NewRectangle(color.Transparent)
	btnWrapper.SetMinSize(fyne.NewSize(50, 50))
	btnContainer := container.NewStack(btnWrapper, btn)

	// Оборачиваем в контейнер
	return container.NewStack(avatar, btnContainer)
}

// createFaworiteIcon создает иконку для чата Избранного
func (ui *UI) createFaworiteIcon() *fyne.Container {
	// Создаем фон с закругленными углами
	avatar := canvas.NewRectangle(color.RGBA{R: 158, G: 158, B: 158, A: 0})
	avatar.CornerRadius = 15
	avatar.StrokeColor = color.RGBA{R: 255, G: 255, B: 255, A: 100}
	avatar.StrokeWidth = 1
	avatar.SetMinSize(fyne.NewSize(50, 50))

	// Создаем кнопку с иконкой поверх графики
	btn := widget.NewButtonWithIcon("", theme.MailAttachmentIcon(), func() {
		ui.showContactsPanel()
	})
	btn.Importance = widget.LowImportance

	// Оборачиваем кнопку в контейнер с фиксированным размером
	btnWrapper := canvas.NewRectangle(color.Transparent)
	btnWrapper.SetMinSize(fyne.NewSize(50, 50))
	btnContainer := container.NewStack(btnWrapper, btn)

	// Оборачиваем в контейнер
	return container.NewStack(avatar, btnContainer)
}

// showContactsPanel показывает панель управления P2P
func (ui *UI) showContactsPanel() {
	controlPanel := ui.createControlPanel()
	ui.chatArea.Objects = []fyne.CanvasObject{controlPanel}
	ui.chatArea.Refresh()
}
