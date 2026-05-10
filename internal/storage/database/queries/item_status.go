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
