package workspace

import (
	"context"
	"log"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/workspace/profile"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	p2p_ui "projectT/internal/services/p2p/ui"

	"fyne.io/fyne/v2"
)

// createRemoteProfileContent создает контент для удалённого профиля
func (ws *Workspace) createRemoteProfileContent() fyne.CanvasObject {
	if ws.remoteProfileUI == nil || ws.remoteProfileUI.GetPeerID() != ws.remoteProfilePeerID {
		var p2pUI *p2p_ui.UIP2P
		if ws.p2pNetwork != nil {
			p2pUI = p2p_ui.NewUIP2P(ws.p2pNetwork)
		}
		ws.remoteProfileUI = profile.NewRemoteProfileUI(ws.remoteProfilePeerID, p2pUI)
		ws.remoteProfileUI.SetConfig(ws.config)
		ws.remoteProfileUI.SetOnSave(ws.onSave)
		ws.remoteProfileUI.RestoreLayoutMode()
		ws.remoteProfileUI.SetOnOpenElements(func() {
			ws.OpenRemoteSaved(ws.remoteProfilePeerID, ws.remoteProfileName, "")
		})
		ws.remoteProfileUI.SetLayoutChangeHandler(func() {
			ws.contentCache[ContentTypeRemoteProfile] = ws.remoteProfileUI.Container()
			ws.container.Objects = []fyne.CanvasObject{ws.contentCache[ContentTypeRemoteProfile]}
			ws.container.Refresh()
		})
	}
	return ws.remoteProfileUI.Container()
}

// createRemoteSavedContent создает контент для удалённых элементов
func (ws *Workspace) createRemoteSavedContent() fyne.CanvasObject {
	var items []*models.Item
	remoteItems, err := queries.GetRemoteItemsByPeer(ws.remoteProfilePeerID)
	if err == nil {
		if ws.remoteFolderUUID == "" {
			for _, item := range remoteItems {
				if item.ParentUUID == nil || *item.ParentUUID == "" {
					items = append(items, item)
				}
			}
		} else {
			for _, item := range remoteItems {
				if item.ParentUUID != nil && *item.ParentUUID == ws.remoteFolderUUID {
					items = append(items, item)
				}
			}
		}
	}

	ws.gridManager.LoadItemsWithoutCreateElement(items)

	// Всегда запрашиваем элементы у пира (не только если БД пуста)
	// Потому что в БД могут быть только pinned items, а не все элементы
	if ws.p2pNetwork != nil {
		go ws.requestRemoteFolderFromPeer(ws.remoteProfilePeerID, ws.remoteFolderUUID)
	}

	return ws.gridManager.GetContainer()
}

// requestRemoteFolderFromPeer запрашивает элементы папки у пира и сохраняет в БД
func (ws *Workspace) requestRemoteFolderFromPeer(peerID, folderUUID string) {
	peerIDObj, err := peer.Decode(peerID)
	if err != nil {
		return
	}

	ctx := context.Background()
	items, err := ws.p2pNetwork.ItemSync().RequestFolder(ctx, peerIDObj, folderUUID)
	if err != nil {
		return
	}

	if len(items) == 0 {
		return
	}

	for _, item := range items {
		item.OwnerType = "remote"
		item.SourcePeerID = &peerID
		_ = queries.CreateRemoteItem(item)
	}

	// После сохранения запрашиваем ВСЕ элементы из БД для обновления UI
	var allItems []*models.Item
	remoteItems, err := queries.GetRemoteItemsByPeer(peerID)
	if err == nil {
		if folderUUID == "" {
			for _, item := range remoteItems {
				if item.ParentUUID == nil || *item.ParentUUID == "" {
					allItems = append(allItems, item)
				}
			}
		} else {
			for _, item := range remoteItems {
				if item.ParentUUID != nil && *item.ParentUUID == folderUUID {
					allItems = append(allItems, item)
				}
			}
		}
	}

	fyne.CurrentApp().Driver().DoFromGoroutine(func() {
		ws.gridManager.LoadItemsWithoutCreateElement(allItems)
	}, false)
}

func (ws *Workspace) OpenRemoteProfile(peerID string) {
	ws.remoteProfilePeerID = peerID
	ws.remoteFolderUUID = ""
	ws.remoteFolderTitle = ""
	ws.remoteFolderPath = nil

	profile, err := queries.GetProfileByPeerID(peerID)
	if err == nil && profile != nil {
		ws.remoteProfileName = profile.Username
	}

	// Если профиль не найден или пустой — запрашиваем у пира
	if profile == nil || (profile.Username == "" && profile.ContentChar == "") {
		if ws.p2pNetwork != nil && ws.p2pNetwork.ProfileExchange() != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				peerIDObj, err := peer.Decode(peerID)
				if err != nil {
					return
				}

				_, err = ws.p2pNetwork.ProfileExchange().RequestFullProfile(ctx, peerIDObj)
				if err != nil {
					log.Printf("[RemoteProfile] ⚠️ Не удалось запросить профиль %s: %v", peerID[:min(10, len(peerID))], err)
					return
				}

				// После получения профиля обновляем UI
				fyne.CurrentApp().Driver().DoFromGoroutine(func() {
					updatedProfile, err := queries.GetProfileByPeerID(peerID)
					if err == nil && updatedProfile != nil {
						ws.remoteProfileName = updatedProfile.Username
					}
					if ws.remoteProfileUI != nil && ws.remoteProfileUI.GetPeerID() == peerID {
						ws.remoteProfileUI.Refresh()
					}
				}, false)
			}()
		}
	}

	ws.notifyRemoteModeChanged(true)
	ws.UpdateContent(string(ContentTypeRemoteProfile))
}

// OpenRemoteSaved открывает элементы удалённого пользователя
func (ws *Workspace) OpenRemoteSaved(peerID, peerName, folderUUID string) {
	ws.remoteProfilePeerID = peerID
	ws.remoteProfileName = peerName
	ws.remoteFolderUUID = folderUUID
	ws.remoteFolderTitle = ""
	ws.remoteFolderPath = nil

	if folderUUID != "" {
		item, err := queries.GetItemByElementUUID(folderUUID)
		if err == nil && item != nil {
			ws.remoteFolderTitle = item.Title
			ws.remoteFolderPath = []*models.Item{item}
		}
	}

	ws.notifyRemoteModeChanged(true)
	ws.UpdateContent(string(ContentTypeRemoteSaved))
}

// NavigateToRemoteFolder переходит в папку удалённого пользователя
func (ws *Workspace) NavigateToRemoteFolder(folderUUID string) {
	ws.remoteFolderUUID = folderUUID

	item, err := queries.GetItemByElementUUID(folderUUID)
	if err == nil && item != nil {
		ws.remoteFolderTitle = item.Title
		ws.remoteFolderPath = append(ws.remoteFolderPath, item)
	}

	ws.notifyRemoteModeChanged(true)
	ws.UpdateContent(string(ContentTypeRemoteSaved))
}

// OpenRemoteFolderFromChat открывает папку из чата
func (ws *Workspace) OpenRemoteFolderFromChat(peerID, folderUUID string) {
	ws.remoteProfilePeerID = peerID
	ws.remoteFolderUUID = folderUUID
	ws.remoteFolderTitle = ""
	ws.remoteFolderPath = nil

	profile, err := queries.GetProfileByPeerID(peerID)
	if err == nil && profile != nil {
		ws.remoteProfileName = profile.Username
	}

	item, err := queries.GetItemByElementUUID(folderUUID)
	if err == nil && item != nil {
		ws.remoteFolderTitle = item.Title
	}

	go ws.requestRemoteFolderFromPeer(peerID, folderUUID)

	ws.notifyRemoteModeChanged(true)
	ws.UpdateContent(string(ContentTypeRemoteSaved))
}

// ResetToLocalSaved сбрасывает remote режим и возвращает к локальной сортировке
func (ws *Workspace) ResetToLocalSaved() {
	ws.remoteProfilePeerID = ""
	ws.remoteFolderUUID = ""
	ws.remoteFolderTitle = ""
	ws.remoteProfileName = ""
	ws.remoteFolderPath = nil
	ws.remoteProfileUI = nil
	ws.navigationManager.Reset()
	ws.notifyRemoteModeChanged(false)
	ws.UpdateContent(string(ContentTypeSaved))
}

// GetRemoteProfilePeerID возвращает peerID текущего remote профиля
func (ws *Workspace) GetRemoteProfilePeerID() string {
	return ws.remoteProfilePeerID
}

// GetRemoteProfileName возвращает имя текущего remote профиля
func (ws *Workspace) GetRemoteProfileName() string {
	return ws.remoteProfileName
}

// GetRemoteFolderUUID возвращает текущий remote folder UUID
func (ws *Workspace) GetRemoteFolderUUID() string {
	return ws.remoteFolderUUID
}

// GetRemoteFolderTitle возвращает название текущей remote папки
func (ws *Workspace) GetRemoteFolderTitle() string {
	return ws.remoteFolderTitle
}

// SetOnRemoteModeChanged устанавливает callback при смене remote режима
func (ws *Workspace) SetOnRemoteModeChanged(callback func(isRemote bool, peerID, peerName string, path []*models.Item)) {
	ws.onRemoteModeChanged = callback
}

func (ws *Workspace) notifyRemoteModeChanged(isRemote bool) {
	if ws.onRemoteModeChanged != nil {
		ws.onRemoteModeChanged(isRemote, ws.remoteProfilePeerID, ws.remoteProfileName, ws.remoteFolderPath)
	}
}
