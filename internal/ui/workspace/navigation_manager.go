package workspace

import (
	"fmt"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// NavigationManager управляет навигацией по папкам
type NavigationManager struct {
	currentFolderID    int
	folderStack        []*models.Item       // Стек посещенных папок
	onBreadcrumbUpdate func([]*models.Item) // Колбэк для обновления хлебных крошек
}

// NewNavigationManager создает новый менеджер навигации
func NewNavigationManager() *NavigationManager {
	return &NavigationManager{
		currentFolderID: 0, // Корневая папка
		folderStack:     make([]*models.Item, 0),
	}
}

// SetBreadcrumbUpdateCallback устанавливает колбэк для обновления хлебных крошек
func (nm *NavigationManager) SetBreadcrumbUpdateCallback(callback func([]*models.Item)) {
	nm.onBreadcrumbUpdate = callback
}

// GoToFolder переходит в указанную папку
func (nm *NavigationManager) GoToFolder(folderID int) error {
	// Получаем информацию о папке
	folder, err := queries.GetItemByID(folderID)
	if err != nil {
		return err
	}

	// Проверяем, что это действительно папка
	if folder.Type != models.ItemTypeFolder {
		return fmt.Errorf("элемент с ID %d не является папкой", folderID)
	}

	// Добавляем текущую папку в стек перед переходом
	if nm.currentFolderID != 0 {
		currentFolder, _ := queries.GetItemByID(nm.currentFolderID)
		if currentFolder != nil {
			nm.folderStack = append(nm.folderStack, currentFolder)
		}
	}

	// Обновляем текущую папку
	nm.currentFolderID = folderID

	// Уведомляем об обновлении хлебных крошек
	if nm.onBreadcrumbUpdate != nil {
		nm.updateBreadcrumbs()
	}

	return nil
}

// GoToFolderInPath переходит к папке в пути, удаляя все последующие папки из стека
func (nm *NavigationManager) GoToFolderInPath(folderID int) error {
	// Проверяем, является ли папка целевой папкой
	if folderID == nm.currentFolderID {
		return nil // Уже находимся в этой папке
	}

	// Проверяем, есть ли папка в стеке
	targetIndex := -1
	for i, folder := range nm.folderStack {
		if folder.ID == folderID {
			targetIndex = i
			break
		}
	}

	if targetIndex != -1 {
		// Удаляем все папки после найденной (включая текущую папку)
		nm.folderStack = nm.folderStack[:targetIndex]
		// Переходим к найденной папке
		nm.currentFolderID = folderID
	} else {
		// Если папки нет в стеке, проверяем, может быть это корневая папка
		if folderID == 0 {
			return nm.GoToRoot()
		}

		// Иначе ищем папку в базе данных
		folder, err := queries.GetItemByID(folderID)
		if err != nil {
			return err
		}

		// Проверяем, что это действительно папка
		if folder.Type != models.ItemTypeFolder {
			return fmt.Errorf("элемент с ID %d не является папкой", folderID)
		}

		// Добавляем текущую папку в стек перед переходом (как в GoToFolder)
		if nm.currentFolderID != 0 {
			currentFolder, _ := queries.GetItemByID(nm.currentFolderID)
			if currentFolder != nil {
				nm.folderStack = append(nm.folderStack, currentFolder)
			}
		}

		// Переходим к новой папке
		nm.currentFolderID = folderID
	}

	// Уведомляем об обновлении хлебных крошек
	if nm.onBreadcrumbUpdate != nil {
		nm.updateBreadcrumbs()
	}

	return nil
}

// GoBack возвращается на предыдущую папку
func (nm *NavigationManager) GoBack() error {
	if len(nm.folderStack) == 0 {
		// Если стек пуст, возвращаемся в корень
		return nm.GoToRoot()
	}

	// Берем последнюю папку из стека
	lastIndex := len(nm.folderStack) - 1
	folder := nm.folderStack[lastIndex]
	nm.folderStack = nm.folderStack[:lastIndex]

	// Переходим в папку
	nm.currentFolderID = folder.ID
	if nm.onBreadcrumbUpdate != nil {
		nm.updateBreadcrumbs()
	}

	return nil
}

// GoToRoot возвращается в корневую папку
func (nm *NavigationManager) GoToRoot() error {
	nm.currentFolderID = 0
	nm.folderStack = make([]*models.Item, 0)

	if nm.onBreadcrumbUpdate != nil {
		nm.updateBreadcrumbs()
	}

	return nil
}

// GetCurrentFolderID возвращает ID текущей папки
func (nm *NavigationManager) GetCurrentFolderID() int {
	return nm.currentFolderID
}

// GetFolderStack возвращает стек папок
func (nm *NavigationManager) GetFolderStack() []*models.Item {
	return nm.folderStack
}

// GetFullPath возвращает полный путь (включая текущую папку)
func (nm *NavigationManager) GetFullPath() []*models.Item {
	var fullPath []*models.Item

	// Всегда добавляем корневую папку в начало пути
	rootFolder, _ := queries.GetItemByID(0)
	if rootFolder != nil {
		fullPath = append(fullPath, rootFolder)
	}

	// Добавляем папки из стека
	for _, folder := range nm.folderStack {
		// Пропускаем корневую папку, если она повторяется
		if folder.ID != 0 {
			fullPath = append(fullPath, folder)
		}
	}

	// Добавляем текущую папку, если она не корневая
	if nm.currentFolderID != 0 {
		currentFolder, _ := queries.GetItemByID(nm.currentFolderID)
		if currentFolder != nil {
			// Проверяем, не дублируется ли текущая папка
			isDuplicate := false
			for _, folder := range fullPath {
				if folder.ID == nm.currentFolderID {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				fullPath = append(fullPath, currentFolder)
			}
		}
	}

	return fullPath
}

// updateBreadcrumbs обновляет хлебные крошки
func (nm *NavigationManager) updateBreadcrumbs() {
	fullPath := nm.GetFullPath()
	if nm.onBreadcrumbUpdate != nil {
		nm.onBreadcrumbUpdate(fullPath)
	}
}
