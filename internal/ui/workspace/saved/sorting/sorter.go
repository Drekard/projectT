package sorting

import (
	"sort"
	"strings"

	"projectT/internal/services"
	"projectT/internal/storage/database/models"
)

// ItemSorter предоставляет функциональность для сортировки элементов по различным критериям
type ItemSorter struct{}

// NewItemSorter создает новый экземпляр сортировщика
func NewItemSorter() *ItemSorter {
	return &ItemSorter{}
}

// containsIgnoreCase проверяет наличие подстроки в строке без учета регистра
func containsIgnoreCase(text, substr string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(substr))
}

// SortItems сортирует элементы по заданным настройкам
func (is *ItemSorter) SortItems(items []*models.Item, options *services.FilterOptions) []*models.Item {
	if len(items) <= 1 {
		return items
	}

	// Сначала применяем фильтрацию по типу
	filteredItems := is.filterByType(items, options.ItemType)

	// Затем сортируем по заданным критериям
	sortedItems := make([]*models.Item, len(filteredItems))
	copy(sortedItems, filteredItems)

	switch options.SortBy {
	case "name":
		is.sortByName(sortedItems, options.SortOrder)
	case "created_date":
		is.sortByCreatedDate(sortedItems, options.SortOrder)
	case "modified_date":
		is.sortByModifiedDate(sortedItems, options.SortOrder)
	case "content_size":
		is.sortByContentSize(sortedItems, options.SortOrder)
	default:
		// По умолчанию сортируем по имени по возрастанию
		is.sortByName(sortedItems, "asc")
	}

	// Применяем приоритет, если он задан
	if options.Priority != "none" {
		sortedItems = is.applyPriority(sortedItems, options.Priority)
	}

	// Если установлен порядок по убыванию, переворачиваем список
	if options.SortOrder == "desc" {
		is.reverseItems(sortedItems)
	}

	return sortedItems
}

// filterByType фильтрует элементы по типу
func (is *ItemSorter) filterByType(items []*models.Item, itemType string) []*models.Item {
	if itemType == "all" {
		return items
	}

	result := make([]*models.Item, 0)
	for _, item := range items {
		if is.matchItemType(item, itemType) {
			result = append(result, item)
		}
	}
	return result
}

// matchItemType проверяет соответствие типа элемента заданному типу
func (is *ItemSorter) matchItemType(item *models.Item, itemType string) bool {
	switch itemType {
	case "folders":
		return item.Type == models.ItemTypeFolder
	case "images":
		// Изображения определяются по наличию слова "image" в ContentMeta
		return item.Type == models.ItemTypeElement && containsIgnoreCase(item.ContentMeta, "image")
	case "files":
		// Файлы определяются по наличию слова "file" в ContentMeta
		return item.Type == models.ItemTypeElement && containsIgnoreCase(item.ContentMeta, "file")
	case "links":
		// Ссылки определяются по наличию слова "link" в ContentMeta
		return item.Type == models.ItemTypeElement && containsIgnoreCase(item.ContentMeta, "link")
	case "text":
		// Текст определяется как элемент с пустым ContentMeta, но с присутствующим Description и типом element
		return item.Type == models.ItemTypeElement && item.ContentMeta == "" && item.Description != ""
	default:
		// для "all" или неизвестного типа возвращаем true для всех элементов типа element
		return item.Type == models.ItemTypeElement
	}
}

// sortByName сортирует элементы по имени
func (is *ItemSorter) sortByName(items []*models.Item, order string) {
	sort.Slice(items, func(i, j int) bool {
		less := strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
		if order == "desc" {
			return !less
		}
		return less
	})
}

// sortByCreatedDate сортирует элементы по дате создания
func (is *ItemSorter) sortByCreatedDate(items []*models.Item, order string) {
	sort.Slice(items, func(i, j int) bool {
		// CreatedAt уже является time.Time, не нужно парсить
		timeI := items[i].CreatedAt
		timeJ := items[j].CreatedAt

		less := timeI.Before(timeJ)
		if order == "desc" {
			return !less
		}
		return less
	})
}

// sortByModifiedDate сортирует элементы по дате изменения
func (is *ItemSorter) sortByModifiedDate(items []*models.Item, order string) {
	sort.Slice(items, func(i, j int) bool {
		// Используем UpdatedAt вместо ModifiedAt
		timeI := items[i].UpdatedAt
		timeJ := items[j].UpdatedAt

		less := timeI.Before(timeJ)
		if order == "desc" {
			return !less
		}
		return less
	})
}

// sortByContentSize сортирует элементы по размеру контента
func (is *ItemSorter) sortByContentSize(items []*models.Item, order string) {
	sort.Slice(items, func(i, j int) bool {
		// Используем длину ContentMeta как приближенное значение размера контента
		sizeI := len(items[i].ContentMeta)
		sizeJ := len(items[j].ContentMeta)
		less := sizeI < sizeJ
		if order == "desc" {
			return !less
		}
		return less
	})
}

// reverseItems переворачивает порядок элементов в срезе
func (is *ItemSorter) reverseItems(items []*models.Item) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}

// applyPriority применяет приоритет к списку элементов
func (is *ItemSorter) applyPriority(items []*models.Item, priority string) []*models.Item {
	switch priority {
	case "folders_first":
		return is.prioritizeFolders(items)
	case "images_first":
		return is.prioritizeImages(items)
	case "files_first":
		return is.prioritizeFiles(items)
	case "links_first":
		return is.prioritizeLinks(items)
	case "text_first":
		return is.prioritizeText(items)
	default:
		return items
	}
}

// prioritizeFolders перемещает папки в начало списка
func (is *ItemSorter) prioritizeFolders(items []*models.Item) []*models.Item {
	priorityItems := make([]*models.Item, 0)
	otherItems := make([]*models.Item, 0)

	for _, item := range items {
		if item.Type == models.ItemTypeFolder {
			priorityItems = append(priorityItems, item)
		} else {
			otherItems = append(otherItems, item)
		}
	}

	return append(priorityItems, otherItems...)
}

// prioritizeImages перемещает изображения в начало списка
func (is *ItemSorter) prioritizeImages(items []*models.Item) []*models.Item {
	priorityItems := make([]*models.Item, 0)
	otherItems := make([]*models.Item, 0)

	for _, item := range items {
		if item.Type == models.ItemTypeElement && containsIgnoreCase(item.ContentMeta, "image") {
			priorityItems = append(priorityItems, item)
		} else {
			otherItems = append(otherItems, item)
		}
	}

	return append(priorityItems, otherItems...)
}

// prioritizeFiles перемещает файлы в начало списка
func (is *ItemSorter) prioritizeFiles(items []*models.Item) []*models.Item {
	priorityItems := make([]*models.Item, 0)
	otherItems := make([]*models.Item, 0)

	for _, item := range items {
		if item.Type == models.ItemTypeElement && containsIgnoreCase(item.ContentMeta, "file") {
			priorityItems = append(priorityItems, item)
		} else {
			otherItems = append(otherItems, item)
		}
	}

	return append(priorityItems, otherItems...)
}

// prioritizeLinks перемещает ссылки в начало списка
func (is *ItemSorter) prioritizeLinks(items []*models.Item) []*models.Item {
	priorityItems := make([]*models.Item, 0)
	otherItems := make([]*models.Item, 0)

	for _, item := range items {
		if item.Type == models.ItemTypeElement && containsIgnoreCase(item.ContentMeta, "link") {
			priorityItems = append(priorityItems, item)
		} else {
			otherItems = append(otherItems, item)
		}
	}

	return append(priorityItems, otherItems...)
}

// prioritizeText перемещает текстовые элементы в начало списка
func (is *ItemSorter) prioritizeText(items []*models.Item) []*models.Item {
	priorityItems := make([]*models.Item, 0)
	otherItems := make([]*models.Item, 0)

	for _, item := range items {
		if item.Type == models.ItemTypeElement && item.ContentMeta == "" && item.Description != "" {
			priorityItems = append(priorityItems, item)
		} else {
			otherItems = append(otherItems, item)
		}
	}

	return append(priorityItems, otherItems...)
}
