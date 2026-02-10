package profile

import (
	"image/color"
	"projectT/internal/storage/database/queries"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// fieldRow представляет собой строку с пользовательским полем
type fieldRow struct {
	titleLabel *widget.Label
	valueLabel *widget.Label
	valueEntry *widget.Entry
	container  *fyne.Container
}

type UI struct {
	content          fyne.CanvasObject
	editMode         bool
	userNameLabel    *widget.Label
	userStatusLabel  *widget.Label
	userNameEntry    *widget.Entry
	userStatusEntry  *widget.Entry
	avatarImage      *canvas.Image
	avatarContainer  *fyne.Container
	backgroundImage  *canvas.Image
	backgroundRect   *canvas.Rectangle
	customFields     []*fieldRow
	editButton       *widget.Button
	backgroundButton *widget.Button
	applyButton      *widget.Button
	cancelButton     *widget.Button
	avatarPath       string
	backgroundPath   string
	window           fyne.Window
}

func New() *UI {
	ui := &UI{}

	// Загружаем профиль из базы данных
	profile, err := queries.GetProfile()
	if err == nil {
		// Устанавливаем пути из базы данных
		ui.avatarPath = profile.AvatarPath
		ui.backgroundPath = profile.BackgroundPath
	}

	ui.createView()
	return ui
}

func (p *UI) createView() {
	// Создаем основные компоненты
	p.createComponents()

	// Создаем верхнюю часть интерфейса
	topPart := p.createTopPart()

	// Создаем нижнюю часть интерфейса
	bottomPart := p.createBottomPart()

	// Создаем горизонтальный разделитель (1 пиксель высотой) между верхней и нижней частями
	horizontalSeparator := canvas.NewRectangle(color.Gray{Y: 128}) // Серый цвет
	horizontalSeparator.SetMinSize(fyne.NewSize(100, 1))

	// Компоновка верхней и нижней частей с разделителем - вертикальный бокс для размещения элементов друг под другом
	mainLayout := container.NewVBox(topPart, horizontalSeparator, bottomPart)

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

	// Правая часть верхней секции (пока пустая)
	rightTopPart := container.NewVBox(
		widget.NewLabel("Правая часть верхней секции"),
	)

	// Вертикальный разделитель (1 пиксель шириной)
	verticalSeparator := canvas.NewRectangle(color.Gray{Y: 128}) // Серый цвет
	verticalSeparator.SetMinSize(fyne.NewSize(1, 100))

	// Компоновка верхней части
	topPart := container.NewHBox(leftTopPart, verticalSeparator, rightTopPart)

	return topPart
}

func (p *UI) createBottomPart() fyne.CanvasObject {
	// Одна цельная нижняя часть (без деления на левую и правую)
	bottomPart := container.NewVBox(
		widget.NewLabel("Нижняя часть интерфейса"),
		widget.NewLabel("Здесь будет основной контент нижней части"),
	)

	return bottomPart
}
