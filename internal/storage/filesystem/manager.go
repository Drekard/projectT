package filesystem

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
)

// FileData содержит информацию о файле
type FileData struct {
	Hash     string
	Size     int64
	MimeType string
	Path     string
}

// SaveFile сохраняет файл на диск по хэш-ориентированной структуре
// Принимает содержимое файла и возвращает хэш, размер и MIME-тип
func SaveFile(fileBytes []byte) (*FileData, error) {
	return SaveFileWithOriginalName(fileBytes, "")
}

// SaveFileWithOriginalName сохраняет файл на диск по хэш-ориентированной структуре с сохранением расширения оригинального файла
// Принимает содержимое файла и оригинальное имя файла, возвращает хэш, размер и MIME-тип
func SaveFileWithOriginalName(fileBytes []byte, originalName string) (*FileData, error) {
	// Вычисляем хэш файла
	hash := CalculateHash(fileBytes)

	// Извлекаем расширение из оригинального имени файла
	ext := filepath.Ext(originalName)

	// Получаем путь для сохранения файла
	filePath := GetFilePathWithExtension(hash, ext)

	// Проверяем, существует ли уже файл с таким хэшем (и расширением)
	if _, err := os.Stat(filePath); err == nil {
		// Файл уже существует, возвращаем информацию о нем
		info, err := os.Stat(filePath)
		if err != nil {
			return nil, err
		}

		// Определяем MIME-тип на основе расширения или содержимого
		mimeType := mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = detectMimeType(fileBytes)
		}

		return &FileData{
			Hash:     hash,
			Size:     info.Size(),
			MimeType: mimeType,
			Path:     filePath,
		}, nil
	}

	// Создаем родительскую директорию, если она не существует
	if err := EnsureParentDir(filePath); err != nil {
		return nil, fmt.Errorf("ошибка создания директории: %w", err)
	}

	// Сохраняем файл
	if err := os.WriteFile(filePath, fileBytes, 0644); err != nil {
		return nil, fmt.Errorf("ошибка сохранения файла: %w", err)
	}

	// Определяем MIME-тип
	mimeType := detectMimeType(fileBytes)
	if ext != "" {
		// Если расширение есть, используем его для MIME-типа
		tempMimeType := mime.TypeByExtension(ext)
		if tempMimeType != "" {
			mimeType = tempMimeType
		}
	}

	// Получаем информацию о файле
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения информации о файле: %w", err)
	}

	return &FileData{
		Hash:     hash,
		Size:     info.Size(),
		MimeType: mimeType,
		Path:     filePath,
	}, nil
}

// ReadFile читает файл по его хэшу и возвращает его содержимое
func ReadFile(hash string) ([]byte, error) {
	filePath := GetFilePathByHash(hash)

	// Проверяем существование файла
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("файл с хэшем %s не найден", hash)
	}

	// Читаем файл
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла: %w", err)
	}

	return fileBytes, nil
}

// DeleteFile удаляет файл с диска
func DeleteFile(hash string) error {
	filePath := GetFilePathByHash(hash)

	// Проверяем существование файла
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("файл с хэшем %s не найден", hash)
	}

	// Удаляем файл
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("ошибка удаления файла: %w", err)
	}

	return nil
}

// GetFileInfo возвращает информацию о файле по его хэшу
func GetFileInfo(hash string) (*FileData, error) {
	filePath := GetFilePathByHash(hash)

	// Получаем информацию о файле
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("файл с хэшем %s не найден", hash)
		}
		return nil, fmt.Errorf("ошибка получения информации о файле: %w", err)
	}

	// Читаем первые байты файла для определения MIME-типа
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("ошибка чтения файла: %w", err)
	}

	mimeType := detectMimeType(buffer)

	return &FileData{
		Hash:     hash,
		Size:     info.Size(),
		MimeType: mimeType,
		Path:     filePath,
	}, nil
}

// detectMimeType определяет MIME-тип на основе содержимого файла
func detectMimeType(fileBytes []byte) string {
	mimeType := http.DetectContentType(fileBytes)
	return mimeType
}

// Exists проверяет, существует ли файл с заданным хэшем
func Exists(hash string) bool {
	filePath := GetFilePathByHash(hash)
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// GetFilePathByHash возвращает путь к файлу по его хэшу
func GetFilePathByHash(hash string) string {
	// Сначала пробуем найти файл с расширением
	// Проверяем наиболее распространенные расширения
	extensions := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg", ".webp", ".pdf", ".txt", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".mp3", ".mp4", ".avi", ".mkv", ".zip", ".rar", ".7z", ".exe", ".msi", ".dll", ".py", ".js", ".ts", ".go", ".java", ".cpp", ".c", ".h", ".html", ".css", ".json", ".xml", ".csv", ".rtf", ".odt", ".ods", ".odp"}

	for _, ext := range extensions {
		pathWithExt := GetFilePathWithExtension(hash, ext)
		if _, err := os.Stat(pathWithExt); err == nil {
			// Файл с этим расширением существует
			return pathWithExt
		}
	}

	// Если файл с расширением не найден, возвращаем путь без расширения (для обратной совместимости)
	return GetFilePath(hash)
}

// ReadFileByHash читает файл по его хэшу и возвращает его содержимое и информацию
func ReadFileByHash(hash string) ([]byte, *FileData, error) {
	// Проверяем, существует ли файл
	if !Exists(hash) {
		return nil, nil, fmt.Errorf("файл с хэшем %s не найден", hash)
	}

	// Читаем содержимое файла
	content, err := ReadFile(hash)
	if err != nil {
		return nil, nil, err
	}

	// Получаем информацию о файле
	fileInfo, err := GetFileInfo(hash)
	if err != nil {
		return nil, nil, err
	}

	return content, fileInfo, nil
}
