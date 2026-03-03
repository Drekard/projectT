package filesystem

import (
	"os"
	"path/filepath"
)

var (
	// storageRoot корневая директория для хранения файлов
	storageRoot = "./storage"
	// filesDir поддиректория для файлов
	filesDir = "files"
)

// StorageConfig конфигурация файлового хранилища
type StorageConfig interface {
	GetPath() string
	GetFilesDir() string
}

// defaultStorageConfig конфигурация по умолчанию для обратной совместимости
type defaultStorageConfig struct{}

func (defaultStorageConfig) GetPath() string     { return "./storage" }
func (defaultStorageConfig) GetFilesDir() string { return "files" }

// InitStorage инициализирует файловое хранилище с заданной конфигурацией
func InitStorage(cfg StorageConfig) {
	storageRoot = cfg.GetPath()
	filesDir = cfg.GetFilesDir()
}

// EnsureStorageStructure инициализирует структуру папок при старте приложения
func EnsureStorageStructure() error {
	// Создаем основную директорию для хранения
	if err := os.MkdirAll(storageRoot, 0755); err != nil {
		return err
	}

	// Создаем директорию для файлов
	filesPath := filepath.Join(storageRoot, filesDir)
	if err := os.MkdirAll(filesPath, 0755); err != nil {
		return err
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
		return filepath.Join(storageRoot, filesDir, filename)
	}

	// Формируем путь: storage/files/{первые_2_символа_хэша}/{полный_хэш.расширение}
	prefix := hash[:2]
	return filepath.Join(storageRoot, filesDir, prefix, filename)
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

// GetStorageRoot возвращает корневую директорию хранилища
func GetStorageRoot() string {
	return storageRoot
}

// GetFilesDir возвращает директорию для файлов
func GetFilesDir() string {
	return filesDir
}
