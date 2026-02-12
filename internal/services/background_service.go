package services

import (
	"fmt"
	"projectT/internal/storage/database/queries"
)

// BackgroundService предоставляет сервис для работы с фоновыми изображениями
type BackgroundService struct{}

// NewBackgroundService создает новый экземпляр сервиса фона
func NewBackgroundService() *BackgroundService {
	return &BackgroundService{}
}

// SetBackground устанавливает фоновое изображение для профиля
func (bs *BackgroundService) SetBackground(backgroundPath string) error {
	fmt.Printf("DEBUG: Service - Установка фона: %s\n", backgroundPath)
	
	// Получаем текущий профиль
	profile, err := queries.GetProfile()
	if err != nil {
		fmt.Printf("DEBUG: Service - Ошибка получения профиля: %v\n", err)
		return err
	}

	// Обновляем путь к фоновому изображению
	profile.BackgroundPath = backgroundPath
	
	// Сохраняем изменения в базу данных
	err = queries.UpdateProfileField("background_path", backgroundPath, profile.ID)
	if err != nil {
		fmt.Printf("DEBUG: Service - Ошибка сохранения пути к фону: %v\n", err)
		return err
	}
	fmt.Printf("DEBUG: Service - Фон успешно сохранен в базу данных: %s\n", backgroundPath)

	// Уведомляем всех подписчиков об изменении фона
	eventManager := GetBackgroundEventManager()
	fmt.Println("DEBUG: Service - Отправка уведомления об изменении фона")
	eventManager.Notify("background_changed")

	return nil
}

// ClearBackground очищает фоновое изображение для профиля
func (bs *BackgroundService) ClearBackground() error {
	fmt.Println("DEBUG: Service - Очистка фона")
	
	// Получаем текущий профиль
	profile, err := queries.GetProfile()
	if err != nil {
		fmt.Printf("DEBUG: Service - Ошибка получения профиля: %v\n", err)
		return err
	}

	// Очищаем путь к фоновому изображению
	backgroundPath := ""
	
	// Сохраняем изменения в базу данных
	err = queries.UpdateProfileField("background_path", backgroundPath, profile.ID)
	if err != nil {
		fmt.Printf("DEBUG: Service - Ошибка очистки фона: %v\n", err)
		return err
	}
	fmt.Println("DEBUG: Service - Фон успешно очищен в базе данных")

	// Уведомляем всех подписчиков об изменении фона
	eventManager := GetBackgroundEventManager()
	fmt.Println("DEBUG: Service - Отправка уведомления об очистке фона")
	eventManager.Notify("background_cleared")

	return nil
}

// GetCurrentBackground возвращает текущий путь к фоновому изображению
func (bs *BackgroundService) GetCurrentBackground() (string, error) {
	profile, err := queries.GetProfile()
	if err != nil {
		return "", err
	}

	return profile.BackgroundPath, nil
}