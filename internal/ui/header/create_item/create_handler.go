package create_item

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"projectT/internal/services"
	"projectT/internal/storage/database/models"
)

// CreateItem создает новый элемент
func CreateItem(title, description, tags string, selectedFiles *[]string, linkEntries []string, parentID *int, itemType models.ItemType, modalWindow fyne.Window) error {
	// 1. Создаем экземпляр сервиса для обработки блоков контента
	contentService := services.NewContentBlocksService()

	// 2. Извлекаем ссылки из описания
	linksFromDescription := contentService.ExtractLinks(description)

	// 3. Обновляем описание, убирая из него ссылки
	updatedDescription := contentService.RemoveLinksFromText(description, linksFromDescription)

	// 4. Объединяем ссылки из описания с переданными ссылками
	allLinks := append(linksFromDescription, linkEntries...)

	// 5. Обрабатываем файлы и создаем блоки контента
	blocks, processingErrors := contentService.ProcessFileData(selectedFiles, allLinks)
	if len(processingErrors) > 0 {
		dialog.ShowError(fmt.Errorf("Ошибка обработки файлов: %v", processingErrors[0]), modalWindow)
		return fmt.Errorf("ошибка обработки файлов: %v", processingErrors[0])
	}

	// 6. Конвертируем блоки в JSON для сохранения
	contentMeta, err := contentService.BlocksToJSON(blocks)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Ошибка сериализации контента: %v", err), modalWindow)
		return fmt.Errorf("ошибка сериализации контента: %v", err)
	}

	// 7. Создаем элемент в базе данных
	ctx := context.Background()
	item, err := contentService.CreateItemWithTransaction(ctx, title, updatedDescription, itemType, contentMeta, parentID)
	if err != nil {
		return err
	}

	// 8. Обрабатываем теги
	if tags != "" {
		_ = contentService.ProcessTags(ctx, item.ID, tags)
	}

	return nil
}
