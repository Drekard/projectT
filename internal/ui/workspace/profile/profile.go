package profile

import (
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
	ElementUUID string `json:"element_uuid"` // Глобальный UUID элемента для P2P
	Title       string `json:"title"`
	Value       string `json:"value"`
}

// fieldRow представляет собой строку с пользовательским полем
type fieldRow struct {
	elementUUID  string // ElementUUID элемента (для P2P)
	titleLabel   *widget.Label
	titleEntry   *widget.Entry
	valueEntry   *widget.Entry
	removeButton *widget.Button
	container    *fyne.Container
	timer        *time.Timer
}

type UI struct {
	content                  fyne.CanvasObject
	userNameEntry            *widget.Entry
	userTitleEntry           *widget.Entry
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
	userTitleTimer           *time.Timer
}

func New() *UI {
	ui := &UI{}

	// Загружаем профиль из базы данных
	profile, err := queries.GetLocalProfile()
	if err == nil {
		// Устанавливаем пути из базы данных
		ui.avatarPath = profile.AvatarPath
		ui.backgroundPath = profile.BackgroundPath

		// Сохраняем JSON характеристик для последующей загрузки
		ui.loadCharacteristicsJSON = profile.ContentChar
	} else {
		_ = err //nolint:staticcheck // Игнорируем ошибку загрузки профиля
	}

	// Инициализируем gridManager до создания представления
	ui.gridManager = saved.NewGridManager()

	ui.createView()

	// После создания компонентов загружаем характеристики
	ui.LoadCharacteristicsFromJSON(ui.loadCharacteristicsJSON)

	// nextID больше не нужен, так как используем ElementUUID вместо ID
	// Оставляем для обратной совместимости
	ui.nextID = 1

	return ui
}

func (p *UI) createView() {
	// Создаем основные компоненты
	p.createComponents()

	// Создаем левую панель (аватар, имя, титул, кнопки, характеристики)
	leftPanel := p.createLeftPanel()

	// Создаем правую панель (закрепленные элементы)
	rightPanel := p.createRightPanel()

	// Разделяем на левую и правую части с помощью SplitContainer
	split := container.NewHSplit(leftPanel, rightPanel)
	split.SetOffset(0.35) // Левая панель занимает 35% ширины

	p.content = split
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

	p.userNameEntry = widget.NewEntry()
	p.userNameEntry.SetPlaceHolder("Логин")
	p.userNameEntry.OnChanged = func(text string) {
		p.scheduleProfileAutoSave()
	}

	p.userTitleEntry = widget.NewEntry()
	p.userTitleEntry.SetPlaceHolder("Титул")
	p.userTitleEntry.OnChanged = func(text string) {
		p.scheduleProfileAutoSave()
	}

	// Загружаем данные из профиля
	profile, err := queries.GetLocalProfile()
	if err == nil {
		p.userNameEntry.SetText(profile.Username)
		p.userTitleEntry.SetText(profile.Title)
	}

	p.backgroundButton = widget.NewButton("Фон", func() {
		p.showBackgroundDialog()
	})

	p.avatarButton = widget.NewButton("Аватар", func() {
		p.showAvatarDialog()
	})
}

// createLeftPanel создает левую панель профиля (аватар, имя, титул, кнопки, характеристики)
func (p *UI) createLeftPanel() fyne.CanvasObject {
	// Прозрачные прямоугольники для фона полей ввода (ширина 400px)
	nameBg := canvas.NewRectangle(color.Transparent)
	nameBg.SetMinSize(fyne.NewSize(250, 40))
	titleBg := canvas.NewRectangle(color.Transparent)
	titleBg.SetMinSize(fyne.NewSize(250, 40))

	// Аватар, имя, титул, кнопки
	avatarSection := container.NewVBox(
		container.NewCenter(p.avatarContainer),
		container.NewStack(nameBg, p.userNameEntry),
		container.NewStack(titleBg, p.userTitleEntry),
		container.NewCenter(container.NewHBox(p.backgroundButton, p.avatarButton)),
	)

	// Характеристики
	p.characteristicsContainer = container.NewVBox()
	p.characteristicsScroll = container.NewScroll(p.characteristicsContainer)
	p.characteristicsScroll.SetMinSize(fyne.NewSize(0, 200))

	p.addCharacteristicButton = widget.NewButton("+ Добавить характеристику", func() {
		p.AddCharacteristic()
	})
	p.addCharacteristicButton.Importance = widget.LowImportance

	characteristicsSection := container.NewVBox(
		widget.NewLabel("Характеристики"),
		p.characteristicsScroll,
		p.addCharacteristicButton,
	)

	// Горизонтальный разделитель
	separator := canvas.NewRectangle(color.Gray{Y: 128})
	separator.SetMinSize(fyne.NewSize(0, 1))

	content := container.NewVBox(
		avatarSection,
		separator,
		characteristicsSection,
	)

	return container.NewScroll(content)
}

// createRightPanel создает правую панель профиля (закрепленные элементы)
func (p *UI) createRightPanel() fyne.CanvasObject {
	// Используем GridManager для отображения закрепленных элементов
	pinnedGridManager := saved.NewGridManager()

	// Устанавливаем 2 колонки для вкладки профиля
	pinnedGridManager.SetColumnCount(2)

	// Загружаем закрепленные элементы
	p.updatePinnedItems(pinnedGridManager)

	pinnedGridContainer := pinnedGridManager.GetContainer()

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

	// Используем Border для заполнения доступного пространства без горизонтальной прокрутки
	// Заголовок сверху, сетка занимает всё оставшееся место
	content := container.NewBorder(
		widget.NewLabel("Витрина"), // Top
		nil,                        // Bottom
		nil,                        // Left
		nil,                        // Right
		pinnedGridContainer,        // Content
	)

	return content
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
