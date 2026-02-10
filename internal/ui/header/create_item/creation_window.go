package create_item

import (
	"context"
	"image/color"
	"strings"

	"projectT/internal/services"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// SelectedFolder хранит информацию о выбранной папке
type SelectedFolder struct {
	ID   *int
	Name string
}

// ItemType определяет тип создаваемого элемента
type ItemType string

const (
	ItemTypeElement ItemType = "element"
	ItemTypeFolder  ItemType = "folder"
)

// Global переменная для хранения выбранного типа элемента
var selectedType ItemType = ItemTypeElement

// Global переменная для хранения выбранной папки
var selectedFolder *SelectedFolder = &SelectedFolder{ID: nil, Name: "Сохраненное"}

// setCurrentFolder устанавливает текущую выбранную папку
func setCurrentFolder(id *int, name string) {
	selectedFolder = &SelectedFolder{ID: id, Name: name}
}

// Интерфейс для взаимодействия с менеджером хлебных крошек
type BreadcrumbManagerInterface interface {
	GetCurrentFolderID() *int
	SetRefreshCallback(callback func())
	Refresh()
}

// WorkspaceRefresher интерфейс для обновления рабочей области
type WorkspaceRefresher interface {
	RefreshCurrentFolder() error
}

// NewRectangleManager менеджер компонента NewRectangle
type NewRectangleManager struct {
	popup              *widget.PopUp
	breadcrumbManager  BreadcrumbManagerInterface
	workspaceRefresher WorkspaceRefresher
}

// NewNewRectangleManager создает новый менеджер NewRectangle
func NewNewRectangleManager(breadcrumbManager BreadcrumbManagerInterface) *NewRectangleManager {
	return &NewRectangleManager{breadcrumbManager: breadcrumbManager}
}

// NewNewRectangleManagerWithWorkspace создает новый менеджер NewRectangle с возможностью обновления рабочей области
func NewNewRectangleManagerWithWorkspace(breadcrumbManager BreadcrumbManagerInterface, workspaceRefresher WorkspaceRefresher) *NewRectangleManager {
	return &NewRectangleManager{breadcrumbManager: breadcrumbManager, workspaceRefresher: workspaceRefresher}
}

// ShowNewRectangle показывает компонент NewRectangle под кнопкой [+] снизу по центру
func (nrm *NewRectangleManager) ShowNewRectangle(trigger fyne.CanvasObject, onClose func()) {
	window := fyne.CurrentApp().Driver().CanvasForObject(trigger)
	if window == nil {
		return
	}

	// Создаем основной контейнер для NewRectangle
	content := createNewRectangleContent(nrm.breadcrumbManager, onClose)

	nrm.popup = widget.NewPopUp(content, window)

	// Позиция триггера (кнопки [+])
	triggerPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(trigger)

	// Показываем прямо под триггером, по центру
	menuPos := fyne.NewPos(
		triggerPos.X,
		triggerPos.Y+trigger.Size().Height,
	)

	// Проверяем, не выходит ли за нижнюю границу окна
	popupSize := nrm.popup.MinSize()
	windowSize := window.Size()

	if menuPos.Y+popupSize.Height > windowSize.Height {
		// Если выходит, показываем над триггером
		menuPos.Y = triggerPos.Y - popupSize.Height - 5
	}

	// Центрируем по горизонтали относительно триггера
	menuPos.X += (trigger.Size().Width - popupSize.Width) / 2

	nrm.popup.ShowAtPosition(menuPos)
}

// createNewRectangleContent создает содержимое компонента NewRectangle
func createNewRectangleContent(breadcrumbManager BreadcrumbManagerInterface, onClose func()) *fyne.Container {
	// Создаем большой контнер в качестве фона, поддерживающий drag-and-drop
	bgContainer := container.NewStack()

	// Здесь будет форма с полями ввода
	form := createInputForm(breadcrumbManager, onClose)

	// Добавляем форму на фоновый контейнер
	bgContainer.Objects = append(bgContainer.Objects, form)

	return bgContainer
}

// createInputForm создает форму с полями ввода
func createInputForm(breadcrumbManager BreadcrumbManagerInterface, onClose func()) fyne.CanvasObject {
	titleEntry := widget.NewEntry()
	titleEntry.PlaceHolder = "Название"
	titleEntry.Resize(fyne.NewSize(300, 30)) // Устанавливаем размер для стабильности

	descriptionEntry := widget.NewMultiLineEntry()
	descriptionEntry.PlaceHolder = "Описание или ссылки"
	descriptionEntry.Resize(fyne.NewSize(300, 60)) // Устанавливаем размер для стабильности

	tagsEntry := widget.NewEntry()
	tagsEntry.PlaceHolder = "Теги( через запятую )"
	tagsEntry.Resize(fyne.NewSize(300, 30)) // Устанавливаем размер для стабильности

	// Создаем состояние для файла
	fileState := &FileUploadState{
		SelectedFiles: &[]string{},
		UpdateDisplay: func() {},
	}

	// Используем функции из file_selector.go
	fileSelectorContainer := CreateFileSelector(fileState)

	// Кнопка создания
	createButton := widget.NewButton("Создать", func() {
		// Логика сохранения элемента
		err := saveNewItemExtended(titleEntry.Text, descriptionEntry.Text, tagsEntry.Text, fileState.SelectedFiles, []string{}, nil)
		if err == nil {
			// Закрываем окно после успешного создания
			if onClose != nil {
				onClose()
			}

			// Обновляем рабочую область через хлебные крошки
			if breadcrumbManager != nil {
				breadcrumbManager.Refresh()
			}
		}
	})
	createButton.Importance = widget.HighImportance

	// Создаем вкладки для переключения типа элемента
	tabs := container.NewAppTabs(
		container.NewTabItem("Элемент", createElementForm(titleEntry, descriptionEntry, tagsEntry, fileSelectorContainer)),
		container.NewTabItem("Папка", createFolderForm(titleEntry, descriptionEntry, tagsEntry, fileSelectorContainer)),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	// Обработчик смены вкладки для определения типа элемента
	tabs.OnSelected = func(tab *container.TabItem) {
		if tab.Text == "Элемент" {
			// Здесь нужно обновить selectedType в менеджере
			// Но так как мы не имеем доступа к менеджеру из этой функции,
			// нужно будет передавать его как параметр или использовать глобальную переменную
		} else {
			// Здесь тоже нужно обновить selectedType
		}
	}

	// Создаем контейнер для выбора папки
	folderSelectionContainer := CreateFolderSelection(breadcrumbManager)

	// Создаем вертикальный контейнер для формы
	formContainer := container.NewVBox(
		tabs,
		createButton,
		widget.NewLabel("Создать в . . ."),
		folderSelectionContainer,
	)

	// Оборачиваем в контейнер с отступами и фоном
	bgRect := canvas.NewRectangle(color.RGBA{0, 0, 0, 255}) // Черный фон
	bgRect.CornerRadius = 5
	bgRect.StrokeColor = color.RGBA{48, 48, 255, 255} // Темно-серая обводка
	bgRect.StrokeWidth = 1
	bgRect.SetMinSize(fyne.NewSize(300, 300)) // Устанавливаем минимальный размер фона

	outerContainer := container.NewStack(bgRect, container.NewPadded(formContainer))

	return outerContainer
}

// createElementForm создает форму для элемента
func createElementForm(titleEntry *widget.Entry, descriptionEntry *widget.Entry, tagsEntry *widget.Entry, fileSelectorContainer fyne.CanvasObject) *fyne.Container {
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Widget: titleEntry},
			{Widget: descriptionEntry},
			{Widget: tagsEntry},
		},
	}
	return container.NewPadded(container.NewVBox(form, fileSelectorContainer))
}

// createFolderForm создает форму для папки
func createFolderForm(titleEntry *widget.Entry, descriptionEntry *widget.Entry, tagsEntry *widget.Entry, fileSelectorContainer fyne.CanvasObject) *fyne.Container {
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Widget: titleEntry},
			{Widget: descriptionEntry},
			{Widget: tagsEntry},
		},
	}
	return container.NewPadded(container.NewVBox(form, fileSelectorContainer))
}

// saveNewItemExtended сохраняет новый элемент в базу данных с расширенной обработкой
func saveNewItemExtended(title, description, tags string, selectedFiles *[]string, linkEntries []string, canvas fyne.Canvas) error {
	// Используем функцию из нового обработчика
	var window fyne.Window
	if canvas != nil {
		window = canvas.(fyne.Window)
	} else {
		// Создаем новое окно для диалогов, если окно не предоставлено
		// В данном случае, используем главное окно приложения
		app := fyne.CurrentApp()
		wins := app.Driver().AllWindows()
		if len(wins) > 0 {
			window = wins[0]
		} else {
			// Если нет окон, создаем новое (это крайний случай)
			window = app.NewWindow("temp")
		}
	}
	return CreateItem(title, description, tags, selectedFiles, linkEntries, selectedFolder.ID, window)
}

// saveNewItem сохраняет новый элемент в базу данных
func saveNewItem(title, description, tags string) error {
	// Здесь должна быть реализация сохранения элемента
	// Создаем новый элемент
	newItem := &models.Item{
		Type:        models.ItemTypeElement, // По умолчанию элемент
		Title:       title,
		Description: description,
		ParentID:    selectedFolder.ID, // Устанавливаем выбранную папку как родителя
	}

	// Вызываем функцию сохранения из queries
	err := queries.CreateItem(newItem)
	if err != nil {
		return err
	}

	// Если были указаны теги, обрабатываем их
	if tags != "" {
		// Разбиваем теги по запятой и сохраняем их
		tagService := services.NewTagsService()
		tagNames := strings.Split(tags, ",")
		for _, tagName := range tagNames {
			tagName = strings.TrimSpace(tagName)
			if tagName != "" {
				// Получаем или создаем тег
				ctx := context.Background()
				tag, err := tagService.GetOrCreateTag(ctx, tagName)
				if err != nil {
					// Логируем ошибку, но не прерываем процесс
					continue
				}

				// Добавляем тег к элементу
				err = tagService.AddTagToItem(ctx, newItem.ID, tag.ID)
				if err != nil {
					// Логируем ошибку, но не прерываем процесс
					continue
				}
			}
		}
	}

	return nil
}
