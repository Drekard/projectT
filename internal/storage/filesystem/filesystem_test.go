package filesystem

import (
	"testing"
)

func TestFilesystem(t *testing.T) {
	// Инициализируем структуру хранения
	err := EnsureStorageStructure()
	if err != nil {
		t.Fatalf("Failed to initialize storage structure: %v", err)
	}

	// Тестируем сохранение файла
	testData := []byte("Hello, World!")
	fileData, err := SaveFile(testData)
	if err != nil {
		t.Fatalf("Failed to save file: %v", err)
	}

	if fileData.Hash == "" {
		t.Fatal("Expected file hash to be non-empty")
	}

	if fileData.Size != int64(len(testData)) {
		t.Fatalf("Expected file size to be %d, got %d", len(testData), fileData.Size)
	}

	// Тестируем чтение файла
	readData, err := ReadFile(fileData.Hash)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(readData) != string(testData) {
		t.Fatalf("Expected file content to be %s, got %s", testData, readData)
	}

	// Тестируем проверку существования файла
	if !Exists(fileData.Hash) {
		t.Fatal("Expected file to exist")
	}

	// Тестируем получение информации о файле
	fileInfo, err := GetFileInfo(fileData.Hash)
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}

	if fileInfo.Hash != fileData.Hash {
		t.Fatalf("Expected file hash to be %s, got %s", fileData.Hash, fileInfo.Hash)
	}

	// Тестируем удаление файла
	err = DeleteFile(fileData.Hash)
	if err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	// Проверяем, что файл удален
	if Exists(fileData.Hash) {
		t.Fatal("Expected file to be deleted")
	}

	t.Logf("File hash: %s", fileData.Hash)
	t.Log("All tests passed!")
}
