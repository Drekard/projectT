package profile

import (
	"encoding/json"
	"image/color"
	"projectT/internal/services/pinned"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/workspace/saved"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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
	userNameLabel            *widget.Label
	userStatusLabel          *widget.Label
	userNameEntry            *widget.Entry
	userStatusEntry          *widget.Entry
	avatarImage              *canvas.Image
	avatarContainer          *fyne.Container
	customFields             []*fieldRow
	characteristicsContainer *fyne.Container
	characteristicsScroll    *container.Scroll
	backgroundButton         *widget.Button
	avatarButton             *widget.Button
	addCharacteristicButton  *widget.Button
	loadCharacteristicsJSON  string
	nextID                   int
	avatarPath               string
	backgroundPath           string
	window                   fyne.Window
	gridManager              *saved.GridManager
	userNameTimer            *time.Timer
	userStatusTimer          *time.Timer
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
	} else {
		_ = err //nolint:staticcheck // Игнорируем ошибку загрузки профиля
	}

	// Инициализируем gridManager до создания представления
	ui.gridManager = saved.NewGridManager()

	ui.createView()

	// После создания компонентов загружаем характеристики
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

	// Оборачиваем изображение в кликабельный виджет
	avatarClickable := NewAvatarClickableImage(p.avatarImage, nil)

	// Контейнер для аватара
	p.avatarContainer = container.NewCenter(avatarClickable)

	// Информация о пользователе
	p.userNameLabel = widget.NewLabel("Имя пользователя")
	p.userStatusLabel = widget.NewLabel("Статус")

	// Загружаем данные из профиля
	profile, err := queries.GetProfile()
	if err == nil {
		p.userNameEntry = widget.NewEntry()
		p.userNameEntry.SetText(profile.Username)
		p.userNameEntry.OnChanged = func(text string) {
			p.scheduleProfileAutoSave()
		}

		p.userStatusEntry = widget.NewEntry()
		p.userStatusEntry.SetText(profile.Status)
		p.userStatusEntry.OnChanged = func(text string) {
			p.scheduleProfileAutoSave()
		}
	} else {
		p.userNameEntry = widget.NewEntry()
		p.userNameEntry.SetText(p.userNameLabel.Text)
		p.userNameEntry.OnChanged = func(text string) {
			p.scheduleProfileAutoSave()
		}

		p.userStatusEntry = widget.NewEntry()
		p.userStatusEntry.SetText(p.userStatusLabel.Text)
		p.userStatusEntry.OnChanged = func(text string) {
			p.scheduleProfileAutoSave()
		}
	}

	p.backgroundButton = widget.NewButton("Фон", func() {
		p.showBackgroundDialog()
	})

	p.avatarButton = widget.NewButton("Аватар", func() {
		p.showAvatarDialog()
	})
}

func (p *UI) createTopPart() fyne.CanvasObject {
	leftTopPartWrapper := canvas.NewRectangle(color.Transparent)
	leftTopPartWrapper.SetMinSize(fyne.NewSize(300, 0))

	// Объединяем Label и Entry через Stack для имени
	entryNameWrapper := canvas.NewRectangle(color.Transparent)
	entryNameWrapper.SetMinSize(fyne.NewSize(200, 40))
	nameContainer := container.NewStack(entryNameWrapper, p.userNameEntry)

	// Объединяем Label и Entry через Stack для статуса
	entryStatusWrapper := canvas.NewRectangle(color.Transparent)
	entryStatusWrapper.SetMinSize(fyne.NewSize(200, 40))
	statusContainer := container.NewStack(entryStatusWrapper, p.userStatusEntry)

	// Левая часть верхней секции (аватар, имя, статус, кнопки)
	leftTopPart := container.NewVBox(
		leftTopPartWrapper,
		container.NewCenter(p.avatarContainer),
		container.NewCenter(nameContainer),
		container.NewCenter(statusContainer),
		container.NewCenter(container.NewHBox(p.backgroundButton, p.avatarButton)),
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
	eventChan := pinned.GetEventManager().Subscribe()
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
		pinnedItems = []*models.Item{} // Инициализируем пустым списком в случае ошибки
	}

	// Обновляем элементы в GridManager
	gridManager.LoadItemsWithoutCreateElement(pinnedItems)
}
