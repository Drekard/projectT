// Package queries содержит SQL-запросы для работы с базой данных.
package queries

import (
	"database/sql"
	"errors"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
)

// CreateItemFile создаёт запись о файле элемента
func CreateItemFile(file *models.ItemFile) error {
	query := `
		INSERT INTO item_files (item_id, hash, file_path, size, mime_type, is_remote, source_peer_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(item_id, hash) DO UPDATE SET
			file_path = excluded.file_path,
			size = excluded.size,
			mime_type = excluded.mime_type,
			is_remote = excluded.is_remote,
			source_peer_id = excluded.source_peer_id
	`
	_, err := database.DB.Exec(query,
		file.ItemID, file.Hash, file.FilePath, file.Size, file.MimeType,
		file.IsRemote, file.SourcePeerID,
	)
	return err
}

// GetItemFile возвращает файл элемента по ID элемента
func GetItemFile(itemID int) (*models.ItemFile, error) {
	query := `
		SELECT item_id, hash, file_path, size, mime_type, is_remote, source_peer_id
		FROM item_files
		WHERE item_id = ?
		LIMIT 1
	`
	var file models.ItemFile
	var sourcePeerID sql.NullString

	err := database.DB.QueryRow(query, itemID).Scan(
		&file.ItemID, &file.Hash, &file.FilePath, &file.Size,
		&file.MimeType, &file.IsRemote, &sourcePeerID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("файл элемента не найден")
		}
		return nil, err
	}

	if sourcePeerID.Valid {
		file.SourcePeerID = sourcePeerID.String
	}

	return &file, nil
}

// GetFileByHash возвращает файл по хешу
func GetFileByHash(hash string) (*models.ItemFile, error) {
	query := `
		SELECT item_id, hash, file_path, size, mime_type, is_remote, source_peer_id
		FROM item_files
		WHERE hash = ?
		LIMIT 1
	`
	var file models.ItemFile
	var sourcePeerID sql.NullString

	err := database.DB.QueryRow(query, hash).Scan(
		&file.ItemID, &file.Hash, &file.FilePath, &file.Size,
		&file.MimeType, &file.IsRemote, &sourcePeerID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("файл не найден")
		}
		return nil, err
	}

	if sourcePeerID.Valid {
		file.SourcePeerID = sourcePeerID.String
	}

	return &file, nil
}

// GetFilesByItemID возвращает все файлы элемента (может быть несколько)
func GetFilesByItemID(itemID int) ([]*models.ItemFile, error) {
	query := `
		SELECT item_id, hash, file_path, size, mime_type, is_remote, source_peer_id
		FROM item_files
		WHERE item_id = ?
	`
	rows, err := database.DB.Query(query, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*models.ItemFile
	for rows.Next() {
		var file models.ItemFile
		var sourcePeerID sql.NullString

		err := rows.Scan(
			&file.ItemID, &file.Hash, &file.FilePath, &file.Size,
			&file.MimeType, &file.IsRemote, &sourcePeerID,
		)
		if err != nil {
			return nil, err
		}

		if sourcePeerID.Valid {
			file.SourcePeerID = sourcePeerID.String
		}

		files = append(files, &file)
	}

	return files, rows.Err()
}

// GetRemoteFilesByPeer возвращает все файлы от указанного пира
func GetRemoteFilesByPeer(sourcePeerID string) ([]*models.ItemFile, error) {
	query := `
		SELECT item_id, hash, file_path, size, mime_type, is_remote, source_peer_id
		FROM item_files
		WHERE source_peer_id = ? AND is_remote = 1
	`
	rows, err := database.DB.Query(query, sourcePeerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*models.ItemFile
	for rows.Next() {
		var file models.ItemFile
		var spID sql.NullString

		err := rows.Scan(
			&file.ItemID, &file.Hash, &file.FilePath, &file.Size,
			&file.MimeType, &file.IsRemote, &spID,
		)
		if err != nil {
			return nil, err
		}

		if spID.Valid {
			file.SourcePeerID = spID.String
		}

		files = append(files, &file)
	}

	return files, rows.Err()
}

// UpdateItemFile обновляет информацию о файле
func UpdateItemFile(file *models.ItemFile) error {
	query := `
		UPDATE item_files 
		SET file_path = ?, size = ?, mime_type = ?, is_remote = ?, source_peer_id = ?
		WHERE item_id = ? AND hash = ?
	`
	_, err := database.DB.Exec(query,
		file.FilePath, file.Size, file.MimeType, file.IsRemote,
		file.SourcePeerID, file.ItemID, file.Hash,
	)
	return err
}

// DeleteItemFile удаляет файл элемента
func DeleteItemFile(itemID int, hash string) error {
	_, err := database.DB.Exec(`DELETE FROM item_files WHERE item_id = ? AND hash = ?`, itemID, hash)
	return err
}

// DeleteFilesByItemID удаляет все файлы элемента
func DeleteFilesByItemID(itemID int) error {
	_, err := database.DB.Exec(`DELETE FROM item_files WHERE item_id = ?`, itemID)
	return err
}

// DeleteRemoteFilesByPeer удаляет все файлы от указанного пира
func DeleteRemoteFilesByPeer(sourcePeerID string) error {
	_, err := database.DB.Exec(`DELETE FROM item_files WHERE source_peer_id = ?`, sourcePeerID)
	return err
}

// ItemFileExists проверяет, существует ли файл с указанным хешем
func ItemFileExists(hash string) (bool, error) {
	var exists bool
	err := database.DB.QueryRow(`SELECT COUNT(*) > 0 FROM item_files WHERE hash = ?`, hash).Scan(&exists)
	return exists, err
}
