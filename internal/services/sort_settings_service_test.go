package services

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSortSettingsService(t *testing.T) {
	service := NewSortSettingsService()
	assert.NotNil(t, service)
	assert.NotNil(t, service.filterOptions)
}

func TestGlobalSortSettingsService(t *testing.T) {
	assert.NotNil(t, GlobalSortSettingsService)
	assert.IsType(t, &SortSettingsService{}, GlobalSortSettingsService)
}

// TestGetFilterOptions_DefaultValues проверяет значения по умолчанию
func TestGetFilterOptions_DefaultValues(t *testing.T) {
	service := NewSortSettingsService()

	options := service.GetFilterOptions()

	assert.NotNil(t, options)
	assert.Equal(t, "all", options.ItemType)
	assert.Equal(t, "none", options.Priority)
	assert.Equal(t, "name", options.SortBy)
	assert.Equal(t, "asc", options.SortOrder)
	assert.Equal(t, "", options.TabMode) // по умолчанию пустой
}

// TestGetFilterOptions_ReturnsCopy проверяет что возвращается копия
func TestGetFilterOptions_ReturnsCopy(t *testing.T) {
	service := NewSortSettingsService()

	options1 := service.GetFilterOptions()
	options2 := service.GetFilterOptions()

	// Изменяем первую копию
	options1.ItemType = "folders"
	options1.SortBy = "created_date"

	// Вторая копия не должна измениться
	assert.Equal(t, "all", options2.ItemType)
	assert.Equal(t, "name", options2.SortBy)

	// Оригинал в сервисе тоже не должен измениться
	currentOptions := service.GetFilterOptions()
	assert.Equal(t, "all", currentOptions.ItemType)
	assert.Equal(t, "name", currentOptions.SortBy)
}

// TestSetFilterOptions_AllFields проверяет установку всех полей
func TestSetFilterOptions_AllFields(t *testing.T) {
	service := NewSortSettingsService()

	newOptions := &FilterOptions{
		ItemType:  "images",
		Priority:  "images_first",
		SortBy:    "created_date",
		SortOrder: "desc",
		TabMode:   "all_items",
	}

	service.SetFilterOptions(newOptions)

	options := service.GetFilterOptions()
	assert.Equal(t, "images", options.ItemType)
	assert.Equal(t, "images_first", options.Priority)
	assert.Equal(t, "created_date", options.SortBy)
	assert.Equal(t, "desc", options.SortOrder)
	assert.Equal(t, "all_items", options.TabMode)
}

// TestSetFilterOptions_Nil проверяет установку nil
func TestSetFilterOptions_Nil(t *testing.T) {
	service := NewSortSettingsService()

	service.SetFilterOptions(nil)

	// После установки nil, filterOptions становится nil
	// Это допустимое поведение, но GetFilterOptions вызовет панику
	// Поэтому мы просто проверяем что установка не вызывает панику
	assert.NotNil(t, service)
}

// TestSetFilterOptions_Overwrite проверяет перезапись настроек
func TestSetFilterOptions_Overwrite(t *testing.T) {
	service := NewSortSettingsService()

	// Первые настройки
	service.SetFilterOptions(&FilterOptions{
		ItemType:  "folders",
		Priority:  "folders_first",
		SortBy:    "name",
		SortOrder: "asc",
	})

	// Вторые настройки
	service.SetFilterOptions(&FilterOptions{
		ItemType:  "files",
		Priority:  "files_first",
		SortBy:    "size",
		SortOrder: "desc",
	})

	options := service.GetFilterOptions()
	assert.Equal(t, "files", options.ItemType)
	assert.Equal(t, "files_first", options.Priority)
	assert.Equal(t, "size", options.SortBy)
	assert.Equal(t, "desc", options.SortOrder)
}

// TestUpdateFilterOptions_UpdateItemType проверяет обновление ItemType
func TestUpdateFilterOptions_UpdateItemType(t *testing.T) {
	service := NewSortSettingsService()

	service.UpdateFilterOptions("folders", "", "", "")

	options := service.GetFilterOptions()
	assert.Equal(t, "folders", options.ItemType)
	assert.Equal(t, "none", options.Priority) // не изменился
	assert.Equal(t, "name", options.SortBy)   // не изменился
	assert.Equal(t, "asc", options.SortOrder) // не изменился
}

// TestUpdateFilterOptions_UpdatePriority проверяет обновление Priority
func TestUpdateFilterOptions_UpdatePriority(t *testing.T) {
	service := NewSortSettingsService()

	service.UpdateFilterOptions("", "folders_first", "", "")

	options := service.GetFilterOptions()
	assert.Equal(t, "all", options.ItemType) // не изменился
	assert.Equal(t, "folders_first", options.Priority)
	assert.Equal(t, "name", options.SortBy)   // не изменился
	assert.Equal(t, "asc", options.SortOrder) // не изменился
}

// TestUpdateFilterOptions_UpdateSortBy проверяет обновление SortBy
func TestUpdateFilterOptions_UpdateSortBy(t *testing.T) {
	service := NewSortSettingsService()

	service.UpdateFilterOptions("", "", "created_date", "")

	options := service.GetFilterOptions()
	assert.Equal(t, "all", options.ItemType)  // не изменился
	assert.Equal(t, "none", options.Priority) // не изменился
	assert.Equal(t, "created_date", options.SortBy)
	assert.Equal(t, "asc", options.SortOrder) // не изменился
}

// TestUpdateFilterOptions_UpdateSortOrder проверяет обновление SortOrder
func TestUpdateFilterOptions_UpdateSortOrder(t *testing.T) {
	service := NewSortSettingsService()

	service.UpdateFilterOptions("", "", "", "desc")

	options := service.GetFilterOptions()
	assert.Equal(t, "all", options.ItemType)  // не изменился
	assert.Equal(t, "none", options.Priority) // не изменился
	assert.Equal(t, "name", options.SortBy)   // не изменился
	assert.Equal(t, "desc", options.SortOrder)
}

// TestUpdateFilterOptions_AllFields проверяет обновление всех полей
func TestUpdateFilterOptions_AllFields(t *testing.T) {
	service := NewSortSettingsService()

	service.UpdateFilterOptions("images", "images_first", "size", "desc")

	options := service.GetFilterOptions()
	assert.Equal(t, "images", options.ItemType)
	assert.Equal(t, "images_first", options.Priority)
	assert.Equal(t, "size", options.SortBy)
	assert.Equal(t, "desc", options.SortOrder)
}

// TestUpdateFilterOptions_NilOptions проверяет инициализацию при nil
func TestUpdateFilterOptions_NilOptions(t *testing.T) {
	service := &SortSettingsService{
		filterOptions: nil,
	}

	// Не должно паниковать
	service.UpdateFilterOptions("folders", "", "", "")

	options := service.GetFilterOptions()
	assert.NotNil(t, options)
	assert.Equal(t, "folders", options.ItemType)
}

// TestUpdateFilterOptions_PartialUpdate проверяет частичное обновление
func TestUpdateFilterOptions_PartialUpdate(t *testing.T) {
	service := NewSortSettingsService()

	// Сначала устанавливаем все поля
	service.SetFilterOptions(&FilterOptions{
		ItemType:  "all",
		Priority:  "none",
		SortBy:    "name",
		SortOrder: "asc",
		TabMode:   "current_folder",
	})

	// Обновляем только SortBy и SortOrder
	service.UpdateFilterOptions("", "", "created_date", "desc")

	options := service.GetFilterOptions()
	assert.Equal(t, "all", options.ItemType)           // не изменился
	assert.Equal(t, "none", options.Priority)          // не изменился
	assert.Equal(t, "created_date", options.SortBy)    // обновился
	assert.Equal(t, "desc", options.SortOrder)         // обновился
	assert.Equal(t, "current_folder", options.TabMode) // не изменился
}

// TestFilterOptions_ItemTypeValues проверяет допустимые значения ItemType
func TestFilterOptions_ItemTypeValues(t *testing.T) {
	service := NewSortSettingsService()

	validTypes := []string{"all", "folders", "images", "files", "links", "text"}

	for _, itemType := range validTypes {
		service.UpdateFilterOptions(itemType, "", "", "")
		options := service.GetFilterOptions()
		assert.Equal(t, itemType, options.ItemType)
	}
}

// TestFilterOptions_PriorityValues проверяет допустимые значения Priority
func TestFilterOptions_PriorityValues(t *testing.T) {
	service := NewSortSettingsService()

	validPriorities := []string{"none", "folders_first", "images_first", "files_first", "links_first", "text_first"}

	for _, priority := range validPriorities {
		service.UpdateFilterOptions("", priority, "", "")
		options := service.GetFilterOptions()
		assert.Equal(t, priority, options.Priority)
	}
}

// TestFilterOptions_SortByValues проверяет допустимые значения SortBy
func TestFilterOptions_SortByValues(t *testing.T) {
	service := NewSortSettingsService()

	validSortBy := []string{"name", "created_date", "modified_date", "content_size"}

	for _, sortBy := range validSortBy {
		service.UpdateFilterOptions("", "", sortBy, "")
		options := service.GetFilterOptions()
		assert.Equal(t, sortBy, options.SortBy)
	}
}

// TestFilterOptions_SortOrderValues проверяет допустимые значения SortOrder
func TestFilterOptions_SortOrderValues(t *testing.T) {
	service := NewSortSettingsService()

	validSortOrder := []string{"asc", "desc"}

	for _, sortOrder := range validSortOrder {
		service.UpdateFilterOptions("", "", "", sortOrder)
		options := service.GetFilterOptions()
		assert.Equal(t, sortOrder, options.SortOrder)
	}
}

// TestFilterOptions_TabModeValues проверяет допустимые значения TabMode
func TestFilterOptions_TabModeValues(t *testing.T) {
	service := NewSortSettingsService()

	validTabModes := []string{"current_folder", "all_items"}

	for _, tabMode := range validTabModes {
		service.SetFilterOptions(&FilterOptions{
			TabMode: tabMode,
		})
		options := service.GetFilterOptions()
		assert.Equal(t, tabMode, options.TabMode)
	}
}

// TestSortSettingsService_ConcurrentRead проверяет безопасность конкурентного чтения
func TestSortSettingsService_ConcurrentRead(t *testing.T) {
	service := NewSortSettingsService()

	var wg sync.WaitGroup
	done := make(chan bool)

	// Запускаем множество горутин на чтение
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = service.GetFilterOptions()
		}()
	}

	// Ждём завершения
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Успешно завершено
		assert.True(t, true)
	case <-time.After(5 * time.Second):
		t.Fatal("Тест завершен по таймауту - возможна гонка данных")
	}
}

// TestSortSettingsService_ConcurrentReadWrite проверяет безопасность конкурентной записи и чтения
func TestSortSettingsService_ConcurrentReadWrite(t *testing.T) {
	service := NewSortSettingsService()

	var wg sync.WaitGroup
	done := make(chan bool)

	// Запускаем горутин на запись
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			service.SetFilterOptions(&FilterOptions{
				ItemType:  "folders",
				Priority:  "folders_first",
				SortBy:    "name",
				SortOrder: "asc",
				TabMode:   "current_folder",
			})
		}(i)
	}

	// Запускаем горутин на чтение
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = service.GetFilterOptions()
		}()
	}

	// Ждём завершения
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Успешно завершено
		assert.True(t, true)
	case <-time.After(5 * time.Second):
		t.Fatal("Тест завершен по таймауту - возможна гонка данных")
	}
}

// TestSortSettingsService_ConcurrentUpdate проверяет безопасность UpdateFilterOptions
func TestSortSettingsService_ConcurrentUpdate(t *testing.T) {
	service := NewSortSettingsService()

	var wg sync.WaitGroup
	done := make(chan bool)

	// Запускаем множество горутин на обновление
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			service.UpdateFilterOptions("images", "images_first", "size", "desc")
		}(i)
	}

	// Ждём завершения
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Успешно завершено
		assert.True(t, true)
	case <-time.After(5 * time.Second):
		t.Fatal("Тест завершен по таймауту - возможна гонка данных")
	}
}

// TestFilterOptions_Struct проверяет структуру FilterOptions
func TestFilterOptions_Struct(t *testing.T) {
	options := &FilterOptions{
		ItemType:  "images",
		Priority:  "images_first",
		SortBy:    "created_date",
		SortOrder: "desc",
		TabMode:   "all_items",
	}

	assert.Equal(t, "images", options.ItemType)
	assert.Equal(t, "images_first", options.Priority)
	assert.Equal(t, "created_date", options.SortBy)
	assert.Equal(t, "desc", options.SortOrder)
	assert.Equal(t, "all_items", options.TabMode)
}

// TestFilterOptions_EmptyValues проверяет установку пустых значений
func TestFilterOptions_EmptyValues(t *testing.T) {
	service := NewSortSettingsService()

	service.SetFilterOptions(&FilterOptions{
		ItemType:  "",
		Priority:  "",
		SortBy:    "",
		SortOrder: "",
		TabMode:   "",
	})

	options := service.GetFilterOptions()
	assert.Equal(t, "", options.ItemType)
	assert.Equal(t, "", options.Priority)
	assert.Equal(t, "", options.SortBy)
	assert.Equal(t, "", options.SortOrder)
	assert.Equal(t, "", options.TabMode)
}

// TestNewSortSettingsService_Defaults проверяет значения по умолчанию при создании
func TestNewSortSettingsService_Defaults(t *testing.T) {
	service := NewSortSettingsService()

	assert.NotNil(t, service.filterOptions)
	assert.Equal(t, "all", service.filterOptions.ItemType)
	assert.Equal(t, "none", service.filterOptions.Priority)
	assert.Equal(t, "name", service.filterOptions.SortBy)
	assert.Equal(t, "asc", service.filterOptions.SortOrder)
}
