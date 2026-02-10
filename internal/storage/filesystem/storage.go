package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// StorageDir основная директория для хранения файлов
	StorageDir = "./internal/storage"
	// FilesDir поддиректория для файлов
	FilesDir = "files"
)

// EnsureStorageStructure инициализирует структуру папок при старте приложения
func EnsureStorageStructure() error {
	// Создаем основную директорию для хранения
	if err := os.MkdirAll(StorageDir, 0755); err != nil {
		return err
	}

	// Создаем директорию для файлов
	filesPath := filepath.Join(StorageDir, FilesDir)
	if err := os.MkdirAll(filesPath, 0755); err != nil {
		return err
	}

	// Создаем поддиректории для хэшей (00-ff)
	if err := CreateHashDirectories(); err != nil {
		return err
	}

	return nil
}

// CreateHashDirectories создает 256 поддиректорий для хранения файлов по первым 2 символам хэша
func CreateHashDirectories() error {
	// Создаем директории от 00 до ff
	for i := 0; i < 256; i++ {
		dirName := fmt.Sprintf("%02x", i)
		dirPath := filepath.Join(StorageDir, FilesDir, dirName)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return err
		}
	}
	return nil
}

// GetFilePathWithExtension возвращает полный путь к файлу по его хэшу и расширению
func GetFilePathWithExtension(hash string, ext string) string {
	filename := hash
	if ext != "" {
		// Добавляем точку перед расширением, если его нет
		if ext[0] != '.' {
			ext = "." + ext
		}
		filename = hash + ext
	}

	if len(hash) < 2 {
		// Если хэш слишком короткий, возвращаем путь в корневую директорию файлов
		return filepath.Join(StorageDir, FilesDir, filename)
	}

	// Формируем путь: storage/files/{первые_2_символа_хэша}/{полный_хэш.расширение}
	prefix := hash[:2]
	return filepath.Join(StorageDir, FilesDir, prefix, filename)
}

// GetFilePath возвращает полный путь к файлу по его хэшу (без расширения для обратной совместимости)
func GetFilePath(hash string) string {
	return GetFilePathWithExtension(hash, "")
}

// EnsureParentDir создает родительскую директорию для файла, если она не существует
func EnsureParentDir(filePath string) error {
	parentDir := filepath.Dir(filePath)
	return os.MkdirAll(parentDir, 0755)
}
