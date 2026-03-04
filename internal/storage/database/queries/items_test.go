package queries

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB инициализирует тестовую базу данных в памяти
func setupTestDB(t *testing.T) func() {
	// Создаём временный файл для тестовой БД
	tmpFile := ":memory:"

	// Инициализируем БД
	db, err := database.Open(tmpFile)
	require.NoError(t, err)

	// Сохраняем оригинальную БД
	originalDB := database.DB
	database.DB = db

	// Запускаем миграции
	database.RunMigrations()

	// Возвращаем функцию очистки
	return func() {
		database.CloseDB()
		database.DB = originalDB
	}
}

// TestCreateItem проверяет создание элемента
func TestCreateItem(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	item := &models.Item{
		Type:        models.ItemTypeElement,
		Title:       "Test Item",
		Description: "Test Description",
		ContentMeta: `{"type": "text", "content": "hello"}`,
		ParentID:    nil,
	}

	err := CreateItem(item)
	require.NoError(t, err)
	assert.Greater(t, item.ID, 0)

	// Проверяем что элемент создан
	dbItem, err := GetItemByID(item.ID)
	require.NoError(t, err)
	assert.Equal(t, item.Title, dbItem.Title)
	assert.Equal(t, item.Description, dbItem.Description)
	assert.Equal(t, item.Type, dbItem.Type)
	assert.Equal(t, item.ContentMeta, dbItem.ContentMeta)
}

// TestCreateItem_WithParent проверяет создание элемента с родителем
func TestCreateItem_WithParent(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Создаём родительский элемент
	parent := &models.Item{
		Type:  models.ItemTypeFolder,
		Title: "Parent Folder",
	}
	err := CreateItem(parent)
	require.NoError(t, err)

	// Создаём дочерний элемент
	child := &models.Item{
		Type:     models.ItemTypeElement,
		Title:    "Child Item",
		ParentID: &parent.ID,
	}
	err = CreateItem(child)
	require.NoError(t, err)

	// Проверяем что дочерний элемент создан с правильным parent_id
	dbChild, err := GetItemByID(child.ID)
	require.NoError(t, err)
	require.NotNil(t, dbChild.ParentID)
	assert.Equal(t, parent.ID, *dbChild.ParentID)
}

// TestGetItemByID_NotFound проверяет получение несуществующего элемента
func TestGetItemByID_NotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	item, err := GetItemByID(99999)
	assert.Error(t, err)
	assert.Nil(t, item)
	assert.Equal(t, sql.ErrNoRows, err)
}

// TestGetItemsByParent проверяет получение элементов по родителю
func TestGetItemsByParent(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Создаём родительский элемент
	parent := &models.Item{
		Type:  models.ItemTypeFolder,
		Title: "Parent Folder",
	}
	err := CreateItem(parent)
	require.NoError(t, err)

	// Создаём несколько дочерних элементов
	children := []*models.Item{
		{Type: models.ItemTypeElement, Title: "Child 1", ParentID: &parent.ID},
		{Type: models.ItemTypeElement, Title: "Child 2", ParentID: &parent.ID},
		{Type: models.ItemTypeElement, Title: "Child 3", ParentID: &parent.ID},
	}

	for _, child := range children {
		err = CreateItem(child)
		require.NoError(t, err)
	}

	// Получаем дочерние элементы
	dbChildren, err := GetItemsByParent(parent.ID)
	require.NoError(t, err)
	assert.Len(t, dbChildren, 3)

	// Проверяем что все элементы имеют правильный parent_id
	for _, child := range dbChildren {
		require.NotNil(t, child.ParentID)
		assert.Equal(t, parent.ID, *child.ParentID)
	}
}

// TestGetItemsByParent_NoChildren проверяет получение элементов когда нет дочерних
func TestGetItemsByParent_NoChildren(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	parent := &models.Item{
		Type:  models.ItemTypeFolder,
		Title: "Empty Folder",
	}
	err := CreateItem(parent)
	require.NoError(t, err)

	children, err := GetItemsByParent(parent.ID)
	require.NoError(t, err)
	assert.Empty(t, children)
}

// TestGetAllItems проверяет получение всех элементов
func TestGetAllItems(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Создаём несколько элементов
	items := []*models.Item{
		{Type: models.ItemTypeElement, Title: "Item 1"},
		{Type: models.ItemTypeElement, Title: "Item 2"},
		{Type: models.ItemTypeFolder, Title: "Folder 1"},
	}

	for _, item := range items {
		err := CreateItem(item)
		require.NoError(t, err)
	}

	// Получаем все элементы
	allItems, err := GetAllItems()
	require.NoError(t, err)
	assert.Len(t, allItems, 3)
}

// TestGetAllItems_Empty проверяет получение всех элементов из пустой БД
func TestGetAllItems_Empty(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	items, err := GetAllItems()
	require.NoError(t, err)
	assert.Empty(t, items)
}

// TestUpdateItem проверяет обновление элемента
func TestUpdateItem(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Создаём элемент
	item := &models.Item{
		Type:        models.ItemTypeElement,
		Title:       "Original Title",
		Description: "Original Description",
	}
	err := CreateItem(item)
	require.NoError(t, err)

	// Обновляем элемент
	item.Title = "Updated Title"
	item.Description = "Updated Description"
	item.ContentMeta = `{"updated": true}`

	err = UpdateItem(item)
	require.NoError(t, err)

	// Проверяем обновление
	dbItem, err := GetItemByID(item.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", dbItem.Title)
	assert.Equal(t, "Updated Description", dbItem.Description)
	assert.Equal(t, `{"updated": true}`, dbItem.ContentMeta)

	// Проверяем что updated_at обновился
	assert.True(t, dbItem.UpdatedAt.After(item.CreatedAt))
}

// TestUpdateItem_NotFound проверяет обновление несуществующего элемента
func TestUpdateItem_NotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	item := &models.Item{
		ID:    99999,
		Title: "Non-existent",
	}

	err := UpdateItem(item)
	// SQLite не возвращает ошибку при обновлении несуществующей записи
	// но и ничего не обновляет
	assert.NoError(t, err)
}

// TestDeleteItem проверяет удаление элемента
func TestDeleteItem(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Создаём элемент
	item := &models.Item{
		Type:  models.ItemTypeElement,
		Title: "To Delete",
	}
	err := CreateItem(item)
	require.NoError(t, err)

	// Удаляем элемент
	err = DeleteItem(item.ID)
	require.NoError(t, err)

	// Проверяем что элемент удалён
	_, err = GetItemByID(item.ID)
	assert.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
}

// TestDeleteItem_NotFound проверяет удаление несуществующего элемента
func TestDeleteItem_NotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	err := DeleteItem(99999)
	// SQLite не возвращает ошибку при удалении несуществующей записи
	assert.NoError(t, err)
}

// TestSearchItems_ByTitle проверяет поиск по названию
func TestSearchItems_ByTitle(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Создаём тестовые элементы
	items := []*models.Item{
		{Type: models.ItemTypeElement, Title: "Project Alpha"},
		{Type: models.ItemTypeElement, Title: "Project Beta"},
		{Type: models.ItemTypeElement, Title: "Other Item"},
	}

	for _, item := range items {
		err := CreateItem(item)
		require.NoError(t, err)
	}

	// Ищем по названию
	results, err := SearchItems("Alpha")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Project Alpha", results[0].Title)
}

// TestSearchItems_ByDescription проверяет поиск по описанию
func TestSearchItems_ByDescription(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	items := []*models.Item{
		{Type: models.ItemTypeElement, Title: "Item 1", Description: "Important document"},
		{Type: models.ItemTypeElement, Title: "Item 2", Description: "Regular note"},
	}

	for _, item := range items {
		err := CreateItem(item)
		require.NoError(t, err)
	}

	// Ищем по описанию
	results, err := SearchItems("Important")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Important document", results[0].Description)
}

// TestSearchItems_NoResults проверяет поиск без результатов
func TestSearchItems_NoResults(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	results, err := SearchItems("NonExistent")
	require.NoError(t, err)
	assert.Empty(t, results)
}

// TestSearchItems_EmptyQuery проверяет поиск с пустым запросом
func TestSearchItems_EmptyQuery(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Создаём элемент
	item := &models.Item{
		Type:  models.ItemTypeElement,
		Title: "Test",
	}
	err := CreateItem(item)
	require.NoError(t, err)

	// Пустой запрос вернёт все элементы (так как LIKE "%%" матчит всё)
	results, err := SearchItems("")
	require.NoError(t, err)
	assert.NotEmpty(t, results)
}

// TestPinItem проверяет закрепление элемента
func TestPinItem(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Создаём элемент
	item := &models.Item{
		Type:  models.ItemTypeElement,
		Title: "Pinned Item",
	}
	err := CreateItem(item)
	require.NoError(t, err)

	// Закрепляем
	err = PinItem(item.ID)
	require.NoError(t, err)

	// Проверяем что закреплён
	pinned, err := IsItemPinned(item.ID)
	require.NoError(t, err)
	assert.True(t, pinned)
}

// TestUnpinItem проверяет открепление элемента
func TestUnpinItem(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Создаём и закрепляем элемент
	item := &models.Item{
		Type:  models.ItemTypeElement,
		Title: "To Unpin",
	}
	err := CreateItem(item)
	require.NoError(t, err)

	err = PinItem(item.ID)
	require.NoError(t, err)

	// Открепляем
	err = UnpinItem(item.ID)
	require.NoError(t, err)

	// Проверяем что откреплён
	pinned, err := IsItemPinned(item.ID)
	require.NoError(t, err)
	assert.False(t, pinned)
}

// TestIsItemPinned_NotPinned проверяет проверку незакреплённого элемента
func TestIsItemPinned_NotPinned(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	item := &models.Item{
		Type:  models.ItemTypeElement,
		Title: "Not Pinned",
	}
	err := CreateItem(item)
	require.NoError(t, err)

	pinned, err := IsItemPinned(item.ID)
	require.NoError(t, err)
	assert.False(t, pinned)
}

// TestIsItemPinned_NotFound проверяет проверку несуществующего элемента
func TestIsItemPinned_NotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	pinned, err := IsItemPinned(99999)
	require.NoError(t, err)
	assert.False(t, pinned)
}

// TestItem_Integration полный цикл CRUD для элемента
func TestItem_Integration(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// 1. Создаём элемент
	item := &models.Item{
		Type:        models.ItemTypeElement,
		Title:       "Integration Test Item",
		Description: "Testing full CRUD cycle",
		ContentMeta: `{"blocks": [{"type": "text", "content": "Hello World"}]}`,
	}

	err := CreateItem(item)
	require.NoError(t, err)
	initialID := item.ID
	assert.Greater(t, initialID, 0)

	// 2. Читаем элемент
	readItem, err := GetItemByID(item.ID)
	require.NoError(t, err)
	assert.Equal(t, item.Title, readItem.Title)
	assert.Equal(t, item.Description, readItem.Description)

	// 3. Обновляем элемент
	readItem.Title = "Updated Integration Test"
	readItem.Description = "Updated description"
	err = UpdateItem(readItem)
	require.NoError(t, err)

	// 4. Проверяем обновление
	updatedItem, err := GetItemByID(item.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Integration Test", updatedItem.Title)

	// 5. Закрепляем элемент
	err = PinItem(item.ID)
	require.NoError(t, err)

	pinned, err := IsItemPinned(item.ID)
	require.NoError(t, err)
	assert.True(t, pinned)

	// 6. Добавляем теги через queries
	tagIDs, err := GetOrCreateTags(ctx, []string{"test", "integration"})
	require.NoError(t, err)
	assert.Len(t, tagIDs, 2)

	err = ReplaceItemTags(ctx, item.ID, tagIDs)
	require.NoError(t, err)

	// 7. Проверяем теги
	tags, err := GetTagsForItem(ctx, item.ID)
	require.NoError(t, err)
	assert.Len(t, tags, 2)

	// 8. Открепляем элемент
	err = UnpinItem(item.ID)
	require.NoError(t, err)

	// 9. Удаляем элемент
	err = DeleteItem(item.ID)
	require.NoError(t, err)

	// 10. Проверяем что элемент удалён
	_, err = GetItemByID(item.ID)
	assert.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
}

// TestGetItemsByParent_WithSoftDelete проверяет что удалённые элементы не возвращаются
func TestGetItemsByParent_WithTimestamps(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Создаём элемент
	item := &models.Item{
		Type:  models.ItemTypeElement,
		Title: "Timestamp Test",
	}
	err := CreateItem(item)
	require.NoError(t, err)

	// Проверяем что timestamps установлены
	assert.False(t, item.CreatedAt.IsZero())
	assert.False(t, item.UpdatedAt.IsZero())

	// Ждём немного и обновляем
	time.Sleep(10 * time.Millisecond)

	item.Title = "Updated"
	err = UpdateItem(item)
	require.NoError(t, err)

	// Проверяем что updated_at обновился
	updatedItem, err := GetItemByID(item.ID)
	require.NoError(t, err)
	assert.True(t, updatedItem.UpdatedAt.After(item.CreatedAt))
}
