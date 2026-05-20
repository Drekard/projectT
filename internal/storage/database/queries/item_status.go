package queries

import (
	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

// SetItemStatus обновляет статус элемента
func SetItemStatus(itemID int, status models.ItemStatus) error {
	query := `UPDATE items SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := database.DB.Exec(query, status, itemID)
	return err
}

// SetItemVisibility обновляет видимость элемента (public/private)
func SetItemVisibility(itemID int, visibility models.ItemVisibility) error {
	query := `UPDATE items SET visibility = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := database.DB.Exec(query, visibility, itemID)
	return err
}

// ToggleItemVisibility переключает видимость элемента между public и private
func ToggleItemVisibility(itemID int) (models.ItemVisibility, error) {
	item, err := GetItemByID(itemID)
	if err != nil {
		return "", err
	}

	newVisibility := models.ItemVisibilityPublic
	if item.IsPublic() {
		newVisibility = models.ItemVisibilityPrivate
	}

	if err := SetItemVisibility(itemID, newVisibility); err != nil {
		return "", err
	}

	return newVisibility, nil
}
