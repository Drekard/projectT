package profile

import (
	"testing"
)

func TestNew(t *testing.T) {
	// Тестируем создание нового экземпляра UI
	ui := New()
	
	if ui == nil {
		t.Error("Expected UI instance, got nil")
	}
	
	// Проверяем, что gridManager инициализирован
	if ui.gridManager == nil {
		t.Error("Expected gridManager to be initialized, got nil")
	}
}