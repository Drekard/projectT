package workspace

import (
	"projectT/internal/storage/database/models"

	"fyne.io/fyne/v2"
)

// loadSavedContent загружает сохраненные элементы
func (ws *Workspace) loadSavedContent() {
	if ws.gridManager == nil {
		return
	}
	items, err := itemsService.GetSavedItemsByParent(0)
	if err != nil {
		items = []*models.Item{}
	}
	ws.gridManager.LoadItems(items)
	ws.gridManager.SetCurrentParentID(0)
}

// loadPreviewContent загружает preview элементы
func (ws *Workspace) loadPreviewContent() {
	items, err := itemsService.GetPreviewItemsByParent(0)
	if err != nil {
		items = []*models.Item{}
	}
	ws.gridManager.LoadItems(items)
	ws.gridManager.SetCurrentParentID(0)
}

// createSavedContent создает контент для "Сохраненного"
func (ws *Workspace) createSavedContent() fyne.CanvasObject {
	if ws.gridManager == nil {
		return nil
	}
	ws.loadSavedContent()
	return ws.gridManager.GetContainer()
}

// createPreviewContent создает контент для "Загруженного"
func (ws *Workspace) createPreviewContent() fyne.CanvasObject {
	ws.loadPreviewContent()
	return ws.gridManager.GetContainer()
}

// createProfileContent создает контент для профиля
func (ws *Workspace) createProfileContent() fyne.CanvasObject {
	return ws.profileUI.CreateView()
}

// createTagsContent создает контент для тегов
func (ws *Workspace) createTagsContent() fyne.CanvasObject {
	ws.initializeTagsUI()
	ws.tagsUI.Refresh()
	return ws.tagsUI.GetContent()
}

// createChatsContent создает контент для чатов
func (ws *Workspace) createChatsContent() fyne.CanvasObject {
	ws.initializeChatsUI()
	if ws.chatsUI != nil {
		ws.chatsUI.Refresh()
	}
	return ws.chatsUI.CreateView()
}

// createContactsContent создает контент для вкладки "Контакты"
func (ws *Workspace) createContactsContent() fyne.CanvasObject {
	ws.initializeContactsUI()
	return ws.contactsUI.GetContent()
}

// createP2PContent создает контент для вкладки "P2P"
func (ws *Workspace) createP2PContent() fyne.CanvasObject {
	ws.initializeP2PUI()
	return ws.p2pUI.GetContent()
}
