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
	fmt.Printf("Начинаем обработку файлов, количество выбранных файлов: %d\n", len(*selectedFiles))
	var blocks []Block
	var errors []string

	// Вспомогательная функция для обработки одного файла
	processSingleFile := func(filepath, blockType string) (Block, error) {
		fmt.Printf("Обработка файла: %s, тип: %s\n", filepath, blockType)
		// Проверяем существование файла
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			fmt.Printf("Файл не существует: %s\n", filepath)
			return Block{}, fmt.Errorf("файл не существует: %s", filepath)
		}

		// Читаем файл
		fileBytes, err := os.ReadFile(filepath)
		if err != nil {
			fmt.Printf("Ошибка чтения файла %s: %v\n", filepath, err)
			return Block{}, fmt.Errorf("ошибка чтения файла %s: %v", filepath, err)
		}
		fmt.Printf("Файл успешно прочитан, размер: %d байт\n", len(fileBytes))

		// Сохраняем в файловую систему
		fileData, err := filesystem.SaveFileWithOriginalName(fileBytes, filepath)
		if err != nil {
			fmt.Printf("Ошибка сохранения файла %s: %v\n", filepath, err)
			return Block{}, fmt.Errorf("ошибка сохранения файла %s: %v", filepath, err)
		}
		fmt.Printf("Файл успешно сохранен с хешем: %s\n", fileData.Hash)

		return Block{
			Type:         blockType,
			FileHash:     fileData.Hash,
			OriginalName: path.Base(filepath),
			Extension:    strings.TrimPrefix(path.Ext(filepath), "."),
		}, nil
	}

	// Обрабатываем изображения (в данном случае все файлы из selectedFiles)
	for i, filepath := range *selectedFiles {
		fmt.Printf("Обрабатываем файл %d/%d: %s\n", i+1, len(*selectedFiles), filepath)
		ext := strings.ToLower(strings.TrimPrefix(path.Ext(filepath), "."))
		blockType := "file"
		if ext == "jpg" || ext == "jpeg" || ext == "png" || ext == "gif" || ext == "bmp" {
			blockType = "image"
		}
		fmt.Printf("Расширение файла: %s, определенный тип: %s\n", ext, blockType)

		block, err := processSingleFile(filepath, blockType)
		if err != nil {
			fmt.Printf("Ошибка при обработке файла %s: %v\n", filepath, err)
			errors = append(errors, err.Error())
			continue
		}
		blocks = append(blocks, block)
		fmt.Printf("Файл успешно обработан, добавлен блок типа: %s\n", block.Type)
	}

	fmt.Printf("Начинаем обработку ссылок, количество ссылок: %d\n", len(linkEntries))
	// Обрабатываем ссылки
	for i, link := range linkEntries {
		fmt.Printf("Обрабатываем ссылку %d/%d: %s\n", i+1, len(linkEntries), link)
		if link != "" {
			blocks = append(blocks, Block{
				Type:    "link",
				Content: link,
			})
			fmt.Printf("Ссылка добавлена как блок\n")
		} else {
			fmt.Printf("Ссылка пустая, пропускаем\n")
		}
	}

	fmt.Printf("Обработка файлов завершена. Создано блоков: %d, ошибок: %d\n", len(blocks), len(errors))
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
	for _, block := range oldBlocks {
		if block.FileHash != "" && !newHashes[block.FileHash] {
			if err := filesystem.DeleteFile(block.FileHash); err != nil {
				// Логируем, но не прерываем
				fmt.Printf("WARN: ошибка удаления файла %s: %v\n", block.FileHash, err)
			}
		}
	}
}

// CreateItemWithTransaction создает элемент в транзакции
func (s *ContentBlocksService) CreateItemWithTransaction(ctx context.Context, title, description string, itemType models.ItemType, contentMeta string, parentID *int) (*models.Item, error) {
	fmt.Println("Начинаем создание элемента в базе данных...")
	fmt.Printf("Title: %s\n", title)
	fmt.Printf("Description: %s\n", description)
	fmt.Printf("ItemType: %s\n", itemType)
	fmt.Printf("ContentMeta length: %d\n", len(contentMeta))
	fmt.Printf("ParentID: %v\n", parentID)

	// Создаем элемент
	item := &models.Item{
		Type:        itemType,
		Title:       title,
		Description: description,
		ContentMeta: contentMeta,
		ParentID:    parentID,
	}

	fmt.Println("Вызываем queries.CreateItem...")
	if err := queries.CreateItem(item); err != nil {
		fmt.Printf("Ошибка при создании элемента: %v\n", err)
		return nil, fmt.Errorf("ошибка создания элемента: %w", err)
	}
	fmt.Println("Элемент успешно создан в базе данных")

	return item, nil
}

// UpdateItemWithTransaction обновляет элемент в транзакции
func (s *ContentBlocksService) UpdateItemWithTransaction(ctx context.Context, itemID int, title, description string, itemType models.ItemType, contentMeta string, parentID *int) (*models.Item, []Block, error) {
	tx, err := queries.BeginTransaction(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback()

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
			// Логируем ошибку, но продолжаем
			fmt.Printf("WARN: ошибка разбора старых блоков: %v\n", err)
		}
	}

	// Обновляем элемент в транзакции
	updateQuery := `
		UPDATE items
		SET type = ?, title = ?, description = ?, content_meta = ?, parent_id = ?, updated_at = ?
		WHERE id = ?
	`
	_, err = tx.ExecContext(ctx, updateQuery, itemType, title, description, contentMeta, parentID, time.Now(), itemID)
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

	return &item, oldBlocks, nil
}

// ProcessTags обрабатывает теги для элемента
func (s *ContentBlocksService) ProcessTags(ctx context.Context, itemID int, tagsInput string) error {
	fmt.Printf("Начинаем обработку тегов, itemID: %d, теги: '%s'\n", itemID, tagsInput)
	if tagsInput == "" {
		fmt.Println("Теги отсутствуют, возвращаемся")
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

	fmt.Printf("Очищенные теги: %v\n", cleanTagNames)

	if len(cleanTagNames) == 0 {
		fmt.Println("Нет действительных тегов, возвращаемся")
		// При отсутствии действительных тегов удаляем все существующие теги для элемента
		return queries.ReplaceItemTags(ctx, itemID, []int{})
	}

	fmt.Println("Вызываем queries.GetOrCreateTags...")
	// Получаем или создаем теги
	tagIDs, err := queries.GetOrCreateTags(ctx, cleanTagNames)
	if err != nil {
		fmt.Printf("Ошибка при получении/создании тегов: %v\n", err)
		return fmt.Errorf("ошибка обработки тегов: %w", err)
	}
	fmt.Printf("Получены тег ID: %v\n", tagIDs)

	fmt.Println("Вызываем queries.ReplaceItemTags...")
	// Привязываем теги к элементу
	err = queries.ReplaceItemTags(ctx, itemID, tagIDs)
	if err != nil {
		fmt.Printf("Ошибка при привязке тегов к элементу: %v\n", err)
		return err
	}
	fmt.Println("Теги успешно привязаны к элементу")

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
