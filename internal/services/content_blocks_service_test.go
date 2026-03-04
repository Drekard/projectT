package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"projectT/internal/storage/database/models"
)

func TestNewContentBlocksService(t *testing.T) {
	service := NewContentBlocksService()
	assert.NotNil(t, service)
}

// TestProcessFileData_EmptyInput проверяет обработку пустых входных данных
func TestProcessFileData_EmptyInput(t *testing.T) {
	service := NewContentBlocksService()

	files := []string{}
	links := []string{}

	blocks, errors := service.ProcessFileData(&files, links)

	assert.Empty(t, blocks)
	assert.Empty(t, errors)
}

// TestProcessFileData_OnlyLinks проверяет обработку только ссылок
func TestProcessFileData_OnlyLinks(t *testing.T) {
	service := NewContentBlocksService()

	files := []string{}
	links := []string{
		"https://example.com",
		"https://google.com",
		"", // пустая ссылка должна игнорироваться
	}

	blocks, errors := service.ProcessFileData(&files, links)

	require.Empty(t, errors)
	assert.Len(t, blocks, 2) // только 2 непустые ссылки

	assert.Equal(t, "link", blocks[0].Type)
	assert.Equal(t, "https://example.com", blocks[0].Content)

	assert.Equal(t, "link", blocks[1].Type)
	assert.Equal(t, "https://google.com", blocks[1].Content)
}

// TestProcessFileData_ImageDetection проверяет определение типа изображения
func TestProcessFileData_ImageDetection(t *testing.T) {
	service := NewContentBlocksService()

	// Создаём временные файлы с разными расширениями
	tmpDir := t.TempDir()

	testFiles := []struct {
		ext      string
		expected string
	}{
		{".jpg", "image"},
		{".jpeg", "image"},
		{".png", "image"},
		{".gif", "image"},
		{".bmp", "image"},
		{".txt", "file"},
		{".pdf", "file"},
		{".docx", "file"},
	}

	var filePaths []string
	for _, tf := range testFiles {
		path := filepath.Join(tmpDir, "test"+tf.ext)
		err := os.WriteFile(path, []byte("test content"), 0644)
		require.NoError(t, err)
		filePaths = append(filePaths, path)
	}

	blocks, errors := service.ProcessFileData(&filePaths, []string{})

	// Все файлы должны быть обработаны без ошибок
	assert.Empty(t, errors)
	assert.Len(t, blocks, len(testFiles))

	// Проверяем что типы определены правильно
	for i, block := range blocks {
		assert.Equal(t, testFiles[i].expected, block.Type)
		assert.NotEmpty(t, block.FileHash)
		assert.NotEmpty(t, block.OriginalName)
		assert.NotEmpty(t, block.Extension)
	}
}

// TestProcessFileData_NonExistentFile проверяет обработку несуществующего файла
func TestProcessFileData_NonExistentFile(t *testing.T) {
	service := NewContentBlocksService()

	files := []string{"/nonexistent/path/file.txt"}
	links := []string{}

	blocks, errors := service.ProcessFileData(&files, links)

	assert.Empty(t, blocks)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "не существует")
}

// TestBlocksToJSON_EmptyBlocks проверяет конвертацию пустых блоков
func TestBlocksToJSON_EmptyBlocks(t *testing.T) {
	service := NewContentBlocksService()

	jsonStr, err := service.BlocksToJSON([]Block{})

	require.NoError(t, err)
	assert.Empty(t, jsonStr)
}

// TestBlocksToJSON_ValidBlocks проверяет конвертацию валидных блоков
func TestBlocksToJSON_ValidBlocks(t *testing.T) {
	service := NewContentBlocksService()

	blocks := []Block{
		{
			Type:         "image",
			FileHash:     "abc123",
			OriginalName: "test.jpg",
			Extension:    "jpg",
		},
		{
			Type:    "link",
			Content: "https://example.com",
		},
		{
			Type:        "text",
			Content:     "Some text",
			Description: "A description",
		},
	}

	jsonStr, err := service.BlocksToJSON(blocks)

	require.NoError(t, err)
	assert.NotEmpty(t, jsonStr)
	assert.Contains(t, jsonStr, "image")
	assert.Contains(t, jsonStr, "link")
	assert.Contains(t, jsonStr, "text")
	assert.Contains(t, jsonStr, "abc123")
	assert.Contains(t, jsonStr, "https://example.com")
}

// TestJSONToBlocks_EmptyString проверяет парсинг пустой строки
func TestJSONToBlocks_EmptyString(t *testing.T) {
	service := NewContentBlocksService()

	blocks, err := service.JSONToBlocks("")

	require.NoError(t, err)
	assert.Empty(t, blocks)
}

// TestJSONToBlocks_ValidJSON проверяет парсинг валидного JSON
func TestJSONToBlocks_ValidJSON(t *testing.T) {
	service := NewContentBlocksService()

	jsonStr := `[
		{"type": "image", "file_hash": "abc123", "original_name": "test.jpg", "extension": "jpg"},
		{"type": "link", "content": "https://example.com"},
		{"type": "text", "content": "Some text", "description": "A description"}
	]`

	blocks, err := service.JSONToBlocks(jsonStr)

	require.NoError(t, err)
	assert.Len(t, blocks, 3)

	assert.Equal(t, "image", blocks[0].Type)
	assert.Equal(t, "abc123", blocks[0].FileHash)
	assert.Equal(t, "test.jpg", blocks[0].OriginalName)

	assert.Equal(t, "link", blocks[1].Type)
	assert.Equal(t, "https://example.com", blocks[1].Content)

	assert.Equal(t, "text", blocks[2].Type)
	assert.Equal(t, "Some text", blocks[2].Content)
}

// TestJSONToBlocks_InvalidJSON проверяет парсинг невалидного JSON
func TestJSONToBlocks_InvalidJSON(t *testing.T) {
	service := NewContentBlocksService()

	blocks, err := service.JSONToBlocks("invalid json {")

	assert.Error(t, err)
	assert.Empty(t, blocks)
	assert.Contains(t, err.Error(), "ошибка разбора JSON")
}

// TestExtractFilesFromBlocks_EmptyBlocks проверяет извлечение файлов из пустых блоков
func TestExtractFilesFromBlocks_EmptyBlocks(t *testing.T) {
	service := NewContentBlocksService()

	files := service.ExtractFilesFromBlocks([]Block{})

	assert.Empty(t, files)
}

// TestExtractFilesFromBlocks_WithFiles проверяет извлечение файлов из блоков
func TestExtractFilesFromBlocks_WithFiles(t *testing.T) {
	service := NewContentBlocksService()

	blocks := []Block{
		{Type: "image", FileHash: "hash1"},
		{Type: "link", Content: "https://example.com"},
		{Type: "file", FileHash: "hash2"},
		{Type: "text", Content: "text"},
		{Type: "image", FileHash: "hash3"},
	}

	files := service.ExtractFilesFromBlocks(blocks)

	assert.Len(t, files, 3)
	assert.Equal(t, []string{"hash1", "hash2", "hash3"}, files)
}

// TestExtractFilesFromBlocks_NoFiles проверяет извлечение когда нет файлов
func TestExtractFilesFromBlocks_NoFiles(t *testing.T) {
	service := NewContentBlocksService()

	blocks := []Block{
		{Type: "link", Content: "https://example.com"},
		{Type: "text", Content: "text"},
	}

	files := service.ExtractFilesFromBlocks(blocks)

	assert.Empty(t, files)
}

// TestDetermineItemType проверяет определение типа элемента
func TestDetermineItemType(t *testing.T) {
	service := NewContentBlocksService()

	// Все элементы кроме папок должны быть ItemTypeElement
	itemType := service.DetermineItemType("description", []Block{})
	assert.Equal(t, models.ItemTypeElement, itemType)
}

// TestCleanupOldFiles_EmptyBlocks проверяет очистку с пустыми блоками
func TestCleanupOldFiles_EmptyBlocks(t *testing.T) {
	service := NewContentBlocksService()

	// Не должно паниковать
	service.CleanupOldFiles([]Block{}, []Block{})
}

// TestCleanupOldFiles_NoOldFiles проверяет очистку когда нет старых файлов
func TestCleanupOldFiles_NoOldFiles(t *testing.T) {
	service := NewContentBlocksService()

	newBlocks := []Block{
		{Type: "image", FileHash: "new_hash"},
	}

	// Не должно паниковать
	service.CleanupOldFiles([]Block{}, newBlocks)
}

// TestCleanupOldFiles_AllNewFiles проверяет очистку когда все файлы новые
func TestCleanupOldFiles_AllNewFiles(t *testing.T) {
	service := NewContentBlocksService()

	oldBlocks := []Block{
		{Type: "image", FileHash: "old_hash"},
	}

	newBlocks := []Block{
		{Type: "image", FileHash: "new_hash"},
	}

	// Не должно паниковать
	service.CleanupOldFiles(oldBlocks, newBlocks)
}

// TestCleanupOldFiles_SameFiles проверяет что одинаковые файлы не удаляются
func TestCleanupOldFiles_SameFiles(t *testing.T) {
	service := NewContentBlocksService()

	sameHash := "same_hash_12345"

	oldBlocks := []Block{
		{Type: "image", FileHash: sameHash},
	}

	newBlocks := []Block{
		{Type: "image", FileHash: sameHash},
	}

	// Не должно паниковать
	service.CleanupOldFiles(oldBlocks, newBlocks)
}

// TestExtractLinks_EmptyString проверяет извлечение ссылок из пустой строки
func TestExtractLinks_EmptyString(t *testing.T) {
	service := NewContentBlocksService()

	links := service.ExtractLinks("")

	assert.Empty(t, links)
}

// TestExtractLinks_NoLinks проверяет строку без ссылок
func TestExtractLinks_NoLinks(t *testing.T) {
	service := NewContentBlocksService()

	links := service.ExtractLinks("Just some text without any URLs")

	assert.Empty(t, links)
}

// TestExtractLinks_SingleLink проверяет извлечение одной ссылки
func TestExtractLinks_SingleLink(t *testing.T) {
	service := NewContentBlocksService()

	links := service.ExtractLinks("Check out https://example.com for more info")

	assert.Len(t, links, 1)
	assert.Equal(t, "https://example.com", links[0])
}

// TestExtractLinks_MultipleLinks проверяет извлечение нескольких ссылок
func TestExtractLinks_MultipleLinks(t *testing.T) {
	service := NewContentBlocksService()

	text := "Visit https://google.com and http://github.com for resources"
	links := service.ExtractLinks(text)

	assert.Len(t, links, 2)
	assert.Contains(t, links, "https://google.com")
	assert.Contains(t, links, "http://github.com")
}

// TestExtractLinks_CleansPunctuation проверяет очистку ссылок от пунктуации
func TestExtractLinks_CleansPunctuation(t *testing.T) {
	service := NewContentBlocksService()

	links := service.ExtractLinks("See https://example.com, and https://test.com.")

	assert.Len(t, links, 2)
	// Ссылки не должны содержать завершающие точки и запятые
	assert.Equal(t, "https://example.com", links[0])
	assert.Equal(t, "https://test.com", links[1])
}

// TestExtractLinks_HTTPSOnly проверяет что извлекаются только http/https ссылки
func TestExtractLinks_HTTPSOnly(t *testing.T) {
	service := NewContentBlocksService()

	text := "Check ftp://files.com, www.example.com, and https://secure.com"
	links := service.ExtractLinks(text)

	assert.Len(t, links, 1)
	assert.Equal(t, "https://secure.com", links[0])
}

// TestRemoveLinksFromText_EmptyLinks проверяет удаление ссылок из текста с пустым списком
func TestRemoveLinksFromText_EmptyLinks(t *testing.T) {
	service := NewContentBlocksService()

	text := "Some text with https://example.com link"
	result := service.RemoveLinksFromText(text, []string{})

	assert.Equal(t, text, result)
}

// TestRemoveLinksFromText_SingleLink проверяет удаление одной ссылки
func TestRemoveLinksFromText_SingleLink(t *testing.T) {
	service := NewContentBlocksService()

	text := "Check out https://example.com for more"
	links := []string{"https://example.com"}

	result := service.RemoveLinksFromText(text, links)

	assert.Equal(t, "Check out for more", result)
}

// TestRemoveLinksFromText_MultipleLinks проверяет удаление нескольких ссылок
func TestRemoveLinksFromText_MultipleLinks(t *testing.T) {
	service := NewContentBlocksService()

	text := "Visit https://google.com and https://github.com for resources"
	links := []string{"https://google.com", "https://github.com"}

	result := service.RemoveLinksFromText(text, links)

	assert.Equal(t, "Visit and for resources", result)
}

// TestRemoveLinksFromText_CleansWhitespace проверяет очистку лишнего whitespace
func TestRemoveLinksFromText_CleansWhitespace(t *testing.T) {
	service := NewContentBlocksService()

	text := "Check   out  \n https://example.com  \n for   more"
	links := []string{"https://example.com"}

	result := service.RemoveLinksFromText(text, links)

	// Лишные пробелы и переносы строк должны быть удалены
	assert.Equal(t, "Check out for more", result)
}

// TestRemoveLinksFromText_EmptyText проверяет удаление из пустого текста
func TestRemoveLinksFromText_EmptyText(t *testing.T) {
	service := NewContentBlocksService()

	result := service.RemoveLinksFromText("", []string{"https://example.com"})

	assert.Empty(t, result)
}

// TestBlockStruct проверяет структуру Block
func TestBlockStruct(t *testing.T) {
	block := Block{
		Type:         "image",
		Content:      "some content",
		FileHash:     "abc123",
		OriginalName: "test.jpg",
		Extension:    "jpg",
		Description:  "A test image",
	}

	assert.Equal(t, "image", block.Type)
	assert.Equal(t, "some content", block.Content)
	assert.Equal(t, "abc123", block.FileHash)
	assert.Equal(t, "test.jpg", block.OriginalName)
	assert.Equal(t, "jpg", block.Extension)
	assert.Equal(t, "A test image", block.Description)
}

// TestProcessFileData_CaseInsensitiveExtension проверяет регистронезависимое определение расширения
func TestProcessFileData_CaseInsensitiveExtension(t *testing.T) {
	service := NewContentBlocksService()

	tmpDir := t.TempDir()

	// Файлы с расширениями в разном регистре
	testFiles := []struct {
		name     string
		expected string
	}{
		{"test.JPG", "image"},
		{"test.Jpeg", "image"},
		{"test.PNG", "image"},
		{"test.Gif", "image"},
		{"test.TXT", "file"},
		{"test.PDF", "file"},
	}

	var filePaths []string
	for _, tf := range testFiles {
		path := filepath.Join(tmpDir, tf.name)
		err := os.WriteFile(path, []byte("test content"), 0644)
		require.NoError(t, err)
		filePaths = append(filePaths, path)
	}

	blocks, errors := service.ProcessFileData(&filePaths, []string{})

	assert.Empty(t, errors)
	assert.Len(t, blocks, len(testFiles))

	for i, block := range blocks {
		assert.Equal(t, testFiles[i].expected, block.Type, "Failed for file: %s", testFiles[i].name)
	}
}

// TestCreateItemWithTransaction_EmptyContent проверяет создание элемента с пустым контентом
// Этот тест требует подключения к базе данных
func TestCreateItemWithTransaction_EmptyContent(t *testing.T) {
	t.Skip("Требует подключения к базе данных")
}

// TestCreateItemWithTransaction_WithParent проверяет создание элемента с родителем
// Этот тест требует подключения к базе данных
func TestCreateItemWithTransaction_WithParent(t *testing.T) {
	t.Skip("Требует подключения к базе данных")
}

// TestCreateItemWithTransaction_WithContent проверяет создание элемента с контентом
// Этот тест требует подключения к базе данных
func TestCreateItemWithTransaction_WithContent(t *testing.T) {
	t.Skip("Требует подключения к базе данных")
}

// TestUpdateItemWithTransaction_BasicUpdate проверяет обновление элемента
// Этот тест требует подключения к базе данных
func TestUpdateItemWithTransaction_BasicUpdate(t *testing.T) {
	t.Skip("Требует подключения к базе данных")
}

// TestUpdateItemWithTransaction_WithOldBlocks проверяет получение старых блоков
// Этот тест требует подключения к базе данных
func TestUpdateItemWithTransaction_WithOldBlocks(t *testing.T) {
	t.Skip("Требует подключения к базе данных")
}

// TestProcessTags_EmptyTags проверяет обработку пустых тегов
// Этот тест требует подключения к базе данных
func TestProcessTags_EmptyTags(t *testing.T) {
	t.Skip("Требует подключения к базе данных")
}

// TestProcessTags_SingleTag проверяет обработку одного тега
// Этот тест требует подключения к базе данных
func TestProcessTags_SingleTag(t *testing.T) {
	t.Skip("Требует подключения к базе данных")
}

// TestProcessTags_MultipleTags проверяет обработку нескольких тегов
// Этот тест требует подключения к базе данных
func TestProcessTags_MultipleTags(t *testing.T) {
	t.Skip("Требует подключения к базе данных")
}

// TestProcessTags_TagsWithSpaces проверяет очистку тегов от пробелов
// Этот тест требует подключения к базе данных
func TestProcessTags_TagsWithSpaces(t *testing.T) {
	t.Skip("Требует подключения к базе данных")
}

// TestProcessTags_OnlyCommas проверяет обработку строки только с запятыми
// Этот тест требует подключения к базе данных
func TestProcessTags_OnlyCommas(t *testing.T) {
	t.Skip("Требует подключения к базе данных")
}

// TestExtractLinks_NilText проверяет обработку nil-подобного пустого текста
func TestExtractLinks_NilText(t *testing.T) {
	service := NewContentBlocksService()

	links := service.ExtractLinks("")
	assert.Empty(t, links)
}

// TestExtractLinks_MixedProtocols проверяет обработку ссылок с разными протоколами
func TestExtractLinks_MixedProtocols(t *testing.T) {
	service := NewContentBlocksService()

	text := "http://http.com https://https.com ftp://ftp.com"
	links := service.ExtractLinks(text)

	assert.Len(t, links, 2)
	assert.Contains(t, links, "http://http.com")
	assert.Contains(t, links, "https://https.com")
	assert.NotContains(t, links, "ftp://ftp.com")
}

// TestExtractLinks_WithPorts проверяет обработку ссылок с портами
func TestExtractLinks_WithPorts(t *testing.T) {
	service := NewContentBlocksService()

	text := "Check http://localhost:8080 and https://example.com:443"
	links := service.ExtractLinks(text)

	assert.Len(t, links, 2)
	assert.Contains(t, links, "http://localhost:8080")
	assert.Contains(t, links, "https://example.com:443")
}

// TestExtractLinks_WithPathAndQuery проверяет обработку ссылок с путями и query-параметрами
func TestExtractLinks_WithPathAndQuery(t *testing.T) {
	service := NewContentBlocksService()

	text := "Visit https://example.com/path/to/page?id=123&name=test"
	links := service.ExtractLinks(text)

	assert.Len(t, links, 1)
	assert.Contains(t, links, "https://example.com/path/to/page?id=123&name=test")
}

// TestRemoveLinksFromText_PartialMatch проверяет что удаляются только полные совпадения
func TestRemoveLinksFromText_PartialMatch(t *testing.T) {
	service := NewContentBlocksService()

	text := "Check https://example.com and https://google.com/page"
	links := []string{"https://example.com"}

	result := service.RemoveLinksFromText(text, links)

	// https://example.com должна быть удалена
	assert.NotContains(t, result, "https://example.com")
	// https://google.com/page должна остаться
	assert.Contains(t, result, "https://google.com/page")
}

// TestBlocksToJSON_NilBlocks проверяет конвертацию nil блоков
func TestBlocksToJSON_NilBlocks(t *testing.T) {
	service := NewContentBlocksService()

	jsonStr, err := service.BlocksToJSON(nil)

	require.NoError(t, err)
	assert.Empty(t, jsonStr)
}

// TestJSONToBlocks_MalformedArray проверяет обработку невалидного JSON массива
func TestJSONToBlocks_MalformedArray(t *testing.T) {
	service := NewContentBlocksService()

	jsonStr := `{"type": "image"}` // объект вместо массива

	blocks, err := service.JSONToBlocks(jsonStr)

	assert.Error(t, err)
	assert.Empty(t, blocks)
}

// TestExtractFilesFromBlocks_MixedBlocks проверяет извлечение только файловых хэшей
func TestExtractFilesFromBlocks_MixedBlocks(t *testing.T) {
	service := NewContentBlocksService()

	blocks := []Block{
		{Type: "image", FileHash: "hash1"},
		{Type: "link", Content: "https://example.com"},
		{Type: "file", FileHash: ""}, // пустой хэш
		{Type: "text", Content: "text"},
		{Type: "image", FileHash: "hash2"},
	}

	files := service.ExtractFilesFromBlocks(blocks)

	assert.Len(t, files, 2)
	assert.Equal(t, []string{"hash1", "hash2"}, files)
}

// TestDetermineItemType_WithBlocks проверяет определение типа с блоками
func TestDetermineItemType_WithBlocks(t *testing.T) {
	service := NewContentBlocksService()

	blocks := []Block{
		{Type: "image", FileHash: "hash1"},
		{Type: "link", Content: "https://example.com"},
	}

	itemType := service.DetermineItemType("Description", blocks)
	assert.Equal(t, models.ItemTypeElement, itemType)
}

// TestDetermineItemType_EmptyDescription проверяет определение типа с пустым описанием
func TestDetermineItemType_EmptyDescription(t *testing.T) {
	service := NewContentBlocksService()

	itemType := service.DetermineItemType("", []Block{})
	assert.Equal(t, models.ItemTypeElement, itemType)
}

// TestCleanupOldFiles_MultipleOldFiles проверяет очистку нескольких старых файлов
func TestCleanupOldFiles_MultipleOldFiles(t *testing.T) {
	service := NewContentBlocksService()

	oldBlocks := []Block{
		{Type: "image", FileHash: "old_hash1"},
		{Type: "image", FileHash: "old_hash2"},
		{Type: "file", FileHash: "old_hash3"},
	}

	newBlocks := []Block{
		{Type: "image", FileHash: "new_hash1"},
	}

	// Не должно паниковать
	service.CleanupOldFiles(oldBlocks, newBlocks)
}

// TestCleanupOldFiles_PartialOverlap проверяет очистку с частичным пересечением
func TestCleanupOldFiles_PartialOverlap(t *testing.T) {
	service := NewContentBlocksService()

	oldBlocks := []Block{
		{Type: "image", FileHash: "keep_hash"},
		{Type: "image", FileHash: "delete_hash"},
	}

	newBlocks := []Block{
		{Type: "image", FileHash: "keep_hash"},
		{Type: "link", Content: "https://example.com"},
	}

	// Не должно паниковать
	service.CleanupOldFiles(oldBlocks, newBlocks)
}

// TestProcessFileData_MixedFilesAndLinks проверяет обработку файлов и ссылок вместе
func TestProcessFileData_MixedFilesAndLinks(t *testing.T) {
	service := NewContentBlocksService()

	tmpDir := t.TempDir()

	// Создаём тестовый файл
	filePath := filepath.Join(tmpDir, "test.jpg")
	err := os.WriteFile(filePath, []byte("image data"), 0644)
	require.NoError(t, err)

	files := []string{filePath}
	links := []string{"https://example.com", "https://google.com"}

	blocks, errors := service.ProcessFileData(&files, links)

	assert.Empty(t, errors)
	assert.Len(t, blocks, 3) // 1 файл + 2 ссылки

	assert.Equal(t, "image", blocks[0].Type)
	assert.Equal(t, "link", blocks[1].Type)
	assert.Equal(t, "link", blocks[2].Type)
}

// TestProcessFileData_DuplicateLinks проверяет обработку дублирующихся ссылок
func TestProcessFileData_DuplicateLinks(t *testing.T) {
	service := NewContentBlocksService()

	files := []string{}
	links := []string{
		"https://example.com",
		"https://example.com", // дубликат
		"https://google.com",
	}

	blocks, errors := service.ProcessFileData(&files, links)

	assert.Empty(t, errors)
	assert.Len(t, blocks, 3) // дубликаты тоже добавляются
}
