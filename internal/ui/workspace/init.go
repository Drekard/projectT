package workspace

import (
	"projectT/internal/services"
	"projectT/internal/ui/workspace/chats"
	"projectT/internal/ui/workspace/contacts"
	"projectT/internal/ui/workspace/p2p"
	"projectT/internal/ui/workspace/tags"

	p2p_ui "projectT/internal/services/p2p/ui"
)

// getP2PUIShared возвращает или создаёт общий экземпляр UIP2P
func (ws *Workspace) getP2PUIShared() *p2p_ui.UIP2P {
	if ws.p2pUIShared == nil && ws.p2pNetwork != nil {
		ws.p2pUIShared = p2p_ui.NewUIP2P(ws.p2pNetwork)
	}
	return ws.p2pUIShared
}

// initializeTagsUI инициализирует UI тегов при первом обращении
func (ws *Workspace) initializeTagsUI() {
	if !ws.tagsInitialized {
		ws.tagsUI = tags.New()
		ws.tagsInitialized = true
	}
}

// initializeChatsUI инициализирует UI чатов при первом обращении
func (ws *Workspace) initializeChatsUI() {
	if !ws.chatsInitialized {
		ws.chatsUI = chats.New()
		ws.chatsInitialized = true

		ws.chatsUI.SetWindow(ws.window)
		ws.chatsUI.SetConfig(ws.config)
		ws.chatsUI.SetOnSave(ws.onSave)
		ws.chatsUI.RestoreRightPanelState()

		ws.chatsUI.SetOnOpenRemoteProfile(func(peerID string) {
			ws.OpenRemoteProfile(peerID)
		})

		ws.chatsUI.SetOnOpenFolderFromChat(func(peerID, folderUUID string) {
			ws.OpenRemoteFolderFromChat(peerID, folderUUID)
		})

		ws.chatsUI.SetOnChatModeChanged(func(isChatMode bool, chatName string, onBack, onOpenProfile, onAttach, onToggleRight, onProfileClicked func()) {
			if ws.onChatModeChanged != nil {
				ws.onChatModeChanged(isChatMode, chatName, onBack, onOpenProfile, onAttach, onToggleRight, onProfileClicked)
			}
		})

		if ws.p2pNetwork != nil {
			p2pUI := ws.getP2PUIShared()
			p2pUI.SetOnProfileUpdated(func(peerID string) {
				ws.chatsUI.RefreshRightPanel(peerID)
				// Также обновляем remote profile UI если он открыт для этого пира
				if ws.remoteProfileUI != nil && ws.remoteProfileUI.GetPeerID() == peerID {
					ws.remoteProfileUI.Refresh()
				}
			})
			ws.chatsUI.SetP2PService(p2pUI)

			if ws.p2pNetwork.ProfileExchange() != nil {
				ws.p2pNetwork.ProfileExchange().SetUIP2P(p2pUI)
				ws.p2pNetwork.ProfileExchange().SetUIProfilePanel(ws.chatsUI)
			}

			if ws.p2pNetwork.Transfer() != nil {
				ws.p2pNetwork.Transfer().SetOnBatchComplete(func(sourcePeerID string) {
					ws.chatsUI.RefreshRightPanel(sourcePeerID)
				})
			}

			services.SetGlobalP2PNetwork(ws.p2pNetwork)
		}

		ws.chatsUI.SubscribeToMessages()
	}
}

// initializeContactsUI инициализирует UI вкладки "Контакты" при первом обращении
func (ws *Workspace) initializeContactsUI() {
	if !ws.contactsInitialized {
		if ws.chatsUI == nil {
			ws.initializeChatsUI()
		}
		ws.contactsUI = contacts.New(ws.chatsUI)
		ws.contactsInitialized = true

		ws.contactsUI.SetWindow(ws.window)

		if ws.p2pNetwork != nil {
			p2pUI := ws.getP2PUIShared()
			ws.contactsUI.SetP2PService(p2pUI)
		}
	}
}

// initializeP2PUI инициализирует UI вкладки "P2P" при первом обращении
func (ws *Workspace) initializeP2PUI() {
	if !ws.p2pInitialized {
		if ws.chatsUI == nil {
			ws.initializeChatsUI()
		}

		ws.p2pUI = p2p.New(ws.chatsUI, func(contentType string) {
			ws.UpdateContent(contentType)
		})
		ws.p2pInitialized = true

		ws.p2pUI.SetWindow(ws.window)

		if ws.p2pNetwork != nil {
			p2pUI := ws.getP2PUIShared()
			ws.p2pUI.SetP2PService(p2pUI)
		}
	}
}
