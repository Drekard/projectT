package chats

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"os"

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
// Если UUID есть, возвращает его, иначе пытается найти по ID
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

// ContentCharacteristicItem представляет элемент характеристики
type ContentCharacteristicItem struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Value string `json:"value"`
}

// createProfileArea создает правую панель с профилем собеседника
func (ui *UI) createProfileArea() *fyne.Container {
	// Аватар - изображение 100x100
	ui.profileAvatar = canvas.NewImageFromResource(nil)
	ui.profileAvatar.FillMode = canvas.ImageFillContain

	// Черный фон 100x100 под аватарку
	avatarBg := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 255})
	avatarBg.SetMinSize(fyne.NewSize(100, 100))

	// Аватарка поверх фона через Stack
	avatarStack := container.NewStack(avatarBg, ui.profileAvatar)

	// Имя
	ui.profileName = widget.NewLabel("")
	ui.profileName.TextStyle = fyne.TextStyle{Bold: true}
	ui.profileName.Alignment = fyne.TextAlignCenter

	// Текстовый статус пользователя (устанавливается вручную)
	ui.profileStatus = widget.NewLabel("")
	ui.profileStatus.TextStyle = fyne.TextStyle{Italic: true}
	ui.profileStatus.Alignment = fyne.TextAlignCenter

	// Отступы сверху и снизу
	spacerTop := canvas.NewRectangle(color.Transparent)
	spacerTop.SetMinSize(fyne.NewSize(0, 20))
	spacerBottom := canvas.NewRectangle(color.Transparent)
	spacerBottom.SetMinSize(fyne.NewSize(0, 20))

	// Контейнер для аватара и имени
	headerContainer := container.NewVBox(
		//spacerTop,
		container.NewCenter(avatarStack),
		//spacerBottom,
		ui.profileName,
		ui.profileStatus,
		//layout.NewSpacer(),
	)

	// Разделитель
	separator1 := canvas.NewRectangle(color.RGBA{R: 64, G: 64, B: 64, A: 255})
	separator1.SetMinSize(fyne.NewSize(200, 1))

	// Заголовок характеристик
	characteristicsTitle := widget.NewLabel("Характеристики")
	characteristicsTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Контейнер для характеристик
	ui.characteristicsContainer = container.NewVBox()

	// Заголовок витрины элементов
	demoElementsTitle := widget.NewLabel("Витрина элементов")
	demoElementsTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Контейнер для demo элементов
	ui.demoElementsContainer = container.NewVBox()

	// Основная информация (без внутренних прокруток)
	infoContainer := container.NewVBox(
		headerContainer,
		separator1,
		container.NewPadded(container.NewVBox(characteristicsTitle, ui.characteristicsContainer)),
		separator1,
		container.NewPadded(container.NewVBox(demoElementsTitle, ui.demoElementsContainer)),
	)

	// Оборачиваем всю панель в прокрутку
	scrollContainer := container.NewScroll(infoContainer)

	// Фон
	bg := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 255})
	bg.SetMinSize(fyne.NewSize(270, 1))

	ui.profileArea = container.NewStack(bg, scrollContainer)

	// Загружаем профиль текущего пользователя при инициализации
	ui.showUserProfile()

	return ui.profileArea
}

// showUserProfile показывает профиль текущего пользователя в правой панели
func (ui *UI) showUserProfile() {
	// Загружаем локальный профиль
	localProfile, err := queries.GetLocalProfile()
	if err != nil {
		log.Printf("Ошибка загрузки локального профиля: %v", err)
		return
	}

	// Создаём временный контакт с данными профиля
	tempContact := &models.Contact{
		Username:   localProfile.Username,
		Title:      localProfile.Title, // Титул из профиля
		AvatarPath: localProfile.AvatarPath,
		PeerID:     localProfile.PeerID,
	}

	// Обновляем правую панель с профилем пользователя
	ui.updateProfile(tempContact)

	// Загружаем характеристики из профиля
	if localProfile.ContentChar != "" && ui.characteristicsContainer != nil {
		ui.loadCharacteristics(localProfile.ContentChar)
	}

	// Загружаем избранные элементы из pinned_uuids
	if localProfile.PinnedUUIDs != "" && ui.demoElementsContainer != nil {
		ui.loadDemoElements(localProfile.PinnedUUIDs)
	}
}

// updateProfile обновляет профиль собеседника
func (ui *UI) updateProfile(contact *models.Contact) {
	// Проверяем, локальный ли это чат
	if contact.IsLocalChat() {
		// Для локального чата показываем профиль текущего пользователя
		ui.showUserProfile()
		return
	}

	// Обновляем имя
	if ui.profileName != nil {
		ui.profileName.SetText(contact.Username)
	}

	// Обновляем статус (текстовый, из профиля)
	if ui.profileStatus != nil {
		ui.profileStatus.SetText(contact.Title)
	}

	// Загружаем аватар
	if ui.profileAvatar != nil {
		ui.loadAvatar(contact.AvatarPath)
	}

	// Загружаем характеристики из профиля пира
	if contact.PeerID != "" && ui.characteristicsContainer != nil {
		// Загружаем профиль из таблицы profiles по PeerID
		profile, err := queries.GetProfileByPeerID(contact.PeerID)
		if err == nil && profile != nil {
			if profile.ContentChar != "" {
				ui.loadCharacteristics(profile.ContentChar)
			} else {
				// Если характеристик нет, очищаем контейнер
				ui.characteristicsContainer.Objects = nil
				ui.characteristicsContainer.Refresh()
			}
		}
	}

	// Загружаем demo элементы из профиля пира
	if contact.PeerID != "" && ui.demoElementsContainer != nil {
		// Загружаем профиль из таблицы profiles по PeerID
		profile, err := queries.GetProfileByPeerID(contact.PeerID)
		if err == nil && profile != nil {
			if profile.PinnedUUIDs != "" {
				ui.loadDemoElements(profile.PinnedUUIDs)
			} else {
				// Если demo элементов нет, очищаем контейнер
				ui.demoElementsContainer.Objects = nil
				ui.demoElementsContainer.Refresh()
			}
		}
	}

	// Обновляем UI
	if ui.profileArea != nil {
		ui.profileArea.Refresh()
	}
}

// loadAvatar загружает аватар из локального хранилища
func (ui *UI) loadAvatar(avatarPath string) {
	if ui.profileAvatar == nil {
		return
	}

	if avatarPath == "" {
		// Пустой аватар - скрываем изображение
		ui.profileAvatar.Resource = nil
		ui.profileAvatar.Refresh()
		return
	}

	// Проверяем существование файла
	if _, err := os.Stat(avatarPath); os.IsNotExist(err) {
		log.Printf("Аватар не найден: %s", avatarPath)
		ui.profileAvatar.Resource = nil
		ui.profileAvatar.Refresh()
		return
	}

	// Загружаем изображение
	avatarImg, err := fyne.LoadResourceFromPath(avatarPath)
	if err != nil {
		log.Printf("Ошибка загрузки аватара: %v", err)
		ui.profileAvatar.Resource = nil
		ui.profileAvatar.Refresh()
		return
	}

	// Устанавливаем изображение
	ui.profileAvatar.Resource = avatarImg
	ui.profileAvatar.FillMode = canvas.ImageFillContain
	ui.profileAvatar.Refresh()
}

// loadCharacteristics загружает характеристики из JSON
func (ui *UI) loadCharacteristics(jsonStr string) {
	if ui.characteristicsContainer == nil {
		return
	}

	ui.characteristicsContainer.Objects = nil

	var characteristics []ContentCharacteristicItem
	if jsonStr != "" {
		err := json.Unmarshal([]byte(jsonStr), &characteristics)
		if err != nil {
			log.Printf("Ошибка парсинга характеристик: %v", err)
			return
		}
	}

	if len(characteristics) == 0 {
		emptyLabel := widget.NewLabel("Нет характеристик")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.characteristicsContainer.Add(emptyLabel)
	} else {
		for _, item := range characteristics {
			characteristicItem := ui.createCharacteristicItem(item.Title, item.Value)
			ui.characteristicsContainer.Add(characteristicItem)
		}
	}

	ui.characteristicsContainer.Refresh()
}

// createCharacteristicItem создает элемент характеристики (название: значение в одну строку)
func (ui *UI) createCharacteristicItem(title, value string) *fyne.Container {
	// Форматируем как "Название: Значение"
	text := fmt.Sprintf("%s: %s", title, value)
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	return container.NewVBox(label)
}

// createDemoElementCard создает карточку demo элемента (аналогично chat_panel.go)
func (ui *UI) createDemoElementCard(elementUUID string) fyne.CanvasObject {
	if elementUUID == "" {
		// Если UUID пустой, показываем ошибку
		return ui.createDemoElementError("Неверный формат элемента")
	}

	// Загружаем элемент из базы данных по element_uuid
	item, err := queries.GetItemByElementUUID(elementUUID)
	if err != nil {
		// Если элемент не найден, показываем сообщение об ошибке
		return ui.createDemoElementError("Элемент не найден")
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
func (ui *UI) createDemoElementError(errorMsg string) fyne.CanvasObject {
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

// loadDemoElements загружает и отображает элементы из demo_elements
func (ui *UI) loadDemoElements(jsonStr string) {
	if ui.demoElementsContainer == nil {
		return
	}

	ui.demoElementsContainer.Objects = nil

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
		ui.demoElementsContainer.Add(emptyLabel)
		fmt.Println("[DEBUG] Demo elements: элементов не найдено")
	} else {
		fmt.Printf("[DEBUG] Demo elements: загружено %d элементов\n", len(demoElements))
		for i, elem := range demoElements {
			elementUUID := elem.GetElementUUID()
			fmt.Printf("[DEBUG] Элемент #%d: ElementUUID=%s, Title=%s, Value=%s\n", i+1, elementUUID, elem.Title, elem.Value)
			if elementUUID != "" {
				elementCard := ui.createDemoElementCard(elementUUID)
				ui.demoElementsContainer.Add(elementCard)
				fmt.Printf("[DEBUG] Элемент #%d успешно отображён\n", i+1)
			} else {
				fmt.Printf("[DEBUG] Элемент #%d пропущен (пустой ElementUUID) или не найден в БД\n", i+1)
			}
		}
	}

	ui.demoElementsContainer.Refresh()
}
