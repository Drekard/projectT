package queries

import (
	"encoding/json"
	"projectT/internal/storage/database"
)

// PinItem закрепляет элемент
func PinItem(itemID int) error {
	query := `INSERT INTO pinned_items (item_id) VALUES (?)`
	_, err := database.DB.Exec(query, itemID)
	if err != nil {
		return err
	}

	return updatePinnedUUIDs()
}

// UnpinItem открепляет элемент
func UnpinItem(itemID int) error {
	query := `DELETE FROM pinned_items WHERE item_id = ?`
	_, err := database.DB.Exec(query, itemID)
	if err != nil {
		return err
	}

	return updatePinnedUUIDs()
}

// updatePinnedUUIDs обновляет pinned_uuids в profiles на основе pinned_items
func updatePinnedUUIDs() error {
	rows, err := database.DB.Query(`
		SELECT i.element_uuid 
		FROM pinned_items pi
		JOIN items i ON pi.item_id = i.id
		WHERE i.owner_type = 'local'
	`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	var uuids []string
	for rows.Next() {
		var uuid string
		if err := rows.Scan(&uuid); err != nil {
			return err
		}
		uuids = append(uuids, uuid)
	}

	uuidsJSON, err := json.Marshal(uuids)
	if err != nil {
		return err
	}

	_, err = database.DB.Exec(`UPDATE profiles SET pinned_uuids = ? WHERE owner_type = 'local'`, string(uuidsJSON))
	return err
}

// IsItemPinned проверяет, закреплен ли элемент
func IsItemPinned(itemID int) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM pinned_items WHERE item_id = ?`
	err := database.DB.QueryRow(query, itemID).Scan(&count)
	return count > 0, err
}
