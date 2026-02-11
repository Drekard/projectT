package profile

import (
	"encoding/json"
	"fmt"
	"image/color"
	"projectT/internal/storage/database/queries"
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
	p.editButton = widget.NewButtonWithIcon("Редактировать", theme.DocumentCreateIcon(), func() {
		p.toggleEditMode()
	})

	p.backgroundButton = widget.NewButton("Фон", func() {
		// Обработка нажатия на кнопку "Фон"
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
	// Левая часть верхней секции (аватар, имя, статус, кнопки)
	leftTopPart := container.NewVBox(
		container.NewCenter(p.avatarContainer),
		container.NewHBox(
			widget.NewLabel("Имя:"),
			p.userNameLabel,
			p.userNameEntry,
		),
		container.NewHBox(
			widget.NewLabel("Статус:"),
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
	horizontalSeparator.SetMinSize(fyne.NewSize(100, 1))
	aSeparator := canvas.NewRectangle(color.Transparent)
	aSeparator.SetMinSize(fyne.NewSize(1, 12))

	// Создаем контейнер с фиксированной высотой для разделителя
	separatorContainer := container.NewVBox(
		aSeparator,
		horizontalSeparator,
	)

	addDemoElementsButton := widget.NewButton("+ Добавить элемент", func() {})
	addDemoElementsButton.Importance = widget.LowImportance

	borderPart := container.NewBorder(
		nil, nil,
		nil,
		addDemoElementsButton,
		separatorContainer, // Используем контейнер вместо прямого разделителя
	)

	bottomPart := container.NewVBox(
		borderPart,
		widget.NewLabel("Нижняя часть интерфейса"),
		widget.NewLabel("Тут нужно реализовать сетку элементов какая уже есть в internal/ui/workspace/saved, но отображать только элементы чей ID есть в profile DemoElements"),
	)
	addDemoElementsButton.Move(fyne.NewPos(0, 50))

	return bottomPart
}
