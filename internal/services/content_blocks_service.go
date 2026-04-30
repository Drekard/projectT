package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/storage/filesystem"
)

// Block represents a content block of an item
type Block struct {
	Type         string `json:"type"`
	Content      string `json:"content,omitempty"`
	FileHash     string `json:"file_hash,omitempty"`
	OriginalName string `json:"original_name,omitempty"`
	Extension    string `json:"extension,omitempty"`
	Description  string `json:"description,omitempty"`
}

// ContentBlocksService предоставляет методы для работы с блоками контента
type ContentBlocksService struct{}

// NewContentBlocksService создает новый экземпляр сервиса
func NewContentBlocksService() *ContentBlocksService {
	return &ContentBlocksService{}
}

// ProcessFileData обрабатывает файлы и возвращает блоки
func (s *ContentBlocksService) ProcessFileData(selectedFiles *[]string, linkEntries []string) ([]Block, []string) {
	var blocks []Block
	var errors []string

	// Вспомогательная функция для обработки одного файла
	processSingleFile := func(filepath, blockType string) (Block, error) {
		// Проверяем существование файла
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			return Block{}, fmt.Errorf("файл не существует: %s", filepath)
		}

		// Читаем файл
		fileBytes, err := os.ReadFile(filepath)
		if err != nil {
			return Block{}, fmt.Errorf("ошибка чтения файла %s: %v", filepath, err)
		}

		// Сохраняем в файловую систему
		fileData, err := filesystem.SaveFileWithOriginalName(fileBytes, filepath)
		if err != nil {
			return Block{}, fmt.Errorf("ошибка сохранения файла %s: %v", filepath, err)
		}

		return Block{
			Type:         blockType,
			FileHash:     fileData.Hash,
			OriginalName: path.Base(filepath),
			Extension:    strings.TrimPrefix(path.Ext(filepath), "."),
		}, nil
	}

	// Обрабатываем изображения (в данном случае все файлы из selectedFiles)
	for _, filepath := range *selectedFiles {
		ext := strings.ToLower(strings.TrimPrefix(path.Ext(filepath), "."))
		blockType := "file"

		// Изображения
		if ext == "jpg" || ext == "jpeg" || ext == "png" || ext == "gif" || ext == "bmp" || ext == "webp" {
			blockType = "image"
		}
		// Аудио
		if ext == "mp3" || ext == "wav" || ext == "ogg" || ext == "flac" || ext == "aac" || ext == "m4a" {
			blockType = "audio"
		}
		// Видео
		if ext == "mp4" || ext == "avi" || ext == "mkv" || ext == "mov" || ext == "webm" || ext == "wmv" {
			blockType = "video"
		}

		block, err := processSingleFile(filepath, blockType)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		blocks = append(blocks, block)
	}

	// Обрабатываем ссылки
	for _, link := range linkEntries {
		if link != "" {
			blocks = append(blocks, Block{
				Type:    "link",
				Content: link,
			})
		}
	}

	return blocks, errors
}

// BlocksToJSON конвертирует блоки в JSON строку
func (s *ContentBlocksService) BlocksToJSON(blocks []Block) (string, error) {
	if len(blocks) == 0 {
		return "", nil
	}

	contentBytes, err := json.Marshal(blocks)
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации контента: %v", err)
	}

	return string(contentBytes), nil
}

// JSONToBlocks конвертирует JSON строку в блоки
func (s *ContentBlocksService) JSONToBlocks(contentMeta string) ([]Block, error) {
	var blocks []Block

	if contentMeta == "" {
		return blocks, nil
	}

	if err := json.Unmarshal([]byte(contentMeta), &blocks); err != nil {
		return nil, fmt.Errorf("ошибка разбора JSON блоков: %v", err)
	}

	return blocks, nil
}

// ExtractFilesFromBlocks извлекает список файлов из блоков
func (s *ContentBlocksService) ExtractFilesFromBlocks(blocks []Block) []string {
	var files []string

	for _, block := range blocks {
		if block.FileHash != "" {
			files = append(files, block.FileHash)
		}
	}

	return files
}

// DetermineItemType определяет тип элемента на основе содержимого
func (s *ContentBlocksService) DetermineItemType(description string, blocks []Block) models.ItemType {
	// Все элементы, кроме папок, теперь являются элементами типа Element
	return models.ItemTypeElement
}

// CleanupOldFiles удаляет неиспользуемые файлы
func (s *ContentBlocksService) CleanupOldFiles(oldBlocks, newBlocks []Block) {
	// Создаем мапу новых хэшей
	newHashes := make(map[string]bool)
	for _, block := range newBlocks {
		if block.FileHash != "" {
			newHashes[block.FileHash] = true
		}
	}

	// Удаляем старые файлы, которых нет в новых
	/*for _, block := range oldBlocks { хз что это
		if block.FileHash != "" && !newHashes[block.FileHash] {
			if err := filesystem.DeleteFile(block.FileHash); err != nil {
				// Ignore error - file cleanup is not critical
			}
		}
	}*/
}

// CreateItemWithTransaction создает элемент в транзакции
func (s *ContentBlocksService) CreateItemWithTransaction(ctx context.Context, title, description string, itemType models.ItemType, contentMeta string, parentID *int) (*models.Item, error) {
	// Генерируем hash для уникальной идентификации содержимого (дедупликация)
	hash := filesystem.GenerateContentHash(title, description, contentMeta)

	// Генерируем elementUUID - уникальный идентификатор для P2P
	elementUUID := filesystem.GenerateElementUUID()

	// Создаем элемент
	item := &models.Item{
		ElementUUID: elementUUID,
		Hash:        hash,
		Type:        itemType,
		Title:       title,
		Description: description,
		ContentMeta: contentMeta,
		ParentID:    parentID,
	}

	if err := queries.CreateItem(item); err != nil {
		return nil, fmt.Errorf("ошибка создания элемента: %w", err)
	}

	return item, nil
}

// UpdateItemWithTransaction обновляет элемент в транзакции
func (s *ContentBlocksService) UpdateItemWithTransaction(ctx context.Context, itemID int, title, description string, itemType models.ItemType, contentMeta string, parentID *int) (*models.Item, []Block, error) {
	tx, err := queries.BeginTransaction(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Получаем текущий элемент напрямую в транзакции
	var item models.Item
	var currentContentMeta string
	var currentParentID sql.NullInt64

	query := `
		SELECT id, type, title, description, content_meta, parent_id, created_at, updated_at
		FROM items
		WHERE id = ?
	`
	err = tx.QueryRowContext(ctx, query, itemID).Scan(
		&item.ID, &item.Type, &item.Title, &item.Description, &currentContentMeta, &currentParentID, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка получения элемента: %w", err)
	}

	if currentParentID.Valid {
		parentIDValue := int(currentParentID.Int64)
		item.ParentID = &parentIDValue
	}

	// Сохраняем старые блоки для последующей очистки
	var oldBlocks []Block
	if currentContentMeta != "" {
		if err := json.Unmarshal([]byte(currentContentMeta), &oldBlocks); err != nil {
			_ = err // Ignore error
		}
	}

	// Генерируем новый hash
	newHash := filesystem.GenerateContentHash(title, description, contentMeta)

	// Обновляем элемент в транзакции
	updateQuery := `
		UPDATE items
		SET type = ?, title = ?, description = ?, content_meta = ?, parent_id = ?, hash = ?, updated_at = ?
		WHERE id = ?
	`
	_, err = tx.ExecContext(ctx, updateQuery, itemType, title, description, contentMeta, parentID, newHash, time.Now(), itemID)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка обновления элемента: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	// Обновляем item с новыми значениями
	item.Type = itemType
	item.Title = title
	item.Description = description
	item.ContentMeta = contentMeta
	item.ParentID = parentID
	item.Hash = newHash

	return &item, oldBlocks, nil
}

// SaveItemFiles сохраняет информацию о файлах элемента в таблицу item_files
func (s *ContentBlocksService) SaveItemFiles(itemID int, blocks []Block) error {
	for _, block := range blocks {
		if block.FileHash != "" {
			// Получаем информацию о файле
			fileInfo, err := filesystem.GetFileInfo(block.FileHash)
			if err != nil {
				continue
			}

			// Создаём запись в item_files
			itemFile := &models.ItemFile{
				ItemID:       itemID,
				Hash:         block.FileHash,
				FilePath:     fileInfo.Path,
				Size:         fileInfo.Size,
				MimeType:     fileInfo.MimeType,
				IsRemote:     false,
				SourcePeerID: "",
			}

			_ = queries.CreateItemFile(itemFile)
		}
	}
	return nil
}

// ProcessTags обрабатывает теги для элемента
func (s *ContentBlocksService) ProcessTags(ctx context.Context, itemID int, tagsInput string) error {
	if tagsInput == "" {
		// При пустых тегах удаляем все существующие теги для элемента
		return queries.ReplaceItemTags(ctx, itemID, []int{})
	}

	// Разбиваем и очищаем теги
	tagNames := strings.Split(tagsInput, ",")
	var cleanTagNames []string
	for _, name := range tagNames {
		name = strings.TrimSpace(name)
		if name != "" {
			cleanTagNames = append(cleanTagNames, name)
		}
	}

	if len(cleanTagNames) == 0 {
		// При отсутствии действительных тегов удаляем все существующие теги для элемента
		return queries.ReplaceItemTags(ctx, itemID, []int{})
	}

	// Получаем или создаем теги
	tagIDs, err := queries.GetOrCreateTags(ctx, cleanTagNames)
	if err != nil {
		return fmt.Errorf("ошибка обработки тегов: %w", err)
	}

	// Привязываем теги к элементу
	err = queries.ReplaceItemTags(ctx, itemID, tagIDs)
	if err != nil {
		return err
	}

	return nil
}

// ExtractLinks извлекает все HTTP/HTTPS ссылки из текста
func (s *ContentBlocksService) ExtractLinks(text string) []string {
	re := regexp.MustCompile(`https?://[^\s]+`)
	links := re.FindAllString(text, -1)

	// Очищаем найденные ссылки от лишних символов
	var cleanedLinks []string
	for _, link := range links {
		link = strings.TrimRight(link, ".,")
		cleanedLinks = append(cleanedLinks, link)
	}

	return cleanedLinks
}

// RemoveLinksFromText удаляет ссылки из текста, оставляя только описание
func (s *ContentBlocksService) RemoveLinksFromText(text string, links []string) string {
	result := text
	for _, link := range links {
		result = strings.ReplaceAll(result, link, "")
	}
	result = strings.TrimSpace(result)

	// Удаляем лишние пробелы и переносы строк
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")

	return result
}
