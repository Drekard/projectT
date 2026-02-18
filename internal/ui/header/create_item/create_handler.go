package create_item

import (
	"context"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"projectT/internal/services"
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

// Контекст с таймаутом для операций с БД
var (
	dbTimeout = 30 * time.Second
)

// extractLinks извлекает все HTTP/HTTPS ссылки из текста
func extractLinks(text string) []string {
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

// removeLinksFromText удаляет ссылки из текста, оставляя только описание
func removeLinksFromText(text string, links []string) string {
	result := text
	for _, link := range links {
		result = strings.ReplaceAll(result, link, "")
	}
	result = strings.TrimSpace(result)

	// Удаляем лишние пробелы и переносы строк
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")

	return result
}

// processFileData обрабатывает файлы и возвращает блоки
func processFileData(selectedFiles *[]string, linkEntries []string) ([]Block, []string) {
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

// determineItemType определяет тип элемента на основе содержимого
func determineItemType(description string, blocks []Block) models.ItemType {
	// Все элементы, кроме папок, теперь являются элементами типа Element
	return models.ItemTypeElement
}

// createItemWithTransaction создает элемент в транзакции
func createItemWithTransaction(ctx context.Context, title, description string, itemType models.ItemType, contentMeta string, parentID *int) (*models.Item, error) {
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

// processTags обрабатывает теги для элемента
func processTags(ctx context.Context, itemID int, tagsInput string) error {
	fmt.Printf("Начинаем обработку тегов, itemID: %d, теги: '%s'\n", itemID, tagsInput)
	if tagsInput == "" {
		fmt.Println("Теги отсутствуют, возвращаемся")
		return nil
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
		return nil
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

// CreateItem создает новый элемент
func CreateItem(title, description, tags string, selectedFiles *[]string, linkEntries []string, parentID *int, itemType models.ItemType, modalWindow fyne.Window) error {
	fmt.Println("=== НАЧАЛО СОЗДАНИЯ ===")
	fmt.Printf("Title: '%s'\n", title)
	fmt.Printf("Description: '%s'\n", description)
	fmt.Printf("Tags: '%s'\n", tags)
	fmt.Printf("Number of selected files: %d\n", len(*selectedFiles))
	fmt.Printf("Number of link entries: %d\n", len(linkEntries))
	fmt.Printf("Parent ID: %v\n", parentID)

	// 1. Создаем экземпляр сервиса для обработки блоков контента
	contentService := services.NewContentBlocksService()

	// 2. Извлекаем ссылки из описания
	fmt.Println("Шаг 2: Извлечение ссылок из описания...")
	linksFromDescription := contentService.ExtractLinks(description)
	fmt.Printf("Найдено ссылок в описании: %d\n", len(linksFromDescription))
	for i, link := range linksFromDescription {
		fmt.Printf("  Ссылка %d: %s\n", i+1, link)
	}

	// 3. Обновляем описание, убирая из него ссылки
	fmt.Println("Шаг 3: Обновление описания, убираем ссылки...")
	updatedDescription := contentService.RemoveLinksFromText(description, linksFromDescription)
	fmt.Printf("Обновленное описание: '%s'\n", updatedDescription)

	// 4. Объединяем ссылки из описания с переданными ссылками
	fmt.Println("Шаг 4: Объединение ссылок...")
	allLinks := append(linksFromDescription, linkEntries...)
	fmt.Printf("Всего ссылок для обработки: %d\n", len(allLinks))

	// 5. Обрабатываем файлы и создаем блоки контента
	fmt.Println("Шаг 5: Обработка файлов и создание блоков контента...")
	blocks, processingErrors := contentService.ProcessFileData(selectedFiles, allLinks)
	if len(processingErrors) > 0 {
		fmt.Printf("Ошибки обработки: %v\n", processingErrors)
		// Показываем только первую ошибку
		dialog.ShowError(fmt.Errorf("Ошибка обработки файлов: %v", processingErrors[0]), modalWindow)
		return fmt.Errorf("ошибка обработки файлов: %v", processingErrors[0])
	}
	fmt.Printf("Создано блоков: %d\n", len(blocks))

	// 6. Используем переданный тип элемента (элемент или папка)
	fmt.Println("Шаг 6: Использование переданного типа элемента...")
	fmt.Printf("Переданный тип элемента: %s\n", itemType)

	// 7. Конвертируем блоки в JSON для сохранения
	fmt.Println("Шаг 7: Конвертация блоков в JSON...")
	contentMeta, err := contentService.BlocksToJSON(blocks)
	if err != nil {
		fmt.Printf("Ошибка сериализации: %v\n", err)
		dialog.ShowError(fmt.Errorf("Ошибка сериализации контента: %v", err), modalWindow)
		return fmt.Errorf("ошибка сериализации контента: %v", err)
	}
	if contentMeta != "" {
		fmt.Printf("JSON ContentMeta: %s\n", contentMeta)
	} else {
		fmt.Println("Блоки отсутствуют, ContentMeta будет пустым")
	}

	// 8. Создаем элемент в базе данных
	fmt.Println("Шаг 8: Создание элемента в базе данных...")
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	item, err := contentService.CreateItemWithTransaction(ctx, title, updatedDescription, itemType, contentMeta, parentID)
	if err != nil {
		fmt.Printf("ОШИБКА БД при создании: %v\n", err)
		dialog.ShowError(fmt.Errorf("Ошибка БД: %v", err), modalWindow)
		return err
	}
	fmt.Printf("Элемент успешно создан в БД, ID: %d\n", item.ID)

	// 9. Обрабатываем теги
	fmt.Println("Шаг 9: Обработка тегов...")
	if tags != "" {
		if err := contentService.ProcessTags(ctx, item.ID, tags); err != nil {
			fmt.Printf("ОШИБКА обработки тегов: %v\n", err)
			dialog.ShowError(fmt.Errorf("Ошибка обработки тегов: %v", err), modalWindow)
			// Не возвращаем ошибку, так как элемент уже создан
		} else {
			fmt.Println("Теги успешно обработаны")
		}
	} else {
		fmt.Println("Теги отсутствуют, обработка не требуется")
	}

	fmt.Printf("Элемент создан, ID: %d\n", item.ID)
	fmt.Println("=== УСПЕШНО СОЗДАНО ===")

	return nil
}
