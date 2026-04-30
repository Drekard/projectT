package edit_item

import (
	"context"
	"errors"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"projectT/internal/services"
	"projectT/internal/storage/database/models"
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

	currentTab := formWidgets.Tabs.Selected()
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

// SaveItem сохраняет элемент (создает или обновляет)
func SaveItem(viewModel *CreateItemViewModel, formWidgets *FormWidgets, parentWindow fyne.Window) {

	// 1. Обновляем ViewModel из UI
	updateViewModelFromUI(viewModel, formWidgets)

	// 2. Проверяем заголовок только при создании элемента, не при редактировании
	if viewModel.Title == "" && !viewModel.EditMode {
		dialog.ShowError(errors.New("введите заголовок"), parentWindow)
		return
	}

	// 3. Создаем экземпляр сервиса для обработки блоков контента
	contentService := services.NewContentBlocksService()

	// 4. Если это режим редактирования, получаем старые блоки
	var oldBlocks []Block
	if viewModel.EditMode && viewModel.ID != 0 {
		var err error
		svcOldBlocks, err := contentService.JSONToBlocks(viewModel.ContentMeta)
		if err != nil {
			dialog.ShowError(fmt.Errorf("ошибка разбора старого контента: %v", err), parentWindow)
			return
		}
		// Преобразуем блоки из сервиса в локальный тип
		oldBlocks = convertServiceBlocksToLocal(svcOldBlocks)
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
			case "file", "audio", "video":
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
			}
		}

		// теперь обрабатываем новые файлы, которые пользователь добавил
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

		if len(newImages) > 0 {
			imgBlocks, processingErrors := contentService.ProcessFileData(&newImages, []string{})
			if len(processingErrors) > 0 {
				// Показываем только первую ошибку
				dialog.ShowError(fmt.Errorf("ошибка обработки файлов: %v", processingErrors[0]), parentWindow)
				return
			}
			// Преобразуем блоки из сервиса в локальный тип
			localImgBlocks := convertServiceBlocksToLocal(imgBlocks)
			allBlocks = append(allBlocks, localImgBlocks...)
		}

		// Обрабатываем новые файлы (включая audio и video)
		var newFiles []string
		for _, file := range viewModel.Files {
			existsInOld := false
			for _, oldBlock := range oldBlocks {
				if (oldBlock.Type == "file" || oldBlock.Type == "audio" || oldBlock.Type == "video") && oldBlock.OriginalName == file {
					existsInOld = true
					break
				}
			}
			if !existsInOld {
				newFiles = append(newFiles, file)
			}
		}

		if len(newFiles) > 0 {
			fileBlocks, fileErrors := contentService.ProcessFileData(&newFiles, []string{})
			if len(fileErrors) > 0 {
				// Показываем только первую ошибку
				dialog.ShowError(fmt.Errorf("ошибка обработки файлов: %v", fileErrors[0]), parentWindow)
				return
			}
			// Преобразуем блоки из сервиса в локальный тип
			localFileBlocks := convertServiceBlocksToLocal(fileBlocks)
			allBlocks = append(allBlocks, localFileBlocks...)
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
				dialog.ShowError(fmt.Errorf("ошибка обработки файлов: %v", processingErrors[0]), parentWindow)
				return
			}
			// Преобразуем блоки из сервиса в локальный тип
			localImgBlocks := convertServiceBlocksToLocal(imgBlocks)
			allBlocks = append(allBlocks, localImgBlocks...)
		}

		// Обрабатываем файлы
		if len(viewModel.Files) > 0 {
			fileBlocks, fileErrors := contentService.ProcessFileData(&viewModel.Files, []string{})
			if len(fileErrors) > 0 {
				// Показываем только первую ошибку
				dialog.ShowError(fmt.Errorf("ошибка обработки файлов: %v", fileErrors[0]), parentWindow)
				return
			}
			// Преобразуем блоки из сервиса в локальный тип
			localFileBlocks := convertServiceBlocksToLocal(fileBlocks)
			allBlocks = append(allBlocks, localFileBlocks...)
		}

		// Обрабатываем ссылки
		for _, link := range viewModel.Links {
			allBlocks = append(allBlocks, Block{
				Type:    "link",
				Content: link,
			})
		}
	}

	// 6. Преобразуем локальные блоки в блоки сервиса для дальнейшей обработки
	serviceBlocks := convertLocalBlocksToService(allBlocks)

	// 7. Используем тип элемента, выбранный пользователем во вкладках
	itemType := viewModel.ItemType

	// 8. Конвертируем блоки в JSON для сохранения
	contentMeta, err := contentService.BlocksToJSON(serviceBlocks)
	if err != nil {
		dialog.ShowError(fmt.Errorf("ошибка сериализации контента: %v", err), parentWindow)
		return
	}

	// 9. Обработка тегов
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if viewModel.EditMode && viewModel.ID != 0 {
		// Режим редактирования - обновляем существующий элемент

		// Обновляем элемент
		updatedItem, reallyOldBlocks, err := contentService.UpdateItemWithTransaction(ctx, viewModel.ID, viewModel.Title, viewModel.Description, itemType, contentMeta, viewModel.ParentID)
		if err != nil {
			dialog.ShowError(fmt.Errorf("ошибка БД: %v", err), parentWindow)
			return
		}

		// Обрабатываем теги
		if err := contentService.ProcessTags(ctx, updatedItem.ID, viewModel.Tags); err != nil {
			dialog.ShowError(fmt.Errorf("ошибка обработки тегов: %v", err), parentWindow)
		}

		// Очищаем старые файлы, которые больше не используются
		newServiceBlocks, _ := contentService.JSONToBlocks(contentMeta)
		// Преобразуем блоки для очистки
		localReallyOldBlocks := convertServiceBlocksToLocal(reallyOldBlocks)
		localNewBlocks := convertServiceBlocksToLocal(newServiceBlocks)
		cleanupOldFiles(localReallyOldBlocks, localNewBlocks)

	} else {
		// Режим создания - создаем новый элемент

		item, err := contentService.CreateItemWithTransaction(ctx, viewModel.Title, viewModel.Description, itemType, contentMeta, viewModel.ParentID)
		if err != nil {
			dialog.ShowError(fmt.Errorf("ошибка БД: %v", err), parentWindow)
			return
		}

		// Обрабатываем теги
		if viewModel.Tags != "" {
			if err := contentService.ProcessTags(ctx, item.ID, viewModel.Tags); err != nil {
				dialog.ShowError(fmt.Errorf("ошибка обработки тегов: %v", err), parentWindow)
			}
		}
	}

	// Закрываем диалог, если функция закрытия определена
	if formWidgets.CloseDialog != nil {
		formWidgets.CloseDialog()
	}
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
			// ✅ Удаляем файл с диска
			if err := filesystem.DeleteFile(block.FileHash); err != nil {
				fmt.Printf("WARN: ошибка удаления файла %s: %v\n", block.FileHash, err)
			}
		}
	}
}
