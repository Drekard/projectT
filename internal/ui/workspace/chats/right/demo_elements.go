// Package right содержит компоненты правой панели (профиль)
package right

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/cards/concrete"
	"projectT/internal/ui/cards/hover_preview"

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

	log.Printf("[DemoElements] 📦 Загрузка витрины элементов: pinned_uuids=%s", jsonStr)

	p.demoElementsContainer.Objects = nil

	// Используем новую функцию парсинга с поддержкой старых форматов
	demoElements, err := parseDemoElements(jsonStr)
	if err != nil {
		log.Printf("[DemoElements] ❌ Ошибка парсинга JSON: %v", err)
		return
	}

	log.Printf("[DemoElements] 📋 Распарсено %d элементов", len(demoElements))

	if len(demoElements) == 0 {
		emptyLabel := widget.NewLabel("Нет элементов витрины")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		p.demoElementsContainer.Add(emptyLabel)
		log.Printf("[DemoElements] ℹ️ Витрина пуста")
	} else {
		for i, elem := range demoElements {
			elementUUID := elem.GetElementUUID()
			if elementUUID != "" {
				log.Printf("[DemoElements] 🔍 Загрузка элемента #%d: UUID=%s, title=%q", i+1, elementUUID, elem.Title)
				elementCard := p.createDemoElementCard(elementUUID)
				p.demoElementsContainer.Add(elementCard)
			} else {
				log.Printf("[DemoElements] ⚠️ Элемент #%d имеет пустой UUID, пропускаем", i+1)
			}
		}
		log.Printf("[DemoElements] ✅ Витрина загружена: %d элементов отображено", len(demoElements))
	}

	p.demoElementsContainer.Refresh()
}

// createDemoElementCard создает карточку demo элемента (аналогично chat_panel.go)
func (p *Panel) createDemoElementCard(elementUUID string) fyne.CanvasObject {
	if elementUUID == "" {
		log.Printf("[DemoElements] ❌ createDemoElementCard: пустой UUID")
		// Если UUID пустой, показываем ошибку
		return p.createDemoElementError("Неверный формат элемента")
	}

	log.Printf("[DemoElements] 🔍 Загрузка элемента из БД: UUID=%s", elementUUID)

	// Загружаем элемент из базы данных по element_uuid
	item, err := queries.GetItemByElementUUID(elementUUID)
	if err != nil {
		log.Printf("[DemoElements] ❌ Элемент не найден в БД: UUID=%s, ошибка: %v", elementUUID, err)
		// Если элемент не найден, показываем сообщение об ошибке
		return p.createDemoElementError("Элемент не найден")
	}

	log.Printf("[DemoElements] ✅ Элемент найден: ID=%d, title=%q, type=%s, status=%s",
		item.ID, item.Title, item.Type, item.Status)

	// Создаём полноценную карточку элемента используя функционал concrete
	// Для профиля используем режим без кнопок
	var cardRenderer fyne.CanvasObject
	switch item.Type {
	case "folder":
		log.Printf("[DemoElements] 📁 Создание карточки папки")
		cardRenderer = concrete.NewFolderCard(item, true).GetContainer()
	case "element":
		log.Printf("[DemoElements] 📄 Создание карточки элемента")
		// Для элементов используем композитную карточку в режиме без кнопок
		cardRenderer = concrete.NewCompositeCard(item, true).GetContainer()
	default:
		log.Printf("[DemoElements] ❓ Неизвестный тип элемента: %s, используем композитную карточку", item.Type)
		// Для неизвестных типов используем композитную карточку в режиме без кнопок
		cardRenderer = concrete.NewCompositeCard(item, true).GetContainer()
	}

	// Оборачиваем в кликабельный виджет для обработки правого клика и превью
	clickableCard := hover_preview.NewClickableCard(cardRenderer, func() {
		log.Printf("[DemoElements] 🖱️ Правый клик на элементе: ID=%d, title=%q", item.ID, item.Title)
		// Показываем меню при правом клике в режиме без кнопок
		menuManager := hover_preview.NewMenuManager()
		menuManager.ShowSimpleMenu(item, cardRenderer, nil, true)
	})

	// Добавляем индикатор статуса элемента
	statusIndicator := p.createStatusIndicator(item)

	// Компоновка: карточка + индикатор статуса
	cardWithStatus := container.NewVBox(
		clickableCard,
		statusIndicator,
	)

	log.Printf("[DemoElements] ✅ Карточка элемента создана: ID=%d", item.ID)
	return cardWithStatus
}

// createStatusIndicator создаёт индикатор статуса элемента
func (p *Panel) createStatusIndicator(item *models.Item) fyne.CanvasObject {
	// Для saved и preview элементов не показываем индикатор
	if item.IsSaved() || item.IsPreview() {
		return container.NewHBox()
	}

	// Для archived элементов показываем индикатор
	var statusLabel *widget.Label

	if item.IsArchived() {
		statusLabel = widget.NewLabel("🗄️ Архив")
	} else {
		return container.NewHBox()
	}

	statusLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Компоновка индикатора
	indicator := container.NewHBox(statusLabel)

	return indicator
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
