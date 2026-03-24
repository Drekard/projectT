// Package right содержит компоненты правой панели (профиль)
package right

import (
	"encoding/json"
	"fmt"
	"log"

	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/cards/concrete"
	"projectT/internal/ui/cards/hover_preview"

	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// DemoElementItem представляет элемент витрины (demo element)
type DemoElementItem struct {
	ID          *int    `json:"id,omitempty"`           // Старый формат (числовой ID)
	ElementUUID *string `json:"element_uuid,omitempty"` // Новый формат (UUID)
	Title       string  `json:"title"`
	Value       string  `json:"value"`
}

// GetElementUUID возвращает ElementUUID элемента
func (dei *DemoElementItem) GetElementUUID() string {
	if dei.ElementUUID != nil && *dei.ElementUUID != "" {
		return *dei.ElementUUID
	}
	// Если UUID нет, пробуем найти элемент по старому ID
	if dei.ID != nil && *dei.ID > 0 {
		// Загружаем элемент по ID и получаем его UUID
		item, err := queries.GetItemByID(*dei.ID)
		if err == nil && item != nil {
			return item.ElementUUID
		}
	}
	return ""
}

// parseDemoElements парсит JSON строку с pinned UUIDs
// Поддерживает два формата:
// 1. Новый формат (pinned_uuids): ["uuid1", "uuid2", "uuid3"]
// 2. Старый формат с ID: [{"id": 1, "title": "...", "value": "..."}]
// 3. Очень старый формат: [24, 25, 26] (просто массив ID)
func parseDemoElements(jsonStr string) ([]DemoElementItem, error) {
	var result []DemoElementItem

	// Сначала пробуем распарсить как массив строк (новый формат pinned_uuids)
	var uuids []string
	if err := json.Unmarshal([]byte(jsonStr), &uuids); err == nil {
		// Это массив UUID
		for _, uuid := range uuids {
			result = append(result, DemoElementItem{
				ElementUUID: &uuid,
				Title:       "Элемент",
				Value:       "",
			})
		}
		return result, nil
	}

	// Если не получилось, пробуем распарсить как массив объектов
	var objects []DemoElementItem
	if err := json.Unmarshal([]byte(jsonStr), &objects); err == nil {
		return objects, nil
	}

	// Если не получилось, пробуем распарсить как массив чисел (очень старый формат)
	var ids []int
	if err := json.Unmarshal([]byte(jsonStr), &ids); err == nil {
		// Конвертируем ID в DemoElementItem
		for _, id := range ids {
			result = append(result, DemoElementItem{
				ID:    &id,
				Title: fmt.Sprintf("Элемент #%d", id),
				Value: "",
			})
		}
		return result, nil
	}

	// Если всё ещё не получилось, пробуем распарсить как одно число
	var singleID int
	if err := json.Unmarshal([]byte(jsonStr), &singleID); err == nil {
		result = append(result, DemoElementItem{
			ID:    &singleID,
			Title: fmt.Sprintf("Элемент #%d", singleID),
			Value: "",
		})
		return result, nil
	}

	// Если ничего не помогло, возвращаем ошибку
	return nil, fmt.Errorf("неизвестный формат JSON")
}

// loadDemoElements загружает и отображает элементы из demo_elements
func (p *Panel) loadDemoElements(jsonStr string) {
	if p.demoElementsContainer == nil {
		return
	}

	p.demoElementsContainer.Objects = nil

	// Логируем сырой JSON для отладки
	fmt.Printf("[DEBUG] loadDemoElements: получен JSON длиной %d символов\n", len(jsonStr))
	if len(jsonStr) > 0 && len(jsonStr) < 2000 {
		fmt.Printf("[DEBUG] loadDemoElements: JSON=%s\n", jsonStr)
	} else if len(jsonStr) >= 2000 {
		fmt.Printf("[DEBUG] loadDemoElements: JSON (первые 2000 символов)=%s...\n", jsonStr[:2000])
	}

	// Используем новую функцию парсинга с поддержкой старых форматов
	demoElements, err := parseDemoElements(jsonStr)
	if err != nil {
		log.Printf("[ERROR] Ошибка парсинга demo элементов: %v", err)
		log.Printf("[ERROR] Исходный JSON: %s", jsonStr)
		return
	}

	if len(demoElements) == 0 {
		emptyLabel := widget.NewLabel("Нет элементов витрины")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		p.demoElementsContainer.Add(emptyLabel)
		fmt.Println("[DEBUG] Demo elements: элементов не найдено")
	} else {
		fmt.Printf("[DEBUG] Demo elements: загружено %d элементов\n", len(demoElements))
		for i, elem := range demoElements {
			elementUUID := elem.GetElementUUID()
			fmt.Printf("[DEBUG] Элемент #%d: ElementUUID=%s, Title=%s, Value=%s\n", i+1, elementUUID, elem.Title, elem.Value)
			if elementUUID != "" {
				elementCard := p.createDemoElementCard(elementUUID)
				p.demoElementsContainer.Add(elementCard)
				fmt.Printf("[DEBUG] Элемент #%d успешно отображён\n", i+1)
			} else {
				fmt.Printf("[DEBUG] Элемент #%d пропущен (пустой ElementUUID) или не найден в БД\n", i+1)
			}
		}
	}

	p.demoElementsContainer.Refresh()
}

// createDemoElementCard создает карточку demo элемента (аналогично chat_panel.go)
func (p *Panel) createDemoElementCard(elementUUID string) fyne.CanvasObject {
	if elementUUID == "" {
		// Если UUID пустой, показываем ошибку
		return p.createDemoElementError("Неверный формат элемента")
	}

	// Загружаем элемент из базы данных по element_uuid
	item, err := queries.GetItemByElementUUID(elementUUID)
	if err != nil {
		// Если элемент не найден, показываем сообщение об ошибке
		return p.createDemoElementError("Элемент не найден")
	}

	// Создаём полноценную карточку элемента используя функционал concrete
	var cardRenderer fyne.CanvasObject
	switch item.Type {
	case "folder":
		cardRenderer = concrete.NewFolderCard(item).GetContainer()
	case "element":
		// Для элементов используем композитную карточку
		cardRenderer = concrete.NewCompositeCard(item).GetContainer()
	default:
		// Для неизвестных типов используем композитную карточку
		cardRenderer = concrete.NewCompositeCard(item).GetContainer()
	}

	// Оборачиваем в кликабельный виджет для обработки правого клика и превью
	clickableCard := hover_preview.NewClickableCard(cardRenderer, func() {
		// Показываем меню при правом клике
		menuManager := hover_preview.NewMenuManager()
		menuManager.ShowSimpleMenu(item, cardRenderer, nil)
	})

	return clickableCard
}

// createDemoElementError создает карточку с сообщением об ошибке
func (p *Panel) createDemoElementError(errorMsg string) fyne.CanvasObject {
	msgLabel := widget.NewLabel(errorMsg)
	msgLabel.Wrapping = fyne.TextWrapBreak

	// Создаем контейнер с фоном
	bgColor := color.RGBA{R: 200, G: 50, B: 50, A: 200} // Красный для ошибок
	bg := canvas.NewRectangle(bgColor)
	bg.CornerRadius = 10
	bg.SetMinSize(fyne.NewSize(200, 50))

	messageContainer := container.NewStack(bg, container.NewPadded(msgLabel))

	return messageContainer
}
