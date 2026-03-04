package profile

import (
	"testing"

	"projectT/internal/storage/database"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

func TestNew(t *testing.T) {
	t.Skip("Тест требует GUI и не может быть запущен в headless режиме")

	// Инициализируем тестовую БД
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}
	defer database.CloseDB()

	// Сохраняем оригинальную БД
	originalDB := database.DB
	database.DB = db
	defer func() { database.DB = originalDB }()

	// Запускаем миграции
	database.RunMigrations()

	// Создаём профиль по умолчанию
	_ = queries.CreateProfile(&models.Profile{
		Username:              "test",
		Status:                "test",
		AvatarPath:            "",
		BackgroundPath:        "",
		ContentCharacteristic: "",
	})

	// Тестируем создание нового экземпляра UI
	ui := New()

	if ui == nil {
		t.Error("Expected UI instance, got nil")
		return
	}

	// Проверяем, что gridManager инициализирован
	//
	//nolint:staticcheck // Проверяем, что gridManager инициализирован
	if ui.gridManager == nil {
		t.Error("Expected gridManager to be initialized, got nil")
	}
}
