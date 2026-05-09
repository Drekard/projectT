// Package p2p содержит компонент вкладки "Подключение"
package p2p

import (
	"image/color"

	network "projectT/internal/services/p2p/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// UI представляет интерфейс вкладки "Подключение"
type UI struct {
	content       *fyne.Container
	window        fyne.Window
	p2pUI         *network.UIP2P
	p2pUIProvider UIProvider
	onNavigate    func(contentType string)

	// Элементы UI для настроек
	portEntry       *widget.Entry
	stunServerEntry *widget.Entry
	natPortMapCheck *widget.Check
	relayCheck      *widget.Check
	autoRelayCheck  *widget.Check
	dhtCheck        *widget.Check
	mdnsCheck       *widget.Check
	helperModeCheck *widget.Check

	// Адрес
	addressLabel *widget.Label

	// Списки
	connectedPeersList  *fyne.Container
	discoveredPeersList *fyne.Container
	profilesList        *fyne.Container
}

// UIProvider интерфейс для доступа к функциям UI
type UIProvider interface {
	OpenPeerChat(peerID, username string)
	OpenLocalChat()
	OpenRemoteProfile(peerID string)
	GetWindow() fyne.Window
}

// New создает новый UI вкладки "Подключение"
func New(p2pUIProvider UIProvider, onNavigate func(contentType string)) *UI {
	ui := &UI{
		p2pUIProvider: p2pUIProvider,
		onNavigate:    onNavigate,
	}
	ui.content = ui.createP2PContent()
	return ui
}

// SetWindow устанавливает окно
func (ui *UI) SetWindow(window fyne.Window) {
	ui.window = window
}

// SetP2PService устанавливает P2P сервис
func (ui *UI) SetP2PService(p2pUI *network.UIP2P) {
	ui.p2pUI = p2pUI
	// Загружаем настройки после установки p2pUI
	if ui.portEntry != nil {
		ui.loadP2PSettings()
	}
}

// GetContent возвращает контент вкладки
func (ui *UI) GetContent() fyne.CanvasObject {
	return ui.content
}

// Refresh обновляет UI
func (ui *UI) Refresh() {
	ui.loadConnectedPeers()
	// ui.loadBootstrapPeers() // ❌ Удалено - bootstrap пиры не используются
	ui.loadDiscoveredPeers()
	ui.loadProfiles()
	if ui.content != nil {
		ui.content.Refresh()
	}
}

// createP2PContent создает содержимое вкладки "Подключение"
func (ui *UI) createP2PContent() *fyne.Container {
	// Создаем вкладки
	tabs := container.NewAppTabs(
		container.NewTabItem("Settings", ui.createSettingsTab()),
		container.NewTabItem("Connections", ui.createProfilesTab()),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	// Фон
	bg := canvas.NewRectangle(color.RGBA{R: 30, G: 30, B: 30, A: 0})

	return container.NewStack(bg, tabs)
}
