// Package queries содержит SQL-запросы для работы с базой данных.
package queries

import (
	"database/sql"
	"errors"
	"time"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

// CreateRemoteItem создаёт кэшированный элемент от другого пира
func CreateRemoteItem(item *models.RemoteItem) error {
	query := `
		INSERT INTO remote_items (source_peer_id, original_id, original_hash, title, description, 
		                          content_meta, signature, version, cached_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1, CURRENT_TIMESTAMP)
		ON CONFLICT(source_peer_id, original_hash) DO UPDATE SET
			title = excluded.title,
			description = excluded.description,
			content_meta = excluded.content_meta,
			signature = excluded.signature,
			version = version + 1,
			cached_at = CURRENT_TIMESTAMP
	`
	result, err := database.DB.Exec(query,
		item.SourcePeerID, item.OriginalID, item.OriginalHash,
		item.Title, item.Description, item.ContentMeta, item.Signature,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	item.ID = int(id)
	return nil
}

// GetRemoteItemByHash возвращает элемент по хешу и PeerID владельца
func GetRemoteItemByHash(sourcePeerID, originalHash string) (*models.RemoteItem, error) {
	query := `
		SELECT id, source_peer_id, original_id, original_hash, title, description, 
		       content_meta, signature, version, cached_at
		FROM remote_items
		WHERE source_peer_id = ? AND original_hash = ?
		LIMIT 1
	`
	var item models.RemoteItem
	var cachedAt string

	err := database.DB.QueryRow(query, sourcePeerID, originalHash).Scan(
		&item.ID, &item.SourcePeerID, &item.OriginalID, &item.OriginalHash,
		&item.Title, &item.Description, &item.ContentMeta, &item.Signature,
		&item.Version, &cachedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("элемент не найден")
		}
		return nil, err
	}

	item.CachedAt, _ = time.Parse("2006-01-02 15:04:05", cachedAt)
	return &item, nil
}

// GetRemoteItemsByPeer возвращает все кэшированные элементы от пира
func GetRemoteItemsByPeer(sourcePeerID string) ([]*models.RemoteItem, error) {
	query := `
		SELECT id, source_peer_id, original_id, original_hash, title, description, 
		       content_meta, signature, version, cached_at
		FROM remote_items
		WHERE source_peer_id = ?
		ORDER BY title
	`
	rows, err := database.DB.Query(query, sourcePeerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.RemoteItem
	for rows.Next() {
		var item models.RemoteItem
		var cachedAt string

		err := rows.Scan(
			&item.ID, &item.SourcePeerID, &item.OriginalID, &item.OriginalHash,
			&item.Title, &item.Description, &item.ContentMeta, &item.Signature,
			&item.Version, &cachedAt,
		)
		if err != nil {
			return nil, err
		}

		item.CachedAt, _ = time.Parse("2006-01-02 15:04:05", cachedAt)
		items = append(items, &item)
	}

	return items, rows.Err()
}

// GetRemoteItemByID возвращает элемент по локальному ID
func GetRemoteItemByID(id int) (*models.RemoteItem, error) {
	query := `
		SELECT id, source_peer_id, original_id, original_hash, title, description, 
		       content_meta, signature, version, cached_at
		FROM remote_items
		WHERE id = ?
		LIMIT 1
	`
	var item models.RemoteItem
	var cachedAt string

	err := database.DB.QueryRow(query, id).Scan(
		&item.ID, &item.SourcePeerID, &item.OriginalID, &item.OriginalHash,
		&item.Title, &item.Description, &item.ContentMeta, &item.Signature,
		&item.Version, &cachedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("элемент не найден")
		}
		return nil, err
	}

	item.CachedAt, _ = time.Parse("2006-01-02 15:04:05", cachedAt)
	return &item, nil
}

// UpdateRemoteItem обновляет кэшированный элемент
func UpdateRemoteItem(item *models.RemoteItem) error {
	query := `
		UPDATE remote_items 
		SET title = ?, description = ?, content_meta = ?, signature = ?, 
		    version = ?, cached_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := database.DB.Exec(query,
		item.Title, item.Description, item.ContentMeta,
		item.Signature, item.Version, item.ID,
	)
	return err
}

// DeleteRemoteItem удаляет кэшированный элемент
func DeleteRemoteItem(id int) error {
	_, err := database.DB.Exec(`DELETE FROM remote_items WHERE id = ?`, id)
	return err
}

// DeleteRemoteItemsByPeer удаляет все кэшированные элементы от пира
func DeleteRemoteItemsByPeer(sourcePeerID string) error {
	_, err := database.DB.Exec(`DELETE FROM remote_items WHERE source_peer_id = ?`, sourcePeerID)
	return err
}

// RemoteItemExists проверяет, существует ли элемент с указанным хешем
func RemoteItemExists(sourcePeerID, originalHash string) (bool, error) {
	var exists bool
	err := database.DB.QueryRow(`SELECT COUNT(*) > 0 FROM remote_items WHERE source_peer_id = ? AND original_hash = ?`, sourcePeerID, originalHash).Scan(&exists)
	return exists, err
}

// GetAllRemoteItemsCount возвращает общее количество кэшированных элементов
func GetAllRemoteItemsCount() (int, error) {
	var count int
	err := database.DB.QueryRow(`SELECT COUNT(*) FROM remote_items`).Scan(&count)
	return count, err
}
