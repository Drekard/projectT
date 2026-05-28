package sidebar

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// NavigationHandler определяет интерфейс для обработки навигации
type NavigationHandler interface {
	OnNavigationChanged(contentType string, param ...interface{})
	NavigateToFolder(folderID int) error
	SearchByTag(tagName string) error
	SetSearchQuery(query string) error
	ResetToSaved()
	OnSidebarStateChanged() // Вызывается при изменении состояния sidebar
}

// NavigationButtons хранит ссылки на все кнопки навигации
type NavigationButtons struct {
	Profile     *widget.Button
	Saved       *widget.Button
	Preview     *widget.Button
	Tags        *widget.Button
	Chats       *widget.Button
	Contacts    *widget.Button
	P2P         *widget.Button
	Compilation *widget.Button
}

// SetActive обновляет визуальное выделение активной кнопки
func (nb *NavigationButtons) SetActive(section string) {
	buttons := []*widget.Button{nb.Profile, nb.Saved, nb.Preview, nb.Tags, nb.Chats, nb.Contacts, nb.P2P, nb.Compilation}
	for _, btn := range buttons {
		if btn != nil {
			btn.Importance = widget.LowImportance
			btn.Refresh()
		}
	}

	var activeBtn *widget.Button
	switch section {
	case "profile":
		activeBtn = nb.Profile
	case "saved":
		activeBtn = nb.Saved
	case "preview":
		activeBtn = nb.Preview
	case "tags":
		activeBtn = nb.Tags
	case "chats":
		activeBtn = nb.Chats
	case "contacts":
		activeBtn = nb.Contacts
	case "p2p":
		activeBtn = nb.P2P
	case "compilation":
		activeBtn = nb.Compilation
	}
	if activeBtn != nil {
		activeBtn.Importance = widget.MediumImportance
		activeBtn.Refresh()
	}
}

// CreateNavigation создает навигационные кнопки
func CreateNavigation(state *SidebarState, handler NavigationHandler) (*fyne.Container, *NavigationButtons) {
	buttons := &NavigationButtons{}

	profileButton := createCustomNavButton(state, "Profile", theme.AccountIcon(), func() {
		state.ChatMode = false
		state.ActiveSection = "profile"
		buttons.SetActive("profile")
		if handler != nil {
			handler.OnNavigationChanged("profile")
		}
	})
	buttons.Profile = profileButton

	savedButton := createCustomNavButton(state, "Saved", theme.HomeIcon(), func() {
		state.ChatMode = false
		state.ActiveSection = "saved"
		buttons.SetActive("saved")
		if handler != nil {
			handler.OnNavigationChanged("saved")
			handler.ResetToSaved()
		}
	})
	buttons.Saved = savedButton

	previewButton := createCustomNavButton(state, "Downloads", theme.DownloadIcon(), func() {
		state.ChatMode = false
		state.ActiveSection = "preview"
		buttons.SetActive("preview")
		if handler != nil {
			handler.OnNavigationChanged("preview")
		}
	})
	buttons.Preview = previewButton

	tagsButton := createCustomNavButton(state, "My Tags", theme.SettingsIcon(), func() {
		state.ChatMode = false
		state.ActiveSection = "tags"
		buttons.SetActive("tags")
		if handler != nil {
			handler.OnNavigationChanged("tags")
		}
	})
	buttons.Tags = tagsButton

	chatsButton := createCustomNavButton(state, "Chats", theme.MailComposeIcon(), func() {
		state.ActiveSection = "chats"
		buttons.SetActive("chats")
		state.ChatMode = true
		if handler != nil {
			handler.OnSidebarStateChanged()
		}
	})
	buttons.Chats = chatsButton

	contactsButton := createCustomNavButton(state, "Contacts", theme.AccountIcon(), func() {
		state.ChatMode = false
		state.ActiveSection = "contacts"
		buttons.SetActive("contacts")
		if handler != nil {
			handler.OnNavigationChanged("contacts")
		}
	})
	buttons.Contacts = contactsButton

	p2pButton := createCustomNavButton(state, "P2P", theme.ComputerIcon(), func() {
		state.ChatMode = false
		state.ActiveSection = "p2p"
		buttons.SetActive("p2p")
		if handler != nil {
			handler.OnNavigationChanged("p2p")
		}
	})
	buttons.P2P = p2pButton

	compilationButton := createCustomNavButton(state, "Compilation", theme.GridIcon(), func() {
		state.ChatMode = false
		state.ActiveSection = "compilation"
		buttons.SetActive("compilation")
		if handler != nil {
			handler.OnNavigationChanged("compilation")
		}
	})
	buttons.Compilation = compilationButton

	if state.ActiveSection == "" {
		state.ActiveSection = "saved"
	}
	buttons.SetActive(state.ActiveSection)

	if state.Collapsed {
		result := container.NewVBox(
			profileButton,
			savedButton,
			previewButton,
			tagsButton,
			chatsButton,
			contactsButton,
			p2pButton,
			compilationButton,
		)
		return result, buttons
	}

	separator := widget.NewSeparator()
	result := container.NewVBox(
		profileButton,
		savedButton,
		previewButton,
		tagsButton,
		chatsButton,
		contactsButton,
		p2pButton,
		compilationButton,
		separator,
	)
	return result, buttons
}

func createCustomNavButton(state *SidebarState, label string, icon fyne.Resource, onClick func()) *widget.Button {
	button := widget.NewButtonWithIcon("", icon, onClick)
	button.Alignment = widget.ButtonAlignLeading
	button.Importance = widget.LowImportance
	if !state.Collapsed {
		button.SetText(label)
	}
	return button
}
