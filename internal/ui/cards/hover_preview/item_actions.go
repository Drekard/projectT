package hover_preview

import (
	"fmt"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/storage/filesystem"
)

func (mm *MenuManager) deleteItem(item *models.Item) error {
	if item.Type == models.ItemTypeFolder {
		items, err := queries.GetItemsByParent(item.ID)
		if err != nil {
			return fmt.Errorf("ошибка получения вложенных элементов: %v", err)
		}

		for _, childItem := range items {
			if err := mm.deleteItem(childItem); err != nil {
				return fmt.Errorf("ошибка удаления вложенного элемента: %v", err)
			}
		}
	}

	files, err := queries.GetFilesByItemID(item.ID)
	if err == nil && len(files) > 0 {
		for _, file := range files {
			if err := filesystem.DeleteFile(file.Hash); err != nil {
				fmt.Printf("WARN: ошибка удаления файла %s: %v\n", file.Hash, err)
			}
		}
		if err := queries.DeleteFilesByItemID(item.ID); err != nil {
			fmt.Printf("WARN: ошибка удаления записей о файлах: %v\n", err)
		}
	}

	if err := queries.DeleteItem(item.ID); err != nil {
		return fmt.Errorf("ошибка удаления элемента: %v", err)
	}

	return nil
}

func (mm *MenuManager) saveItemToCollection(item *models.Item) error {
	err := queries.SetItemStatus(item.ID, models.ItemStatusSaved)
	if err != nil {
		return fmt.Errorf("ошибка изменения статуса элемента: %v", err)
	}

	return nil
}

func (mm *MenuManager) removeItemFromCollection(item *models.Item) error {
	err := queries.SetItemStatus(item.ID, models.ItemStatusPreview)
	if err != nil {
		return fmt.Errorf("ошибка изменения статуса элемента: %v", err)
	}

	return nil
}

// MoveItemToFolder перемещает элемент в указанную папку
func (mm *MenuManager) MoveItemToFolder(itemID int, folderID *int) error {
	item, err := queries.GetItemByID(itemID)
	if err != nil {
		return fmt.Errorf("ошибка получения элемента: %v", err)
	}

	item.ParentID = folderID

	if err := queries.UpdateItem(item); err != nil {
		return fmt.Errorf("ошибка обновления элемента: %v", err)
	}

	return nil
}

// formatPeerID форматирует PeerID для отображения
func formatPeerID(peerID string) string {
	if len(peerID) <= 16 {
		return peerID
	}
	return fmt.Sprintf("%s...%s", peerID[:8], peerID[len(peerID)-4:])
}

// getDescriptionForItem возвращает описание элемента
func getDescriptionForItem(item *models.Item) string {
	if item.Description == "" {
		return "--description not available--"
	}
	return item.Description
}

func getTitleForItem(item *models.Item) string {
	if item.Title == "" {
		return "--title not available--"
	}
	return "**" + item.Title + "**"
}
