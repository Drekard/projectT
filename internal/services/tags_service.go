package services

import (
	"context"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// TagsService предоставляет сервис для работы с тегами
type TagsService struct{}

// NewTagsService создает новый экземпляр сервиса тегов
func NewTagsService() *TagsService {
	return &TagsService{}
}

// CreateTag создает новый тег
func (ts *TagsService) CreateTag(ctx context.Context, tag *models.Tag) error {
	return queries.CreateTag(ctx, tag)
}

// GetTagByID возвращает тег по ID
func (ts *TagsService) GetTagByID(ctx context.Context, id int) (*models.Tag, error) {
	return queries.GetTagByID(ctx, id)
}

// GetTagByName возвращает тег по имени
func (ts *TagsService) GetTagByName(ctx context.Context, name string) (*models.Tag, error) {
	return queries.GetTagByName(ctx, name)
}

// GetOrCreateTag получает существующий тег или создает новый
func (ts *TagsService) GetOrCreateTag(ctx context.Context, name string) (*models.Tag, error) {
	return queries.GetOrCreateTag(ctx, name)
}

// GetOrCreateTags получает или создает несколько тегов
func (ts *TagsService) GetOrCreateTags(ctx context.Context, tagNames []string) ([]int, error) {
	return queries.GetOrCreateTags(ctx, tagNames)
}

// GetAllTags возвращает все теги с подсчетом элементов
func (ts *TagsService) GetAllTags(ctx context.Context) ([]*models.Tag, error) {
	return queries.GetAllTags(ctx)
}

// SearchTagsByName ищет теги по имени
func (ts *TagsService) SearchTagsByName(ctx context.Context, name string) ([]*models.Tag, error) {
	return queries.SearchTagsByName(ctx, name)
}

// UpdateTag обновляет тег
func (ts *TagsService) UpdateTag(ctx context.Context, tag *models.Tag) error {
	return queries.UpdateTag(ctx, tag)
}

// DeleteTag удаляет тег
func (ts *TagsService) DeleteTag(ctx context.Context, id int) error {
	return queries.DeleteTag(ctx, id)
}

// AddTagToItem добавляет связь тега с элементом
func (ts *TagsService) AddTagToItem(ctx context.Context, itemID, tagID int) error {
	return queries.AddTagToItem(ctx, itemID, tagID)
}

// RemoveTagFromItem удаляет связь тега с элементом
func (ts *TagsService) RemoveTagFromItem(ctx context.Context, itemID, tagID int) error {
	return queries.RemoveTagFromItem(ctx, itemID, tagID)
}

// ReplaceItemTags заменяет все теги элемента на новые
func (ts *TagsService) ReplaceItemTags(ctx context.Context, itemID int, tagIDs []int) error {
	return queries.ReplaceItemTags(ctx, itemID, tagIDs)
}

// GetTagsForItem возвращает все теги элемента
func (ts *TagsService) GetTagsForItem(ctx context.Context, itemID int) ([]*models.Tag, error) {
	return queries.GetTagsForItem(ctx, itemID)
}

// GetItemsForTag возвращает все элементы тега
func (ts *TagsService) GetItemsForTag(ctx context.Context, tagID int) ([]*models.Item, error) {
	return queries.GetItemsForTag(ctx, tagID)
}

// GetTagsUsageCount возвращает количество использований каждого тега
func (ts *TagsService) GetTagsUsageCount(ctx context.Context) (map[int]int, error) {
	return queries.GetTagsUsageCount(ctx)
}

// BulkUpdateTags обновляет несколько тегов в одной транзакции
func (ts *TagsService) BulkUpdateTags(ctx context.Context, tags []*models.Tag) error {
	return queries.BulkUpdateTags(ctx, tags)
}
