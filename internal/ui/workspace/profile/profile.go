package profile

import (
	"encoding/json"
	"fmt"
	"image/color"
	"projectT/internal/services"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/workspace/saved"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ContentCharacteristicItem represents a single characteristic item with title and value
type ContentCharacteristicItem struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Value string `json:"value"`
}

// fieldRow представляет собой строку с пользовательским полем
type fieldRow struct {
	id           int
	titleLabel   *widget.Label
	titleEntry   *widget.Entry
	valueEntry   *widget.Entry
	removeButton *widget.Button
	container    *fyne.Container
	timer        *time.Timer
}

type UI struct {
	content                  fyne.CanvasObject
	editMode                 bool
	userNameLabel            *widget.Label
	userStatusLabel          *widget.Label
	userNameEntry            *widget.Entry
	userStatusEntry          *widget.Entry
	avatarImage              *canvas.Image
	avatarContainer          *fyne.Container
	backgroundImage          *canvas.Image
	backgroundRect           *canvas.Rectangle
	customFields             []*fieldRow
	characteristicsContainer *fyne.Container
	characteristicsScroll    *container.Scroll
	editButton               *widget.Button
	backgroundButton         *widget.Button
	applyButton              *widget.Button
	cancelButton             *widget.Button
	addCharacteristicButton  *widget.Button
	loadCharacteristicsJSON  string
	nextID                   int
	avatarPath               string
	backgroundPath           string
	window                   fyne.Window
	gridManager              *saved.GridManager
}

func New() *UI {
	ui := &UI{}

	// Загружаем профиль из базы данных
	profile, err := queries.GetProfile()
	if err == nil {
		// Устанавливаем пути из базы данных
		ui.avatarPath = profile.AvatarPath
		ui.backgroundPath = profile.BackgroundPath

		// Сохраняем JSON характеристик для последующей загрузки
		ui.loadCharacteristicsJSON = profile.ContentCharacteristic
		fmt.Printf("DEBUG: Загружен профиль из базы данных. ContentCharacteristic: %s\n", profile.ContentCharacteristic)
	} else {
		fmt.Printf("DEBUG: Ошибка загрузки профиля из базы данных: %v\n", err)
	}

	// Инициализируем gridManager до создания представления
	ui.gridManager = saved.NewGridManager()
	ui.loadDemoElements()

	ui.createView()

	// После создания компонентов загружаем характеристики
	fmt.Println("DEBUG: Начинаем загрузку характеристик из JSON")
	ui.LoadCharacteristicsFromJSON(ui.loadCharacteristicsJSON)

	// Устанавливаем nextID на основе максимального ID из загруженных характеристик
	maxID := 0
	var characteristics []ContentCharacteristicItem
	if ui.loadCharacteristicsJSON != "" {
		err := json.Unmarshal([]byte(ui.loadCharacteristicsJSON), &characteristics)
		if err == nil {
			for _, item := range characteristics {
				if item.ID > maxID {
					maxID = item.ID
				}
			}
		}
	}
	ui.nextID = maxID + 1

	return ui
}

func (p *UI) createView() {
	// Создаем основные компоненты
	p.createComponents()

	// Создаем верхнюю часть интерфейса
	topPart := p.createTopPart()

	// Создаем нижнюю часть интерфейса
	bottomPart := p.createBottomPart()

	// Компоновка верхней и нижней частей с разделителем - вертикальный бокс для размещения элементов друг под другом
	mainLayout := container.NewVBox(topPart, bottomPart)

	p.content = mainLayout
}

func (p *UI) createComponents() {
	// Создание компонентов для профиля

	// Аватар
	var avatarImagePath string
	if p.avatarPath != "" {
		avatarImagePath = p.avatarPath
	} else {
		avatarImagePath = "assets/icons/icon.png" // временный путь
	}

	p.avatarImage = canvas.NewImageFromFile(avatarImagePath)
	p.avatarImage.FillMode = canvas.ImageFillContain
	p.avatarImage.SetMinSize(fyne.NewSize(100, 100))

	// Контейнер для аватара
	p.avatarContainer = container.NewCenter(p.avatarImage)

	// Информация о пользователе
	p.userNameLabel = widget.NewLabel("Имя пользователя")
	p.userStatusLabel = widget.NewLabel("Статус")

	p.userNameEntry = widget.NewEntry()
	p.userNameEntry.SetText(p.userNameLabel.Text)
	p.userNameEntry.Hide()

	p.userStatusEntry = widget.NewEntry()
	p.userStatusEntry.SetText(p.userStatusLabel.Text)
	p.userStatusEntry.Hide()

	// Кнопки
	p.editButton = widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
		p.toggleEditMode()
	})

	p.backgroundButton = widget.NewButton("Фон", func() {
		p.showBackgroundDialog()
	})

	p.applyButton = widget.NewButtonWithIcon("Применить", theme.ConfirmIcon(), func() {
		p.applyChanges()
	})
	p.applyButton.Hide()

	p.cancelButton = widget.NewButtonWithIcon("Отмена", theme.CancelIcon(), func() {
		p.cancelChanges()
	})
	p.cancelButton.Hide()
}

func (p *UI) createTopPart() fyne.CanvasObject {
	leftTopPartWrapper := canvas.NewRectangle(color.Transparent)
	leftTopPartWrapper.SetMinSize(fyne.NewSize(300, 0))

	// Левая часть верхней секции (аватар, имя, статус, кнопки)
	leftTopPart := container.NewVBox(
		leftTopPartWrapper,
		container.NewCenter(p.avatarContainer),
		container.NewCenter(
			p.userNameLabel,
			p.userNameEntry,
		),
		container.NewCenter(
			p.userStatusLabel,
			p.userStatusEntry,
		),
		container.NewHBox(
			p.editButton,
			p.backgroundButton,
			p.applyButton,
			p.cancelButton,
		),
	)

	// Правая часть верхней секции - прокручиваемый список характеристик
	p.characteristicsContainer = container.NewVBox()

	// Создаем прокручиваемый контейнер
	p.characteristicsScroll = container.NewScroll(p.characteristicsContainer)
	p.characteristicsScroll.SetMinSize(fyne.NewSize(400, 200)) // Устанавливаем минимальный размер

	// Кнопка добавления новой характеристики
	p.addCharacteristicButton = widget.NewButton("+ Добавить характеристику", func() {
		p.AddCharacteristic()
	})
	p.addCharacteristicButton.Importance = widget.LowImportance

	// Правая часть верхней секции
	rightTopPart := container.NewVBox(
		widget.NewLabel("Характеристики профиля"),
		p.characteristicsScroll,
		p.addCharacteristicButton,
	)

	// Вертикальный разделитель (1 пиксель шириной)
	verticalSeparator := canvas.NewRectangle(color.Gray{Y: 128}) // Серый цвет
	verticalSeparator.SetMinSize(fyne.NewSize(1, 100))

	// Компоновка верхней части
	topPart := container.NewHBox(leftTopPart, verticalSeparator, rightTopPart)

	return topPart
}

func (p *UI) createBottomPart() fyne.CanvasObject {

	horizontalSeparator := canvas.NewRectangle(color.Gray{Y: 128})
	horizontalSeparator.SetMinSize(fyne.NewSize(100, 2))
	aSeparator := canvas.NewRectangle(color.Transparent)
	aSeparator.SetMinSize(fyne.NewSize(1, 13))
	separatorContainer := container.NewVBox(
		aSeparator,
		horizontalSeparator,
	)

	// Используем GridManager для отображения закрепленных элементов
	pinnedGridManager := saved.NewGridManager()

	// Загружаем закрепленные элементы
	p.updatePinnedItems(pinnedGridManager)

	pinnedGridContainer := pinnedGridManager.GetContainer()
	pinnedGridContainer.SetMinSize(fyne.NewSize(400, 400))

	// Подписываемся на события изменения закрепленных элементов
	eventChan := services.GetPinnedEventManager().Subscribe()
	go func() {
		for eventType := range eventChan {
			if eventType == "pinned_items_changed" {
				// Обновляем закрепленные элементы
				p.updatePinnedItems(pinnedGridManager)
			}
		}
	}()

	bottomPart := container.NewVBox(
		container.NewBorder(nil, nil, widget.NewLabel("Витрина"), nil, separatorContainer),
		pinnedGridContainer,
	)

	return bottomPart
}

// updatePinnedItems обновляет отображение закрепленных элементов
func (p *UI) updatePinnedItems(gridManager *saved.GridManager) {
	// Загружаем закрепленные элементы
	pinnedItems, err := queries.GetPinnedItems()
	if err != nil {
		fmt.Printf("DEBUG: Ошибка загрузки закрепленных элементов: %v\n", err)
		pinnedItems = []*models.Item{} // Инициализируем пустым списком в случае ошибки
	}

	// Обновляем элементы в GridManager
	gridManager.LoadItemsWithoutCreateElement(pinnedItems)
}

// loadDemoElements загружает элементы из поля DemoElements профиля
func (p *UI) loadDemoElements() {
	// Загружаем профиль для получения DemoElements
	profile, err := queries.GetProfile()
	if err != nil {
		fmt.Printf("DEBUG: Ошибка загрузки профиля для DemoElements: %v\n", err)
		return
	}

	// Парсим JSON-массив ID элементов
	var elementIDs []int
	if profile.DemoElements != "" {
		err := json.Unmarshal([]byte(profile.DemoElements), &elementIDs)
		if err != nil {
			fmt.Printf("DEBUG: Ошибка парсинга DemoElements JSON: %v\n", err)
			return
		}
	}

	// Загружаем элементы по ID и передаем их в GridManager
	var items []*models.Item
	for _, id := range elementIDs {
		item, err := queries.GetItemByID(id)
		if err != nil {
			fmt.Printf("DEBUG: Ошибка получения элемента по ID %d: %v\n", id, err)
			continue
		}
		items = append(items, item)
	}

	// Загружаем элементы в GridManager
	p.gridManager.LoadItemsWithoutCreateElement(items)
	fmt.Printf("DEBUG: Загружено %d элементов в сетку DemoElements\n", len(items))
}

// addElementToDemoElements добавляет элемент в DemoElements профиля
func (p *UI) addElementToDemoElements(elementID int) {
	// Получаем текущий профиль
	profile, err := queries.GetProfile()
	if err != nil {
		fmt.Printf("DEBUG: Ошибка получения профиля: %v\n", err)
		return
	}

	// Парсим текущие ID элементов
	var currentElementIDs []int
	if profile.DemoElements != "" {
		err := json.Unmarshal([]byte(profile.DemoElements), &currentElementIDs)
		if err != nil {
			fmt.Printf("DEBUG: Ошибка парсинга текущих DemoElements: %v\n", err)
			return
		}
	}

	// Проверяем, не добавлен ли уже элемент
	for _, id := range currentElementIDs {
		if id == elementID {
			fmt.Printf("DEBUG: Элемент с ID %d уже добавлен в DemoElements\n", elementID)
			return
		}
	}

	// Добавляем новый ID
	currentElementIDs = append(currentElementIDs, elementID)

	// Сохраняем обновленный список обратно в профиль
	updatedJSON, err := json.Marshal(currentElementIDs)
	if err != nil {
		fmt.Printf("DEBUG: Ошибка маршалинга DemoElements: %v\n", err)
		return
	}

	profile.DemoElements = string(updatedJSON)

	// Обновляем профиль в базе данных
	err = queries.UpdateProfileField("demo_elements", profile.DemoElements, profile.ID)
	if err != nil {
		fmt.Printf("DEBUG: Ошибка обновления DemoElements в базе данных: %v\n", err)
		return
	}

	fmt.Printf("DEBUG: Элемент с ID %d добавлен в DemoElements\n", elementID)

	// Перезагружаем элементы в сетке
	p.loadDemoElements()
}
