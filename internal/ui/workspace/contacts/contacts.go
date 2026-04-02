// Package contacts содержит компонент вкладки "Контакты"
package contacts

import (
	"fmt"
	"image/color"
	"log"

	network "projectT/internal/services/p2p/ui"
	"projectT/internal/storage/database/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// UI представляет интерфейс вкладки "Контакты"
type UI struct {
	content      *fyne.Container
	window       fyne.Window
	p2pUI        *network.UIP2P
	contactsUI   UIProvider
	contactsList *fyne.Container

	// Элементы для добавления контакта
	addressEntry  *widget.Entry
	usernameEntry *widget.Entry
}

// UIProvider интерфейс для доступа к функциям UI
type UIProvider interface {
	OpenPeerChat(peerID, username string)
	OpenLocalChat()
	GetWindow() fyne.Window
}

// New создает новый UI вкладки "Контакты"
func New(contactsUI UIProvider) *UI {
	ui := &UI{
		contactsUI: contactsUI,
	}
	ui.content = ui.createContactsContent()
	return ui
}

// SetWindow устанавливает окно
func (ui *UI) SetWindow(window fyne.Window) {
	ui.window = window
}

// SetP2PService устанавливает P2P сервис
func (ui *UI) SetP2PService(p2pUI *network.UIP2P) {
	ui.p2pUI = p2pUI
	ui.loadContactsList()
}

// GetContent возвращает контент вкладки
func (ui *UI) GetContent() fyne.CanvasObject {
	return ui.content
}

// Refresh обновляет UI
func (ui *UI) Refresh() {
	ui.loadContactsList()
	if ui.content != nil {
		ui.content.Refresh()
	}
}

// createContactsContent создает содержимое вкладки "Контакты"
func (ui *UI) createContactsContent() *fyne.Container {
	// Заголовок
	title := widget.NewLabel("Контакты")
	title.TextStyle = fyne.TextStyle{Bold: true}

	// Кнопка добавления контакта
	addButton := widget.NewButtonWithIcon("Добавить контакт", theme.ContentAddIcon(), func() {
		ui.showAddContactDialog()
	})
	addButton.Importance = widget.HighImportance

	// Список контактов
	ui.contactsList = container.NewVBox()

	// Разделитель
	sep := widget.NewSeparator()

	// Разделитель для добавления контакта вручную
	manualLabel := widget.NewLabel("Добавить контакт по адресу")
	manualLabel.TextStyle = fyne.TextStyle{Italic: true}

	ui.addressEntry = widget.NewEntry()
	ui.addressEntry.SetPlaceHolder("projectt:peerid@/ip4/.../tcp/.../p2p/...")

	ui.usernameEntry = widget.NewEntry()
	ui.usernameEntry.SetPlaceHolder("Имя контакта (необязательно)")

	addManualButton := widget.NewButtonWithIcon("Добавить контакт", theme.ContentAddIcon(), func() {
		ui.addContactByAddress()
	})
	addManualButton.Importance = widget.HighImportance

	manualSection := container.NewVBox(
		manualLabel,
		ui.addressEntry,
		ui.usernameEntry,
		addManualButton,
	)

	content := container.NewVBox(
		title,
		addButton,
		sep,
		ui.contactsList,
		widget.NewSeparator(),
		manualSection,
	)

	// Фон
	bg := canvas.NewRectangle(color.RGBA{R: 30, G: 30, B: 30, A: 0})

	return container.NewStack(bg, container.NewScroll(content))
}

// loadContactsList загружает список контактов из базы данных
func (ui *UI) loadContactsList() {
	if ui.contactsList == nil {
		return
	}

	ui.contactsList.Objects = nil

	if ui.p2pUI == nil {
		emptyLabel := widget.NewLabel("P2P сервис не инициализирован")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.contactsList.Add(emptyLabel)
		ui.contactsList.Refresh()
		return
	}

	// Получаем контакты из базы данных
	contacts, err := ui.p2pUI.GetContacts()
	if err != nil {
		emptyLabel := widget.NewLabel(fmt.Sprintf("Ошибка загрузки контактов: %v", err))
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.contactsList.Add(emptyLabel)
		ui.contactsList.Refresh()
		return
	}

	if len(contacts) == 0 {
		emptyLabel := widget.NewLabel("Список контактов пуст")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.contactsList.Add(emptyLabel)
	} else {
		for _, contact := range contacts {
			contactItem := ui.createContactItem(contact)
			ui.contactsList.Add(contactItem)
		}
	}

	ui.contactsList.Refresh()
}

// createContactItem создает элемент контакта
func (ui *UI) createContactItem(contact *models.Contact) *fyne.Container {
	// Индикатор статуса (зеленый если онлайн)
	statusInd := canvas.NewCircle(color.RGBA{R: 128, G: 128, B: 128, A: 255})

	// Проверяем, подключен ли контакт
	if ui.p2pUI != nil && contact.PeerID != "" {
		connectedPeers := ui.p2pUI.GetConnectedPeers()
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
		ui.openChatWithContact(contact)
	})

	// Кнопка подключения
	connectBtn := widget.NewButtonWithIcon("", theme.FolderIcon(), func() {
		ui.connectToContactByContact(contact)
	})

	// Кнопка удаления
	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		ui.deleteContact(contact)
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
func (ui *UI) showAddContactDialog() {
	window := ui.contactsUI.GetWindow()
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
				ui.showErrorDialog("Ошибка", "Введите адрес контакта")
				return
			}

			if ui.p2pUI == nil {
				ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
				return
			}

			err := ui.p2pUI.AddContactByAddress(addressEntry.Text, usernameEntry.Text)
			if err != nil {
				ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось добавить контакт: %v", err))
				return
			}

			ui.showInfoDialog("Успешно", "Контакт добавлен")
			ui.loadContactsList()
		},
		window,
	)
	d.Show()
}

// openChatWithContact открывает чат с контактом
func (ui *UI) openChatWithContact(contact *models.Contact) {
	// ✅ ПРОВЕРКА: если это локальный чат - открываем через OpenLocalChat()
	if contact.IsLocalChat() {
		log.Printf("[Contact] 🏠 Обнаружен локальный чат, открываем через OpenLocalChat()")
		ui.contactsUI.OpenLocalChat()
		return
	}

	if contact.PeerID != "" {
		ui.contactsUI.OpenPeerChat(contact.PeerID, contact.Username)
	} else {
		ui.showErrorDialog("Ошибка", "У контакта нет PeerID")
	}
}

// connectToContactByContact подключается к контакту
func (ui *UI) connectToContactByContact(contact *models.Contact) {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	if contact.PeerID == "" {
		ui.showErrorDialog("Ошибка", "У контакта нет PeerID")
		return
	}

	multiaddr := contact.Multiaddr
	if multiaddr == "" {
		ui.showErrorDialog("Ошибка", "У контакта нет адреса")
		return
	}

	addrStr := fmt.Sprintf("%s@%s", contact.PeerID, multiaddr)

	err := ui.p2pUI.ConnectToContact(addrStr)
	if err != nil {
		ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось подключиться: %v", err))
		return
	}

	ui.showInfoDialog("Подключение", "Попытка подключения к контакту...")
}

// deleteContact удаляет контакт
func (ui *UI) deleteContact(contact *models.Contact) {
	window := ui.contactsUI.GetWindow()
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

			if ui.p2pUI == nil {
				ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
				return
			}

			err := ui.p2pUI.DeleteContact(contact.ID)
			if err != nil {
				ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось удалить контакт: %v", err))
				return
			}

			ui.loadContactsList()
		},
		window,
	)
}

// addContactByAddress добавляет контакт по адресу
func (ui *UI) addContactByAddress() {
	if ui.p2pUI == nil {
		ui.showErrorDialog("Ошибка", "P2P сервис не инициализирован")
		return
	}

	addrStr := ui.addressEntry.Text
	if addrStr == "" {
		ui.showErrorDialog("Ошибка", "Введите адрес контакта")
		return
	}

	username := ui.usernameEntry.Text

	err := ui.p2pUI.AddContactByAddress(addrStr, username)
	if err != nil {
		ui.showErrorDialog("Ошибка", fmt.Sprintf("Не удалось добавить контакт: %v", err))
		return
	}

	ui.showInfoDialog("Успешно", "Контакт добавлен")
	ui.addressEntry.SetText("")
	ui.usernameEntry.SetText("")
	ui.loadContactsList()
}

// showErrorDialog показывает диалог ошибки
func (ui *UI) showErrorDialog(title, message string) {
	window := ui.contactsUI.GetWindow()
	if window == nil {
		log.Printf("[%s] %s\n", title, message)
		return
	}
	dialog.ShowError(fmt.Errorf("%s", message), window)
}

// showInfoDialog показывает информационный диалог
func (ui *UI) showInfoDialog(title, message string) {
	window := ui.contactsUI.GetWindow()
	if window == nil {
		log.Printf("[%s] %s\n", title, message)
		return
	}
	dialog.ShowInformation(title, message, window)
}
