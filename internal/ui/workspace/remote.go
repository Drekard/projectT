package workspace

import (
	"context"
	"log"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/workspace/profile"

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
		ws.remoteProfileUI.SetOnOpenElements(func() {
			ws.OpenRemoteSaved(ws.remoteProfilePeerID, ws.remoteProfileName, "")
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
		log.Printf("[Workspace] Ошибка декодирования peerID: %v", err)
		return
	}

	ctx := context.Background()
	items, err := ws.p2pNetwork.ItemSync().RequestFolder(ctx, peerIDObj, folderUUID)
	if err != nil {
		log.Printf("[Workspace] Ошибка запроса элементов у пира: %v", err)
		return
	}

	if len(items) == 0 {
		log.Printf("[Workspace] Пир %s не вернул элементов для folderUUID=%s", peerID[:8], folderUUID)
		return
	}

	log.Printf("[Workspace] Получено %d элементов от пира %s", len(items), peerID[:8])

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

	log.Printf("[Workspace] Обновление UI: %d элементов из БД для peerID=%s", len(allItems), peerID[:8])
	ws.gridManager.LoadItemsWithoutCreateElement(allItems)
}

// OpenRemoteProfile открывает профиль удалённого пользователя
func (ws *Workspace) OpenRemoteProfile(peerID string) {
	ws.remoteProfilePeerID = peerID
	ws.remoteFolderUUID = ""
	ws.remoteFolderTitle = ""
	ws.remoteFolderPath = nil

	profile, err := queries.GetProfileByPeerID(peerID)
	if err == nil && profile != nil {
		ws.remoteProfileName = profile.Username
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
