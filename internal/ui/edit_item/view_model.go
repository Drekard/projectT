package edit_item

import (
	"context"
	"projectT/internal/services"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"strings"
)

type CreateItemViewModel struct {
	ID          int // ID элемента при редактировании (0 для создания)
	Title       string
	Description string
	Tags        string
	Images      []string
	Files       []string
	Links       []string
	ItemType    models.ItemType
	ContentMeta string
	ParentID    *int // ID родительской папки
	EditMode    bool // Режим редактирования
}

// Методы для работы с ViewModel
func (vm *CreateItemViewModel) Clear() {
	vm.Title = ""
	vm.Description = ""
	vm.Tags = ""
	vm.Images = []string{}
	vm.Files = []string{}
	vm.ItemType = models.ItemTypeElement
	vm.ContentMeta = ""
}

func NewCreateItemViewModel() *CreateItemViewModel {
	return &CreateItemViewModel{
		ID:       0, // 0 означает создание нового элемента
		ItemType: models.ItemTypeElement,
		Links:    []string{},
		ParentID: nil,   // Изначально без родителя
		EditMode: false, // По умолчанию режим создания
	}
}

// NewCreateItemViewModelForEdit создает ViewModel для редактирования элемента
func NewCreateItemViewModelForEdit(itemID int) (*CreateItemViewModel, error) {
	// Получаем элемент по ID
	item, err := queries.GetItemByID(itemID)
	if err != nil {
		return nil, err
	}

	// Загружаем теги для элемента
	tags, err := queries.GetTagsForItem(context.Background(), item.ID)
	if err != nil {
		return nil, err
	}

	var tagNames []string
	for _, tag := range tags {
		tagNames = append(tagNames, tag.Name)
	}

	// Используем ContentBlocksService для парсинга ContentMeta
	contentService := services.NewContentBlocksService()
	blocks, err := contentService.JSONToBlocks(item.ContentMeta)
	if err != nil {
		return nil, err
	}

	var images, files []string
	var links []string

	for _, block := range blocks {
		switch block.Type {
		case "image":
			images = append(images, block.OriginalName) // Используем OriginalName для отображения в интерфейсе
		case "file":
			files = append(files, block.OriginalName) // Используем OriginalName для отображения в интерфейсе
		case "link":
			links = append(links, block.Content)
		}
	}

	viewModel := &CreateItemViewModel{
		ID:          item.ID,
		Title:       item.Title,
		Description: item.Description,
		Tags:        strings.Join(tagNames, ", "),
		Images:      images,
		Files:       files,
		Links:       links,
		ItemType:    item.Type,
		ContentMeta: item.ContentMeta,
		ParentID:    item.ParentID,
		EditMode:    true, // Режим редактирования
	}

	return viewModel, nil
}
