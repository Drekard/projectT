package services

import "sync"

// SortSettingsService отвечает за хранение и управление настройками сортировки
type SortSettingsService struct {
	mu            sync.RWMutex
	filterOptions *FilterOptions
}

// FilterOptions хранит выбранные опции фильтрации и сортировки
type FilterOptions struct {
	ItemType  string // Тип элемента: "all", "folders", "images", "files", "links", "text"
	Priority  string // Приоритет: "none", "folders_first", "images_first", "files_first", "links_first", "text_first"
	SortBy    string // Сортировка: "name", "created_date", "modified_date", "content_size"
	SortOrder string // Порядок: "asc", "desc"
	TabMode   string // Режим вкладки: "current_folder" или "all_items"
}

// GlobalSortSettingsService глобальный экземпляр сервиса настроек сортировки
var GlobalSortSettingsService = NewSortSettingsService()

// NewSortSettingsService создает новый экземпляр сервиса настроек сортировки
func NewSortSettingsService() *SortSettingsService {
	return &SortSettingsService{
		filterOptions: &FilterOptions{
			ItemType:  "all",
			Priority:  "none",
			SortBy:    "name",
			SortOrder: "asc",
		},
	}
}

// GetFilterOptions возвращает текущие настройки фильтрации и сортировки
func (sss *SortSettingsService) GetFilterOptions() *FilterOptions {
	sss.mu.RLock()
	defer sss.mu.RUnlock()

	// Возвращаем копию, чтобы избежать проблем с конкурентным доступом
	options := *sss.filterOptions
	return &options
}

// SetFilterOptions устанавливает новые настройки фильтрации и сортировки
func (sss *SortSettingsService) SetFilterOptions(options *FilterOptions) {
	sss.mu.Lock()
	defer sss.mu.Unlock()

	sss.filterOptions = options
}

// UpdateFilterOptions обновляет конкретные поля настроек
func (sss *SortSettingsService) UpdateFilterOptions(itemType, priority, sortBy, sortOrder string) {
	sss.mu.Lock()
	defer sss.mu.Unlock()

	if sss.filterOptions == nil {
		sss.filterOptions = &FilterOptions{}
	}

	if itemType != "" {
		sss.filterOptions.ItemType = itemType
	}
	if priority != "" {
		sss.filterOptions.Priority = priority
	}
	if sortBy != "" {
		sss.filterOptions.SortBy = sortBy
	}
	if sortOrder != "" {
		sss.filterOptions.SortOrder = sortOrder
	}
}
