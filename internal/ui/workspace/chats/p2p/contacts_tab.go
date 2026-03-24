// Package p2p содержит компоненты панели управления P2P
package p2p

import (
	"fmt"
	"image/color"

	"projectT/internal/storage/database/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// createContactsTab создает вкладку с контактами
func (p *Panel) createContactsTab() fyne.CanvasObject {
	// Заголовок
	title := widget.NewLabel("Контакты")
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Кнопка добавления контакта
	addButton := widget.NewButtonWithIcon("Добавить контакт", theme.ContentAddIcon(), func() {
		p.showAddContactDialog()
	})
	addButton.Importance = widget.HighImportance

	// Список контактов
	p.contactsListInPanel = container.NewVBox()
	p.loadContactsList()

	// Разделитель
	sep := widget.NewSeparator()

	// Разделитель для добавления контакта вручную
	manualLabel := widget.NewLabel("Добавить контакт по адресу")
	manualLabel.TextStyle = fyne.TextStyle{Italic: true}

	p.addressEntry = widget.NewEntry()
	p.addressEntry.SetPlaceHolder("projectt:peerid@/ip4/.../tcp/.../p2p/...")

	p.usernameEntry = widget.NewEntry()
	p.usernameEntry.SetPlaceHolder("Имя контакта (необязательно)")

	addManualButton := widget.NewButtonWithIcon("Добавить контакт", theme.ContentAddIcon(), func() {
		p.addContactByAddress()
	})
	addManualButton.Importance = widget.HighImportance

	manualSection := container.NewVBox(
		manualLabel,
		p.addressEntry,
		p.usernameEntry,
		addManualButton,
	)

	content := container.NewVBox(
		title,
		addButton,
		sep,
		p.contactsListInPanel,
		widget.NewSeparator(),
		manualSection,
	)

	return container.NewScroll(content)
}

// loadContactsList загружает список контактов из базы данных
func (p *Panel) loadContactsList() {
	if p.contactsListInPanel == nil {
		return
	}

	p.contactsListInPanel.Objects = nil

	if p.p2pUI == nil {
		emptyLabel := widget.NewLabel("P2P сервис не инициализирован")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		p.contactsListInPanel.Add(emptyLabel)
		p.contactsListInPanel.Refresh()
		return
	}

	// Получаем контакты из базы данных
	contacts, err := p.p2pUI.GetContacts()
	if err != nil {
		emptyLabel := widget.NewLabel(fmt.Sprintf("Ошибка загрузки контактов: %v", err))
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		p.contactsListInPanel.Add(emptyLabel)
		p.contactsListInPanel.Refresh()
		return
	}

	if len(contacts) == 0 {
		emptyLabel := widget.NewLabel("Список контактов пуст")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		p.contactsListInPanel.Add(emptyLabel)
	} else {
		for _, contact := range contacts {
			contactItem := p.createContactItem(contact)
			p.contactsListInPanel.Add(contactItem)
		}
	}

	p.contactsListInPanel.Refresh()
}

// createContactItem создает элемент контакта
func (p *Panel) createContactItem(contact *models.Contact) *fyne.Container {
	// Индикатор статуса (зеленый если онлайн)
	statusInd := canvas.NewCircle(color.RGBA{R: 128, G: 128, B: 128, A: 255})

	// Проверяем, подключен ли контакт
	if p.p2pUI != nil && contact.PeerID != "" {
		connectedPeers := p.p2pUI.GetConnectedPeers()
		for _, peer := range connectedPeers {
			if peer.PeerID == contact.PeerID {
				statusInd.FillColor = color.RGBA{R: 76, G: 175, B: 80, A: 255}
				break
			}
		}
	}

	// Имя контакта
	nameLabel := widget.NewLabel(contact.Username)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	// PeerID (сокращенный)
	peerIDText := "нет ID"
	if contact.PeerID != "" {
		peerIDText = contact.PeerID
		if len(peerIDText) > 16 {
			peerIDText = peerIDText[:8] + "..." + peerIDText[len(peerIDText)-8:]
		}
	}
	peerIDLabel := widget.NewLabel(peerIDText)
	peerIDLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Кнопка начала чата
	chatBtn := widget.NewButtonWithIcon("", theme.MailComposeIcon(), func() {
		p.openChatWithContact(contact)
	})

	// Кнопка подключения
	connectBtn := widget.NewButtonWithIcon("", theme.FolderIcon(), func() {
		p.connectToContactByContact(contact)
	})

	// Кнопка удаления
	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		p.deleteContact(contact)
	})

	content := container.NewBorder(
		nil,
		container.NewHBox(chatBtn, connectBtn, deleteBtn),
		container.NewHBox(statusInd, container.NewVBox(nameLabel, peerIDLabel)),
		nil,
		widget.NewSeparator(),
	)

	return content
}

// showAddContactDialog показывает диалог добавления контакта
func (p *Panel) showAddContactDialog() {
	window := p.chatsUI.GetWindow()
	if window == nil {
		return
	}

	addressEntry := widget.NewEntry()
	addressEntry.SetPlaceHolder("projectt:peerid@/ip4/.../tcp/.../p2p/...")
	addressEntry.MultiLine = true
	addressEntry.Wrapping = fyne.TextWrapBreak

	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("Имя контакта (необязательно)")

	content := container.NewVBox(
		widget.NewLabel("Введите адрес контакта:"),
		addressEntry,
		widget.NewLabel("Имя (необязательно):"),
		usernameEntry,
	)

	d := dialog.NewCustomConfirm(
		"Добавить контакт",
		"Добавить",
		"Отмена",
		content,
		func(confirmed bool) {
			if !confirmed {
				return
			}

			if addressEntry.Text == "" {
				p.showErrorDialog("Ошибка", "Введите адрес контакта")
				return
			}

			if p.p2pUI == nil {
				p.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
				return
			}

			err := p.p2pUI.AddContactByAddress(addressEntry.Text, usernameEntry.Text)
			if err != nil {
				p.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось добавить контакт: %v", err))
				return
			}

			p.showInfoDialog("Успешно", "Контакт добавлен")
			p.loadContactsList()

			// Обновляем список контактов в левой панели
			p.chatsUI.RefreshContactsList()
		},
		window,
	)
	d.Show()
}

// openChatWithContact открывает чат с контактом
func (p *Panel) openChatWithContact(contact *models.Contact) {
	// Открываем чат через публичный метод
	if contact.PeerID != "" {
		p.chatsUI.OpenPeerChat(contact.PeerID, contact.Username)
	} else {
		p.showErrorDialog("Ошибка", "У контакта нет PeerID")
	}
}

// connectToContactByContact подключается к контакту
func (p *Panel) connectToContactByContact(contact *models.Contact) {
	if p.p2pUI == nil {
		p.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	if contact.PeerID == "" {
		p.showErrorDialog("Ошибка", "У контакта нет PeerID")
		return
	}

	// Получаем multiaddr контакта
	multiaddr := contact.Multiaddr
	if multiaddr == "" {
		p.showErrorDialog("Ошибка", "У контакта нет адреса")
		return
	}

	addrStr := fmt.Sprintf("%s@%s", contact.PeerID, multiaddr)

	err := p.p2pUI.ConnectToContact(addrStr)
	if err != nil {
		p.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось подключиться: %v", err))
		return
	}

	p.showInfoDialog("Подключение", "Попытка подключения к контакту...")
}

// deleteContact удаляет контакт
func (p *Panel) deleteContact(contact *models.Contact) {
	window := p.chatsUI.GetWindow()
	if window == nil {
		return
	}

	dialog.ShowConfirm(
		"Удаление контакта",
		fmt.Sprintf("Вы действительно хотите удалить контакт \"%s\"?", contact.Username),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			if p.p2pUI == nil {
				p.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
				return
			}

			err := p.p2pUI.DeleteContact(contact.ID)
			if err != nil {
				p.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось удалить контакт: %v", err))
				return
			}

			p.loadContactsList()

			// Обновляем список контактов в левой панели
			p.chatsUI.RefreshContactsList()
		},
		window,
	)
}

// addContactByAddress добавляет контакт по адресу
func (p *Panel) addContactByAddress() {
	if p.p2pUI == nil {
		p.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	addrStr := p.addressEntry.Text
	if addrStr == "" {
		p.showErrorDialog("Ошибка", "Введите адрес контакта")
		return
	}

	username := p.usernameEntry.Text

	err := p.p2pUI.AddContactByAddress(addrStr, username)
	if err != nil {
		p.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось добавить контакт: %v", err))
		return
	}

	p.showInfoDialog("Успешно", "Контакт добавлен")
	p.addressEntry.SetText("")
	p.usernameEntry.SetText("")

	// Обновляем список контактов в левой панели
	p.chatsUI.RefreshContactsList()

	// Обновляем список контактов во вкладке "Контакты"
	p.loadContactsList()
}
