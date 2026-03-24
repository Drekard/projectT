// Package p2p содержит компоненты панели управления P2P
package p2p

import (
	"image/color"

	"projectT/internal/services/p2p/network"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Panel представляет панель управления P2P с тремя вкладками
type Panel struct {
	container *fyne.Container
	chatsUI   UIProvider
	p2pUI     *network.UIP2P

	// Элементы UI для вкладок
	contactsListInPanel *fyne.Container
	connectedPeersList  *fyne.Container
	bootstrapList       *fyne.Container
	discoveredPeersList *fyne.Container
	profilesListInPanel *fyne.Container

	// Элементы настроек
	myAddressLabel        *widget.Label
	connectionStatusLabel *widget.Label
	peersCountLabel       *widget.Label
	natStatusLabel        *widget.Label
	portEntry             *widget.Entry
	addressEntry          *widget.Entry
	usernameEntry         *widget.Entry
	bootstrapEntry        *widget.Entry
	stunServerEntry       *widget.Entry
	natPortMapCheck       *widget.Check
	relayCheck            *widget.Check
	autoRelayCheck        *widget.Check
	dhtCheck              *widget.Check
	mdnsCheck             *widget.Check
	stunCheck             *widget.Check
	helperModeCheck       *widget.Check
}

// UIProvider интерфейс для доступа к функциям UI чатов
type UIProvider interface {
	OpenPeerChat(peerID, username string)
	RefreshContactsList()
	GetP2PService() *network.UIP2P
	GetWindow() fyne.Window
}

// New создает новую панель управления P2P
func New(chatsUI UIProvider) *Panel {
	p := &Panel{
		chatsUI: chatsUI,
	}
	p.container = p.createControlPanel()
	return p
}

// Container возвращает контейнер панели
func (p *Panel) Container() *fyne.Container {
	return p.container
}

// SetP2PService устанавливает P2P сервис
func (p *Panel) SetP2PService(p2pUI *network.UIP2P) {
	p.p2pUI = p2pUI
}

// createControlPanel создает панель управления P2P с тремя вкладками
func (p *Panel) createControlPanel() *fyne.Container {
	// Создаем вкладки
	tabs := container.NewAppTabs(
		container.NewTabItem("Контакты", p.createContactsTab()),
		container.NewTabItem("Настройки P2P", p.createSettingsTab()),
		container.NewTabItem("Сеть", p.createNetworkTab()),
	)

	tabs.SetTabLocation(container.TabLocationTop)

	// Фон
	bg := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 255})

	return container.NewStack(bg, tabs)
}
