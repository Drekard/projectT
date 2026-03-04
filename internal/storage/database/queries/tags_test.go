package queries

import (
	"context"
	"testing"

	"projectT/internal/storage/database/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateTag проверяет создание тега
func TestCreateTag(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tag := &models.Tag{
		Name:        "test-tag",
		Color:       "#FF5733",
		Description: "Test tag description",
	}

	err := CreateTag(ctx, tag)
	require.NoError(t, err)
	assert.Greater(t, tag.ID, 0)

	// Проверяем что тег создан
	dbTag, err := GetTagByID(ctx, tag.ID)
	require.NoError(t, err)
	assert.Equal(t, tag.Name, dbTag.Name)
	assert.Equal(t, tag.Color, dbTag.Color)
	assert.Equal(t, tag.Description, dbTag.Description)
}

// TestCreateTag_WithoutDescription проверяет создание тега без описания
func TestCreateTag_WithoutDescription(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tag := &models.Tag{
		Name:  "simple-tag",
		Color: "#00FF00",
	}

	err := CreateTag(ctx, tag)
	require.NoError(t, err)

	dbTag, err := GetTagByID(ctx, tag.ID)
	require.NoError(t, err)
	assert.Equal(t, "simple-tag", dbTag.Name)
	assert.Equal(t, "#00FF00", dbTag.Color)
}

// TestGetTagByID_NotFound проверяет получение несуществующего тега
func TestGetTagByID_NotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tag, err := GetTagByID(ctx, 99999)
	assert.Error(t, err)
	assert.Nil(t, tag)
	assert.Contains(t, err.Error(), "не найден")
}

// TestGetTagByName проверяет получение тега по имени
func TestGetTagByName(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём тег
	tag := &models.Tag{
		Name:  "unique-name",
		Color: "#123456",
	}
	err := CreateTag(ctx, tag)
	require.NoError(t, err)

	// Получаем по имени
	dbTag, err := GetTagByName(ctx, "unique-name")
	require.NoError(t, err)
	assert.Equal(t, tag.ID, dbTag.ID)
	assert.Equal(t, "unique-name", dbTag.Name)
}

// TestGetTagByName_NotFound проверяет получение несуществующего тега по имени
func TestGetTagByName_NotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tag, err := GetTagByName(ctx, "non-existent")
	assert.Error(t, err)
	assert.Nil(t, tag)
	assert.Contains(t, err.Error(), "не найден")
}

// TestGetOrCreateTag_Existing проверяет получение существующего тега
func TestGetOrCreateTag_Existing(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём тег
	originalTag := &models.Tag{
		Name:  "existing-tag",
		Color: "#ABCDEF",
	}
	err := CreateTag(ctx, originalTag)
	require.NoError(t, err)

	// Получаем или создаём
	tag, err := GetOrCreateTag(ctx, "existing-tag")
	require.NoError(t, err)
	assert.Equal(t, originalTag.ID, tag.ID)
	assert.Equal(t, "existing-tag", tag.Name)
}

// TestGetOrCreateTag_New проверяет создание нового тега
func TestGetOrCreateTag_New(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tag, err := GetOrCreateTag(ctx, "brand-new-tag")
	require.NoError(t, err)
	assert.Greater(t, tag.ID, 0)
	assert.Equal(t, "brand-new-tag", tag.Name)
	assert.Equal(t, "#808080", tag.Color) // Цвет по умолчанию

	// Проверяем что тег действительно создан
	dbTag, err := GetTagByID(ctx, tag.ID)
	require.NoError(t, err)
	assert.Equal(t, tag.ID, dbTag.ID)
}

// TestGetOrCreateTags_Multiple проверяет получение или создание нескольких тегов
func TestGetOrCreateTags_Multiple(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём один тег заранее
	existing := &models.Tag{
		Name:  "existing",
		Color: "#111111",
	}
	err := CreateTag(ctx, existing)
	require.NoError(t, err)

	// Получаем или создаём несколько тегов
	tagIDs, err := GetOrCreateTags(ctx, []string{"existing", "new1", "new2"})
	require.NoError(t, err)
	assert.Len(t, tagIDs, 3)

	// Все ID должны быть больше 0
	for _, id := range tagIDs {
		assert.Greater(t, id, 0)
	}

	// Проверяем что теги существуют
	for _, id := range tagIDs {
		tag, err := GetTagByID(ctx, id)
		require.NoError(t, err)
		assert.NotNil(t, tag)
	}
}

// TestGetOrCreateTags_WithDuplicates проверяет обработку дубликатов
func TestGetOrCreateTags_WithDuplicates(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tagIDs, err := GetOrCreateTags(ctx, []string{"tag1", "tag1", "tag2", "tag2"})
	require.NoError(t, err)

	// Должны быть только уникальные теги
	assert.Len(t, tagIDs, 2)

	// Проверяем что теги существуют
	for _, id := range tagIDs {
		tag, err := GetTagByID(ctx, id)
		require.NoError(t, err)
		assert.NotNil(t, tag)
	}
}

// TestGetOrCreateTags_Empty проверяет обработку пустого списка
func TestGetOrCreateTags_Empty(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tagIDs, err := GetOrCreateTags(ctx, []string{})
	require.NoError(t, err)
	assert.Empty(t, tagIDs)
}

// TestGetOrCreateTags_WithEmptyStrings проверяет обработку пустых строк
func TestGetOrCreateTags_WithEmptyStrings(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tagIDs, err := GetOrCreateTags(ctx, []string{"", "  ", "valid-tag", ""})
	require.NoError(t, err)
	assert.Len(t, tagIDs, 1)
}

// TestGetAllTags проверяет получение всех тегов
func TestGetAllTags(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём несколько тегов
	tags := []*models.Tag{
		{Name: "alpha", Color: "#AA0000"},
		{Name: "beta", Color: "#00AA00"},
		{Name: "gamma", Color: "#0000AA"},
	}

	for _, tag := range tags {
		err := CreateTag(ctx, tag)
		require.NoError(t, err)
	}

	// Получаем все теги
	allTags, err := GetAllTags(ctx)
	require.NoError(t, err)
	assert.Len(t, allTags, 3)

	// Проверяем что item_count = 0 (теги не привязаны к элементам)
	for _, tag := range allTags {
		assert.Equal(t, 0, tag.ItemCount)
	}
}

// TestGetAllTags_Empty проверяет получение всех тегов из пустой БД
func TestGetAllTags_Empty(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tags, err := GetAllTags(ctx)
	require.NoError(t, err)
	assert.Empty(t, tags)
}

// TestGetAllTags_WithItemCount проверяет подсчёт количества элементов
func TestGetAllTags_WithItemCount(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём тег
	tag := &models.Tag{
		Name:  "counted-tag",
		Color: "#FF00FF",
	}
	err := CreateTag(ctx, tag)
	require.NoError(t, err)

	// Создаём элементы и привязываем теги
	items := []*models.Item{
		{Type: models.ItemTypeElement, Title: "Item 1"},
		{Type: models.ItemTypeElement, Title: "Item 2"},
		{Type: models.ItemTypeElement, Title: "Item 3"},
	}

	for _, item := range items {
		err = CreateItem(item)
		require.NoError(t, err)

		err = AddTagToItem(ctx, item.ID, tag.ID)
		require.NoError(t, err)
	}

	// Получаем все теги
	allTags, err := GetAllTags(ctx)
	require.NoError(t, err)
	assert.Len(t, allTags, 1)

	// Проверяем что item_count = 3
	assert.Equal(t, 3, allTags[0].ItemCount)
}

// TestSearchTagsByName проверяет поиск тегов по имени
func TestSearchTagsByName(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём теги
	tags := []*models.Tag{
		{Name: "programming", Color: "#111111"},
		{Name: "python", Color: "#222222"},
		{Name: "javascript", Color: "#333333"},
		{Name: "database", Color: "#444444"},
	}

	for _, tag := range tags {
		err := CreateTag(ctx, tag)
		require.NoError(t, err)
	}

	// Ищем по частичному совпадению
	results, err := SearchTagsByName(ctx, "script")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "javascript", results[0].Name)
}

// TestSearchTagsByName_CaseInsensitive проверяет регистронезависимый поиск
func TestSearchTagsByName_CaseInsensitive(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tag := &models.Tag{
		Name:  "Python",
		Color: "#FFFF00",
	}
	err := CreateTag(ctx, tag)
	require.NoError(t, err)

	// Ищем в нижнем регистре
	results, err := SearchTagsByName(ctx, "python")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Python", results[0].Name)
}

// TestSearchTagsByName_NoResults проверяет поиск без результатов
func TestSearchTagsByName_NoResults(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	results, err := SearchTagsByName(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, results)
}

// TestSearchTagsByName_Limit проверяет ограничение результатов
func TestSearchTagsByName_Limit(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём много тегов с одинаковым префиксом
	for i := 0; i < 60; i++ {
		tag := &models.Tag{
			Name:  "tag-" + string(rune('A'+i%26)),
			Color: "#CCCCCC",
		}
		err := CreateTag(ctx, tag)
		require.NoError(t, err)
	}

	results, err := SearchTagsByName(ctx, "tag")
	require.NoError(t, err)
	// Ограничение 50
	assert.LessOrEqual(t, len(results), 50)
}

// TestUpdateTag проверяет обновление тега
func TestUpdateTag(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём тег
	tag := &models.Tag{
		Name:        "old-name",
		Color:       "#000000",
		Description: "old description",
	}
	err := CreateTag(ctx, tag)
	require.NoError(t, err)

	// Обновляем
	tag.Name = "new-name"
	tag.Color = "#FFFFFF"
	tag.Description = "new description"

	err = UpdateTag(ctx, tag)
	require.NoError(t, err)

	// Проверяем обновление
	dbTag, err := GetTagByID(ctx, tag.ID)
	require.NoError(t, err)
	assert.Equal(t, "new-name", dbTag.Name)
	assert.Equal(t, "#FFFFFF", dbTag.Color)
	assert.Equal(t, "new description", dbTag.Description)
}

// TestDeleteTag проверяет удаление тега
func TestDeleteTag(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём тег
	tag := &models.Tag{
		Name:  "to-delete",
		Color: "#999999",
	}
	err := CreateTag(ctx, tag)
	require.NoError(t, err)

	// Удаляем
	err = DeleteTag(ctx, tag.ID)
	require.NoError(t, err)

	// Проверяем что удалён
	_, err = GetTagByID(ctx, tag.ID)
	assert.Error(t, err)
}

// TestDeleteTag_WithItems проверяет удаление тега с привязанными элементами
func TestDeleteTag_WithItems(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём тег
	tag := &models.Tag{
		Name:  "tag-with-items",
		Color: "#777777",
	}
	err := CreateTag(ctx, tag)
	require.NoError(t, err)

	// Создаём элемент и привязываем тег
	item := &models.Item{
		Type:  models.ItemTypeElement,
		Title: "Item with tag",
	}
	err = CreateItem(item)
	require.NoError(t, err)

	err = AddTagToItem(ctx, item.ID, tag.ID)
	require.NoError(t, err)

	// Проверяем что тег привязан
	tags, err := GetTagsForItem(ctx, item.ID)
	require.NoError(t, err)
	assert.Len(t, tags, 1)

	// Удаляем тег
	err = DeleteTag(ctx, tag.ID)
	require.NoError(t, err)

	// Проверяем что связь удалена
	tags, err = GetTagsForItem(ctx, item.ID)
	require.NoError(t, err)
	assert.Empty(t, tags)

	// Элемент должен остаться
	dbItem, err := GetItemByID(item.ID)
	require.NoError(t, err)
	assert.NotNil(t, dbItem)
}

// TestAddTagToItem проверяет добавление связи тега с элементом
func TestAddTagToItem(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём тег и элемент
	tag := &models.Tag{
		Name:  "test-tag",
		Color: "#123456",
	}
	err := CreateTag(ctx, tag)
	require.NoError(t, err)

	item := &models.Item{
		Type:  models.ItemTypeElement,
		Title: "Test Item",
	}
	err = CreateItem(item)
	require.NoError(t, err)

	// Добавляем связь
	err = AddTagToItem(ctx, item.ID, tag.ID)
	require.NoError(t, err)

	// Проверяем связь
	tags, err := GetTagsForItem(ctx, item.ID)
	require.NoError(t, err)
	assert.Len(t, tags, 1)
	assert.Equal(t, tag.ID, tags[0].ID)
}

// TestAddTagToItem_Duplicate проверяет добавление дубликата связи
func TestAddTagToItem_Duplicate(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tag := &models.Tag{Name: "dup-tag"}
	err := CreateTag(ctx, tag)
	require.NoError(t, err)

	item := &models.Item{Type: models.ItemTypeElement, Title: "Item"}
	err = CreateItem(item)
	require.NoError(t, err)

	// Добавляем связь дважды
	err = AddTagToItem(ctx, item.ID, tag.ID)
	require.NoError(t, err)
	err = AddTagToItem(ctx, item.ID, tag.ID)
	require.NoError(t, err)

	// Должна быть только одна связь
	tags, err := GetTagsForItem(ctx, item.ID)
	require.NoError(t, err)
	assert.Len(t, tags, 1)
}

// TestRemoveTagFromItem проверяет удаление связи тега с элементом
func TestRemoveTagFromItem(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tag := &models.Tag{Name: "remove-tag"}
	err := CreateTag(ctx, tag)
	require.NoError(t, err)

	item := &models.Item{Type: models.ItemTypeElement, Title: "Item"}
	err = CreateItem(item)
	require.NoError(t, err)

	// Добавляем связь
	err = AddTagToItem(ctx, item.ID, tag.ID)
	require.NoError(t, err)

	// Удаляем связь
	err = RemoveTagFromItem(ctx, item.ID, tag.ID)
	require.NoError(t, err)

	// Проверяем что связь удалена
	tags, err := GetTagsForItem(ctx, item.ID)
	require.NoError(t, err)
	assert.Empty(t, tags)
}

// TestReplaceItemTags проверяет замену всех тегов элемента
func TestReplaceItemTags(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём элемент
	item := &models.Item{
		Type:  models.ItemTypeElement,
		Title: "Tag Test Item",
	}
	err := CreateItem(item)
	require.NoError(t, err)

	// Создаём теги
	oldTags := []*models.Tag{
		{Name: "old1"},
		{Name: "old2"},
	}
	var oldTagIDs []int
	for _, tag := range oldTags {
		err = CreateTag(ctx, tag)
		require.NoError(t, err)
		oldTagIDs = append(oldTagIDs, tag.ID)
	}

	// Привязываем старые теги
	err = ReplaceItemTags(ctx, item.ID, oldTagIDs)
	require.NoError(t, err)

	// Проверяем
	tags, err := GetTagsForItem(ctx, item.ID)
	require.NoError(t, err)
	assert.Len(t, tags, 2)

	// Создаём новые теги
	newTags := []*models.Tag{
		{Name: "new1"},
		{Name: "new2"},
		{Name: "new3"},
	}
	var newTagIDs []int
	for _, tag := range newTags {
		err = CreateTag(ctx, tag)
		require.NoError(t, err)
		newTagIDs = append(newTagIDs, tag.ID)
	}

	// Заменяем теги
	err = ReplaceItemTags(ctx, item.ID, newTagIDs)
	require.NoError(t, err)

	// Проверяем что старые теги удалены, новые добавлены
	tags, err = GetTagsForItem(ctx, item.ID)
	require.NoError(t, err)
	assert.Len(t, tags, 3)

	// Проверяем что это именно новые теги
	tagNames := make(map[string]bool)
	for _, tag := range tags {
		tagNames[tag.Name] = true
	}
	assert.True(t, tagNames["new1"])
	assert.True(t, tagNames["new2"])
	assert.True(t, tagNames["new3"])
	assert.False(t, tagNames["old1"])
	assert.False(t, tagNames["old2"])
}

// TestReplaceItemTags_Empty проверяет замену всех тегов на пустой список
func TestReplaceItemTags_Empty(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	item := &models.Item{Type: models.ItemTypeElement, Title: "Item"}
	err := CreateItem(item)
	require.NoError(t, err)

	tag := &models.Tag{Name: "to-remove"}
	err = CreateTag(ctx, tag)
	require.NoError(t, err)

	// Привязываем тег
	err = AddTagToItem(ctx, item.ID, tag.ID)
	require.NoError(t, err)

	// Заменяем на пустой список
	err = ReplaceItemTags(ctx, item.ID, []int{})
	require.NoError(t, err)

	// Все теги должны быть удалены
	tags, err := GetTagsForItem(ctx, item.ID)
	require.NoError(t, err)
	assert.Empty(t, tags)
}

// TestGetTagsForItem проверяет получение всех тегов элемента
func TestGetTagsForItem(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	item := &models.Item{Type: models.ItemTypeElement, Title: "Item"}
	err := CreateItem(item)
	require.NoError(t, err)

	tags := []*models.Tag{
		{Name: "tag1", Color: "#111111"},
		{Name: "tag2", Color: "#222222"},
		{Name: "tag3", Color: "#333333"},
	}
	var tagIDs []int
	for _, tag := range tags {
		err = CreateTag(ctx, tag)
		require.NoError(t, err)
		tagIDs = append(tagIDs, tag.ID)
	}

	err = ReplaceItemTags(ctx, item.ID, tagIDs)
	require.NoError(t, err)

	itemTags, err := GetTagsForItem(ctx, item.ID)
	require.NoError(t, err)
	assert.Len(t, itemTags, 3)
}

// TestGetTagsForItem_NoTags проверяет получение тегов когда их нет
func TestGetTagsForItem_NoTags(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	item := &models.Item{Type: models.ItemTypeElement, Title: "Item"}
	err := CreateItem(item)
	require.NoError(t, err)

	tags, err := GetTagsForItem(ctx, item.ID)
	require.NoError(t, err)
	assert.Empty(t, tags)
}

// TestGetItemsForTag проверяет получение всех элементов тега
func TestGetItemsForTag(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tag := &models.Tag{Name: "items-tag"}
	err := CreateTag(ctx, tag)
	require.NoError(t, err)

	items := []*models.Item{
		{Type: models.ItemTypeElement, Title: "Item 1"},
		{Type: models.ItemTypeElement, Title: "Item 2"},
		{Type: models.ItemTypeElement, Title: "Item 3"},
	}

	for _, item := range items {
		err = CreateItem(item)
		require.NoError(t, err)
		err = AddTagToItem(ctx, item.ID, tag.ID)
		require.NoError(t, err)
	}

	tagItems, err := GetItemsForTag(ctx, tag.ID)
	require.NoError(t, err)
	assert.Len(t, tagItems, 3)
}

// TestGetTagsUsageCount проверяет подсчёт использования тегов
func TestGetTagsUsageCount(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём теги
	tags := []*models.Tag{
		{Name: "used-once"},
		{Name: "used-twice"},
		{Name: "not-used"},
	}
	var tagIDs []int
	for _, tag := range tags {
		err := CreateTag(ctx, tag)
		require.NoError(t, err)
		tagIDs = append(tagIDs, tag.ID)
	}

	// Создаём элементы и привязываем теги
	item1 := &models.Item{Type: models.ItemTypeElement, Title: "Item 1"}
	err := CreateItem(item1)
	require.NoError(t, err)
	err = AddTagToItem(ctx, item1.ID, tagIDs[0]) // used-once
	require.NoError(t, err)
	err = AddTagToItem(ctx, item1.ID, tagIDs[1]) // used-twice
	require.NoError(t, err)

	item2 := &models.Item{Type: models.ItemTypeElement, Title: "Item 2"}
	err = CreateItem(item2)
	require.NoError(t, err)
	err = AddTagToItem(ctx, item2.ID, tagIDs[1]) // used-twice
	require.NoError(t, err)

	// Получаем статистику
	count, err := GetTagsUsageCount(ctx)
	require.NoError(t, err)

	assert.Equal(t, 1, count[tagIDs[0]]) // used-once
	assert.Equal(t, 2, count[tagIDs[1]]) // used-twice
	_, exists := count[tagIDs[2]]
	assert.False(t, exists) // not-used не должен быть в мапе
}

// TestBulkUpdateTags проверяет массовое обновление тегов
func TestBulkUpdateTags(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём теги
	tags := []*models.Tag{
		{Name: "tag1", Color: "#111111"},
		{Name: "tag2", Color: "#222222"},
		{Name: "tag3", Color: "#333333"},
	}

	for _, tag := range tags {
		err := CreateTag(ctx, tag)
		require.NoError(t, err)
	}

	// Обновляем теги
	tags[0].Name = "updated1"
	tags[0].Color = "#AAAAAA"
	tags[1].Name = "updated2"
	tags[1].Color = "#BBBBBB"

	err := BulkUpdateTags(ctx, tags)
	require.NoError(t, err)

	// Проверяем обновление
	for _, tag := range tags {
		dbTag, err := GetTagByID(ctx, tag.ID)
		require.NoError(t, err)
		assert.Equal(t, tag.Name, dbTag.Name)
		assert.Equal(t, tag.Color, dbTag.Color)
	}
}

// TestBulkUpdateTags_Empty проверяет массовое обновление пустого списка
func TestBulkUpdateTags_Empty(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	err := BulkUpdateTags(ctx, []*models.Tag{})
	require.NoError(t, err)
}

// TestTag_Integration полный цикл CRUD для тегов
func TestTag_Integration(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// 1. Создаём тег
	tag := &models.Tag{
		Name:        "integration-tag",
		Color:       "#FF0000",
		Description: "Integration test tag",
	}
	err := CreateTag(ctx, tag)
	require.NoError(t, err)
	assert.Greater(t, tag.ID, 0)

	// 2. Получаем тег
	dbTag, err := GetTagByID(ctx, tag.ID)
	require.NoError(t, err)
	assert.Equal(t, tag.Name, dbTag.Name)

	// 3. Обновляем тег
	dbTag.Color = "#00FF00"
	err = UpdateTag(ctx, dbTag)
	require.NoError(t, err)

	// 4. Создаём элемент и привязываем тег
	item := &models.Item{
		Type:  models.ItemTypeElement,
		Title: "Tagged Item",
	}
	err = CreateItem(item)
	require.NoError(t, err)

	err = AddTagToItem(ctx, item.ID, tag.ID)
	require.NoError(t, err)

	// 5. Проверяем связь
	tags, err := GetTagsForItem(ctx, item.ID)
	require.NoError(t, err)
	assert.Len(t, tags, 1)

	// 6. Проверяем статистику
	count, err := GetTagsUsageCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count[tag.ID])

	// 7. Получаем элементы тега
	items, err := GetItemsForTag(ctx, tag.ID)
	require.NoError(t, err)
	assert.Len(t, items, 1)

	// 8. Заменяем теги на другие
	newTag := &models.Tag{Name: "replacement-tag"}
	err = CreateTag(ctx, newTag)
	require.NoError(t, err)

	err = ReplaceItemTags(ctx, item.ID, []int{newTag.ID})
	require.NoError(t, err)

	// 9. Проверяем что старый тег отвязан
	tags, err = GetTagsForItem(ctx, item.ID)
	require.NoError(t, err)
	assert.Len(t, tags, 1)
	assert.Equal(t, newTag.ID, tags[0].ID)

	// 10. Удаляем тег
	err = DeleteTag(ctx, tag.ID)
	require.NoError(t, err)

	// 11. Проверяем что тег удалён
	_, err = GetTagByID(ctx, tag.ID)
	assert.Error(t, err)
}
