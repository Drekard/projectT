package edit_item

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"projectT/internal/services"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/storage/filesystem"
)

// Block represents a content block of an item (дублируем определение для совместимости)
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

// updateViewModelFromUI обновляет ViewModel с данными из UI
func updateViewModelFromUI(viewModel *CreateItemViewModel, formWidgets *FormWidgets) {

	currentTab := formWidgets.Tabs.CurrentTab()
	if currentTab == nil {
		return
	}

	// Определяем тип элемента по активной вкладке
	switch currentTab.Text {
	case "Элемент":
		viewModel.ItemType = models.ItemTypeElement
	case "Папка":
		viewModel.ItemType = models.ItemTypeFolder
	}

	// Обновляем основные поля
	viewModel.Title = formWidgets.TitleEntry.Text
	viewModel.Description = formWidgets.DescriptionEntry.Text
	viewModel.Tags = formWidgets.TagsEntry.Text

	// Собираем ссылки
	var links []string
	for _, entry := range formWidgets.LinkEntries {
		if entry.Text != "" {
			links = append(links, entry.Text)
		}
	}
	viewModel.Links = links
}

// processFileData обрабатывает файлы и возвращает блоки
func processFileData(viewModel *CreateItemViewModel, modalWindow fyne.Window) ([]Block, []string) {
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

	// Обрабатываем изображения
	for _, filepath := range viewModel.Images {
		block, err := processSingleFile(filepath, "image")
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		blocks = append(blocks, block)
	}

	// Обрабатываем файлы
	for _, filepath := range viewModel.Files {
		block, err := processSingleFile(filepath, "file")
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		blocks = append(blocks, block)
	}

	// Обрабатываем ссылки
	for _, link := range viewModel.Links {
		if link != "" {
			blocks = append(blocks, Block{
				Type:    "link",
				Content: link,
			})
		}
	}

	return blocks, errors
}

// determineItemType определяет тип элемента на основе содержимого
func determineItemType(viewModel *CreateItemViewModel, blocks []Block) models.ItemType {
	if viewModel.ItemType == models.ItemTypeFolder {
		return models.ItemTypeFolder
	}

	// Все элементы, кроме папок, теперь являются элементами типа Element
	return models.ItemTypeElement
}

// createOrUpdateItem создает или обновляет элемент с транзакцией
func createOrUpdateItem(ctx context.Context, viewModel *CreateItemViewModel,
	itemType models.ItemType, contentMeta string) (*models.Item, []Block, error) {

	if viewModel.EditMode && viewModel.ID != 0 {
		// Режим редактирования
		return updateItemWithTransaction(ctx, viewModel, itemType, contentMeta)
	}

	// Режим создания
	return createItemWithTransaction(ctx, viewModel, itemType, contentMeta)
}

// updateItemWithTransaction обновляет элемент в транзакции
func updateItemWithTransaction(ctx context.Context, viewModel *CreateItemViewModel,
	itemType models.ItemType, contentMeta string) (*models.Item, []Block, error) {

	tx, err := queries.BeginTransaction(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback()

	// Получаем текущий элемент
	item, err := queries.GetItemByID(viewModel.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка получения элемента: %w", err)
	}

	// Сохраняем старые блоки для последующей очистки
	var oldBlocks []Block
	if item.ContentMeta != "" {
		if err := json.Unmarshal([]byte(item.ContentMeta), &oldBlocks); err != nil {
			// Логируем ошибку, но продолжаем
			fmt.Printf("WARN: ошибка разбора старых блоков: %v\n", err)
		}
	}

	// Обновляем элемент
	item.Type = itemType
	item.Title = viewModel.Title
	item.Description = viewModel.Description
	item.ContentMeta = contentMeta
	item.ParentID = viewModel.ParentID

	if err := queries.UpdateItem(item); err != nil {
		return nil, nil, fmt.Errorf("ошибка обновления элемента: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	return item, oldBlocks, nil
}

// createItemWithTransaction создает элемент в транзакции
func createItemWithTransaction(ctx context.Context, viewModel *CreateItemViewModel,
	itemType models.ItemType, contentMeta string) (*models.Item, []Block, error) {

	// Создаем элемент
	item := &models.Item{
		Type:        itemType,
		Title:       viewModel.Title,
		Description: viewModel.Description,
		ContentMeta: contentMeta,
		ParentID:    viewModel.ParentID,
	}

	if err := queries.CreateItem(item); err != nil {
		return nil, nil, fmt.Errorf("ошибка создания элемента: %w", err)
	}

	return item, []Block{}, nil
}

// processTags обрабатывает теги для элемента
func processTags(ctx context.Context, itemID int, tagsInput string,
	editMode bool, modalWindow fyne.Window) error {

	if tagsInput == "" {
		if editMode {
			// В режиме редактирования при пустых тегах удаляем все
			return queries.ReplaceItemTags(ctx, itemID, []int{})
		}
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

	if len(cleanTagNames) == 0 {
		if editMode {
			return queries.ReplaceItemTags(ctx, itemID, []int{})
		}
		return nil
	}

	// Получаем или создаем теги
	tagIDs, err := queries.GetOrCreateTags(ctx, cleanTagNames)
	if err != nil {
		return fmt.Errorf("ошибка обработки тегов: %w", err)
	}

	// Привязываем теги к элементу
	return queries.ReplaceItemTags(ctx, itemID, tagIDs)
}

// SaveItem сохраняет элемент (создает или обновляет)
func SaveItem(viewModel *CreateItemViewModel, formWidgets *FormWidgets, modalWindow fyne.Window) {
	fmt.Println("=== НАЧАЛО СОХРАНЕНИЯ ===")

	// 1. Обновляем ViewModel из UI
	updateViewModelFromUI(viewModel, formWidgets)
	fmt.Println("ViewModel обновлена из UI")

	// 2. Проверяем заголовок только при создании элемента, не при редактировании
	if viewModel.Title == "" && !viewModel.EditMode {
		dialog.ShowError(errors.New("Введите заголовок"), modalWindow)
		return
	}

	// 3. Создаем экземпляр сервиса для обработки блоков контента
	contentService := services.NewContentBlocksService()
	fmt.Println("Сервис обработки контента инициализирован")

	// 4. Если это режим редактирования, получаем старые блоки
	var oldBlocks []Block
	if viewModel.EditMode && viewModel.ID != 0 {
		var err error
		svcOldBlocks, err := contentService.JSONToBlocks(viewModel.ContentMeta)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Ошибка разбора старого контента: %v", err), modalWindow)
			return
		}
		// Преобразуем блоки из сервиса в локальный тип
		oldBlocks = convertServiceBlocksToLocal(svcOldBlocks)
		fmt.Printf("Получено %d старых блоков\n", len(oldBlocks))
	}

	// 5. Обрабатываем файлы и создаем новые блоки
	var allBlocks []Block

	// Если это режим редактирования, нам нужно определить, какие файлы остались, а какие были удалены
	if viewModel.EditMode && viewModel.ID != 0 {
		// Для редактирования: берем только те блоки, которые соответствуют текущему состоянию
		for _, oldBlock := range oldBlocks {
			shouldInclude := false

			switch oldBlock.Type {
			case "image":
				// Проверяем, есть ли это изображение в текущем списке
				for _, img := range viewModel.Images {
					if oldBlock.OriginalName == img {
						shouldInclude = true
						break
					}
				}
			case "file":
				// Проверяем, есть ли этот файл в текущем списке
				for _, file := range viewModel.Files {
					if oldBlock.OriginalName == file {
						shouldInclude = true
						break
					}
				}
			case "link":
				// Проверяем, есть ли эта ссылка в текущем списке
				for _, link := range viewModel.Links {
					if oldBlock.Content == link {
						shouldInclude = true
						break
					}
				}
			}

			if shouldInclude {
				// Добавляем старый блок без изменений
				allBlocks = append(allBlocks, oldBlock)
				fmt.Printf("Сохранен старый блок: %s (%s)\n", oldBlock.OriginalName, oldBlock.Type)
			} else {
				fmt.Printf("Блок будет удален: %s (%s)\n", oldBlock.OriginalName, oldBlock.Type)
			}
		}

		// Теперь обрабатываем новые файлы, которые пользователь добавил
		// Обрабатываем новые изображения
		var newImages []string
		for _, img := range viewModel.Images {
			existsInOld := false
			for _, oldBlock := range oldBlocks {
				if oldBlock.Type == "image" && oldBlock.OriginalName == img {
					existsInOld = true
					break
				}
			}
			if !existsInOld {
				newImages = append(newImages, img)
			}
		}
		fmt.Printf("Найдено %d новых изображений для обработки\n", len(newImages))

		if len(newImages) > 0 {
			imgBlocks, processingErrors := contentService.ProcessFileData(&newImages, []string{})
			if len(processingErrors) > 0 {
				// Показываем только первую ошибку
				dialog.ShowError(fmt.Errorf("Ошибка обработки файлов: %v", processingErrors[0]), modalWindow)
				return
			}
			// Преобразуем блоки из сервиса в локальный тип
			localImgBlocks := convertServiceBlocksToLocal(imgBlocks)
			allBlocks = append(allBlocks, localImgBlocks...)
			fmt.Printf("Обработано %d новых изображений\n", len(imgBlocks))
		}

		// Обрабатываем новые файлы
		var newFiles []string
		for _, file := range viewModel.Files {
			existsInOld := false
			for _, oldBlock := range oldBlocks {
				if oldBlock.Type == "file" && oldBlock.OriginalName == file {
					existsInOld = true
					break
				}
			}
			if !existsInOld {
				newFiles = append(newFiles, file)
			}
		}
		fmt.Printf("Найдено %d новых файлов для обработки\n", len(newFiles))

		if len(newFiles) > 0 {
			fileBlocks, fileErrors := contentService.ProcessFileData(&newFiles, []string{})
			if len(fileErrors) > 0 {
				// Показываем только первую ошибку
				dialog.ShowError(fmt.Errorf("Ошибка обработки файлов: %v", fileErrors[0]), modalWindow)
				return
			}
			// Преобразуем блоки из сервиса в локальный тип
			localFileBlocks := convertServiceBlocksToLocal(fileBlocks)
			allBlocks = append(allBlocks, localFileBlocks...)
			fmt.Printf("Обработано %d новых файлов\n", len(fileBlocks))
		}

		// Обрабатываем новые ссылки
		var newLinks []string
		for _, link := range viewModel.Links {
			existsInOld := false
			for _, oldBlock := range oldBlocks {
				if oldBlock.Type == "link" && oldBlock.Content == link {
					existsInOld = true
					break
				}
			}
			if !existsInOld {
				newLinks = append(newLinks, link)
			}
		}
		fmt.Printf("Найдено %d новых ссылок для обработки\n", len(newLinks))

		for _, link := range newLinks {
			allBlocks = append(allBlocks, Block{
				Type:    "link",
				Content: link,
			})
		}
	} else {
		// Для режима создания: обрабатываем все файлы как новые

		// Обрабатываем изображения
		if len(viewModel.Images) > 0 {
			imgBlocks, processingErrors := contentService.ProcessFileData(&viewModel.Images, []string{})
			if len(processingErrors) > 0 {
				// Показываем только первую ошибку
				dialog.ShowError(fmt.Errorf("Ошибка обработки файлов: %v", processingErrors[0]), modalWindow)
				return
			}
			// Преобразуем блоки из сервиса в локальный тип
			localImgBlocks := convertServiceBlocksToLocal(imgBlocks)
			allBlocks = append(allBlocks, localImgBlocks...)
			fmt.Printf("Обработано %d изображений\n", len(localImgBlocks))
		}

		// Обрабатываем файлы
		if len(viewModel.Files) > 0 {
			fileBlocks, fileErrors := contentService.ProcessFileData(&viewModel.Files, []string{})
			if len(fileErrors) > 0 {
				// Показываем только первую ошибку
				dialog.ShowError(fmt.Errorf("Ошибка обработки файлов: %v", fileErrors[0]), modalWindow)
				return
			}
			// Преобразуем блоки из сервиса в локальный тип
			localFileBlocks := convertServiceBlocksToLocal(fileBlocks)
			allBlocks = append(allBlocks, localFileBlocks...)
			fmt.Printf("Обработано %d файлов\n", len(localFileBlocks))
		}

		// Обрабатываем ссылки
		for _, link := range viewModel.Links {
			allBlocks = append(allBlocks, Block{
				Type:    "link",
				Content: link,
			})
		}
		fmt.Printf("Обработано %d ссылок\n", len(viewModel.Links))
	}

	fmt.Printf("Всего собрано %d блоков\n", len(allBlocks))

	// 6. Преобразуем локальные блоки в блоки сервиса для дальнейшей обработки
	serviceBlocks := convertLocalBlocksToService(allBlocks)
	fmt.Println("Блоки преобразованы для обработки сервисом")

	// 7. Определяем тип элемента на основе контента
	itemType := contentService.DetermineItemType(viewModel.Description, serviceBlocks)
	fmt.Printf("Определен тип элемента: %s\n", itemType)

	// 8. Конвертируем блоки в JSON для сохранения
	contentMeta, err := contentService.BlocksToJSON(serviceBlocks)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Ошибка сериализации контента: %v", err), modalWindow)
		return
	}
	fmt.Printf("Контент сериализован в JSON, длина: %d символов\n", len(contentMeta))

	// 9. Обработка тегов
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if viewModel.EditMode && viewModel.ID != 0 {
		// Режим редактирования - обновляем существующий элемент
		fmt.Printf("Режим редактирования, ID элемента: %d\n", viewModel.ID)

		// Обновляем элемент
		updatedItem, reallyOldBlocks, err := contentService.UpdateItemWithTransaction(ctx, viewModel.ID, viewModel.Title, viewModel.Description, itemType, contentMeta, viewModel.ParentID)
		if err != nil {
			fmt.Printf("ОШИБКА БД при обновлении: %v\n", err)
			dialog.ShowError(fmt.Errorf("Ошибка БД: %v", err), modalWindow)
			return
		}

		fmt.Printf("Элемент обновлен, ID: %d\n", updatedItem.ID)

		// Обрабатываем теги
		if err := contentService.ProcessTags(ctx, updatedItem.ID, viewModel.Tags); err != nil {
			dialog.ShowError(fmt.Errorf("Ошибка обработки тегов: %v", err), modalWindow)
			// Не возвращаем ошибку, так как элемент уже обновлен
		} else {
			fmt.Println("Теги успешно обработаны")
		}

		// Очищаем старые файлы, которые больше не используются
		newServiceBlocks, _ := contentService.JSONToBlocks(contentMeta)
		// Преобразуем блоки для очистки
		localReallyOldBlocks := convertServiceBlocksToLocal(reallyOldBlocks)
		localNewBlocks := convertServiceBlocksToLocal(newServiceBlocks)
		cleanupOldFiles(localReallyOldBlocks, localNewBlocks)
		fmt.Println("Старые файлы очищены")

	} else {
		// Режим создания - создаем новый элемент
		fmt.Println("Режим создания нового элемента")

		item, err := contentService.CreateItemWithTransaction(ctx, viewModel.Title, viewModel.Description, itemType, contentMeta, viewModel.ParentID)
		if err != nil {
			fmt.Printf("ОШИБКА БД при создании: %v\n", err)
			dialog.ShowError(fmt.Errorf("Ошибка БД: %v", err), modalWindow)
			return
		}

		fmt.Printf("Элемент создан, ID: %d\n", item.ID)

		// Обрабатываем теги
		if viewModel.Tags != "" {
			if err := contentService.ProcessTags(ctx, item.ID, viewModel.Tags); err != nil {
				dialog.ShowError(fmt.Errorf("Ошибка обработки тегов: %v", err), modalWindow)
				// Не возвращаем ошибку, так как элемент уже создан
			} else {
				fmt.Println("Теги успешно обработаны")
			}
		}
	}

	fmt.Println("=== УСПЕШНО СОХРАНЕНО ===")
	modalWindow.Close()
}

// convertServiceBlocksToLocal преобразует блоки из сервиса в локальный тип
func convertServiceBlocksToLocal(serviceBlocks []services.Block) []Block {
	localBlocks := make([]Block, len(serviceBlocks))
	for i, svcBlock := range serviceBlocks {
		localBlocks[i] = Block{
			Type:         svcBlock.Type,
			Content:      svcBlock.Content,
			FileHash:     svcBlock.FileHash,
			OriginalName: svcBlock.OriginalName,
			Extension:    svcBlock.Extension,
			Description:  svcBlock.Description,
		}
	}
	return localBlocks
}

// convertLocalBlocksToService преобразует локальные блоки в блоки сервиса
func convertLocalBlocksToService(localBlocks []Block) []services.Block {
	serviceBlocks := make([]services.Block, len(localBlocks))
	for i, localBlock := range localBlocks {
		serviceBlocks[i] = services.Block{
			Type:         localBlock.Type,
			Content:      localBlock.Content,
			FileHash:     localBlock.FileHash,
			OriginalName: localBlock.OriginalName,
			Extension:    localBlock.Extension,
			Description:  localBlock.Description,
		}
	}
	return serviceBlocks
}

// cleanupOldFiles удаляет неиспользуемые файлы
func cleanupOldFiles(oldBlocks, newBlocks []Block) {
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
