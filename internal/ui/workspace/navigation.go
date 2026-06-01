package workspace

import (
	"fmt"
	"projectT/internal/services"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/workspace/saved/sorting"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// NavigateToPreviewFolder переходит в указанную папку для preview элементов
func (ws *Workspace) NavigateToPreviewFolder(folderID int) error {
	err := ws.navigationManager.GoToFolderInPath(folderID)
	if err != nil {
		return err
	}

	currentParentID := ws.navigationManager.GetCurrentFolderID()
	err = ws.gridManager.LoadItemsByParentWithSort(currentParentID)
	if err != nil {
		return err
	}

	ws.currentType = ContentType("folder_" + fmt.Sprintf("%d", currentParentID))
	return nil
}

// NavigateToFolder переходит в указанную папку
func (ws *Workspace) NavigateToFolder(folderID int) error {
	if ws.remoteProfilePeerID != "" {
		item, err := queries.GetItemByID(folderID)
		if err == nil && item != nil && item.OwnerType == "remote" {
			ws.NavigateToRemoteFolder(item.ElementUUID)
			return nil
		}
	}

	err := ws.navigationManager.GoToFolderInPath(folderID)
	if err != nil {
		return err
	}

	currentParentID := ws.navigationManager.GetCurrentFolderID()
	err = ws.gridManager.LoadItemsByParentWithSort(currentParentID)
	if err != nil {
		return err
	}

	ws.currentType = ContentType("folder_" + fmt.Sprintf("%d", currentParentID))
	return nil
}

// RefreshCurrentFolder обновляет содержимое текущей папки
func (ws *Workspace) RefreshCurrentFolder() error {
	currentParentID := ws.navigationManager.GetCurrentFolderID()
	err := ws.gridManager.LoadItemsByParentWithSort(currentParentID)
	if err != nil {
		return err
	}

	ws.currentType = ContentType("folder_" + fmt.Sprintf("%d", currentParentID))
	return nil
}

// SearchByTag выполняет поиск элементов по тегу
func (ws *Workspace) SearchByTag(tagName string) error {
	return ws.SearchItems(tagName)
}

// SetSearchQuery устанавливает значение в поисковую строку
func (ws *Workspace) SetSearchQuery(query string) error {
	return nil
}

// SetupNavigation настраивает навигацию
func (ws *Workspace) SetupNavigation(scroll *container.Scroll) {
	scroll.OnScrolled = func(pos fyne.Position) {
		if ws.gridManager != nil {
			ws.gridManager.UpdateLayout()
		}
	}
}

// OnSizeChanged обработчик изменения размера
func (ws *Workspace) OnSizeChanged(pos interface{}) {
	if ws.gridManager != nil {
		ws.gridManager.UpdateLayout()
	}
}

// SearchItems выполняет поиск элементов по запросу
func (ws *Workspace) SearchItems(query string) error {
	if query == "" {
		currentParentID := ws.navigationManager.GetCurrentFolderID()
		return ws.gridManager.LoadItemsByParentWithSort(currentParentID)
	}

	err := ws.gridManager.LoadItemsBySearchWithSort(query)
	if err != nil {
		return err
	}

	ws.currentType = ContentType("search_" + query)
	return nil
}

// ClearSearch очищает результаты поиска
func (ws *Workspace) ClearSearch() error {
	currentParentID := ws.navigationManager.GetCurrentFolderID()
	err := ws.gridManager.LoadItemsByParentWithSort(currentParentID)
	if err != nil {
		return err
	}

	ws.currentType = ContentType("folder_" + fmt.Sprintf("%d", currentParentID))
	return nil
}

// ApplyFilters применяет фильтры и обновляет сетку элементов
func (ws *Workspace) ApplyFilters(options services.FilterOptions) {
	ws.gridManager.SetSortOptions(&options)

	if options.TabMode == "all_items" {
		ws.showMode = "all_items"

		var allItems []*models.Item
		var err error

		if ws.currentPreviewMode == PreviewModePreview {
			allItems, err = itemsService.GetPreviewItemsWithoutParentFilter()
		} else {
			allItems, err = itemsService.GetSavedItemsWithoutParentFilter()
		}

		if err != nil {
			allItems = []*models.Item{}
		}

		sortedItems := ws.sortItems(allItems, &options)
		ws.gridManager.LoadItems(sortedItems)
	} else {
		ws.showMode = "current_folder"
		currentParentID := ws.navigationManager.GetCurrentFolderID()

		var err error
		if ws.currentPreviewMode == PreviewModePreview {
			err = ws.gridManager.LoadItemsByParentWithSort(currentParentID)
		} else {
			err = ws.gridManager.LoadItemsByParentWithSort(currentParentID)
		}

		if err != nil {
			_ = err
		}
	}
}

// sortItems сортирует элементы по заданным настройкам
func (ws *Workspace) sortItems(items []*models.Item, options *services.FilterOptions) []*models.Item {
	itemSorter := sorting.NewItemSorter()
	return itemSorter.SortItems(items, options)
}
