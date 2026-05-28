package create_item

import (
	"image/color"
	"strings"

	"projectT/internal/storage/database/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
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
var selectedFolder *SelectedFolder = &SelectedFolder{ID: nil, Name: "Saved"}

// Global переменная для режима создания отдельных элементов из файлов
var multiFileMode bool = false

// Global переменная для отображения описания на карточке
var showDescriptionOnCard bool = false

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
		// Если всё равно не влезает сверху, показываем от верхнего края окна
		if menuPos.Y < 0 {
			menuPos.Y = 0
		}
	}

	// Центрируем по горизонтали относительно триггера
	menuPos.X += (trigger.Size().Width - popupSize.Width) / 2
	// Не даём уйти за левый край
	if menuPos.X < 0 {
		menuPos.X = 0
	}

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
	titleEntry.PlaceHolder = "Title"
	titleEntry.Resize(fyne.NewSize(300, 30))

	descriptionEntry := widget.NewMultiLineEntry()
	descriptionEntry.PlaceHolder = "Description or links"
	descriptionEntry.Resize(fyne.NewSize(300, 60))

	tagsEntry := widget.NewEntry()
	tagsEntry.PlaceHolder = "Tags (comma separated)"
	tagsEntry.Resize(fyne.NewSize(300, 30))

	// Получаем окно для диалогов
	var parentWindow fyne.Window
	windows := fyne.CurrentApp().Driver().AllWindows()
	if len(windows) > 0 {
		parentWindow = windows[0]
	} else {
		return container.NewCenter(widget.NewLabel("Error: no window available"))
	}

	// Создаем состояние для файлов
	fileState := &FileUploadState{
		SelectedFiles: &[]string{},
		UpdateDisplay: func() {},
	}

	// Создаем область для загрузки файлов
	fileConfig := FileUploadConfig{
		Label:           "Add a file/image",
		Filter:          nil,
		BackgroundColor: color.RGBA{R: 30, G: 30, B: 30, A: 25},
		MinSize:         fyne.NewSize(150, 30),
	}

	fileUploadArea := CreateFileUploadArea(fileConfig, fileState, parentWindow)

	// Кнопка создания
	createButton := widget.NewButton("Create", func() {
		// Проверка на наличие хотя бы одного тега
		if strings.TrimSpace(tagsEntry.Text) == "" {
			dialog.ShowInformation("Validation Error", "Please add at least one tag", parentWindow)
			return
		}

		if multiFileMode && len(*fileState.SelectedFiles) > 1 {
			// Режим создания отдельных элементов для каждого файла
			for _, filePath := range *fileState.SelectedFiles {
				singleFileSlice := []string{filePath}
				err := saveNewItemExtended("", "", tagsEntry.Text, &singleFileSlice, []string{}, nil, false)
				if err != nil {
					break
				}
			}
		} else {
			// Обычный режим создания одного элемента
			err := saveNewItemExtended(titleEntry.Text, descriptionEntry.Text, tagsEntry.Text, fileState.SelectedFiles, []string{}, nil, showDescriptionOnCard)
			if err == nil {
				if onClose != nil {
					onClose()
				}

				if breadcrumbManager != nil {
					breadcrumbManager.Refresh()
				}
			}
		}

		// Закрываем popup и обновляем после создания всех элементов
		if multiFileMode {
			if onClose != nil {
				onClose()
			}
			if breadcrumbManager != nil {
				breadcrumbManager.Refresh()
			}
		}
	})
	createButton.Importance = widget.HighImportance

	// Создаем контейнер с областью загрузки
	fileSelectorContainer := container.NewVBox(fileUploadArea)

	// Создаем вкладки для переключения типа элемента
	tabs := container.NewAppTabs(
		container.NewTabItem("Element", createElementForm(titleEntry, descriptionEntry, tagsEntry, fileSelectorContainer, &multiFileMode)),
		container.NewTabItem("Folder", createFolderForm(titleEntry, descriptionEntry, tagsEntry, fileSelectorContainer)),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	// Обработчик смены вкладки для определения типа элемента
	tabs.OnSelected = func(tab *container.TabItem) {
		switch tab.Text {
		case "Element":
			selectedType = ItemTypeElement
		case "Folder":
			selectedType = ItemTypeFolder
		}
	}

	// Устанавливаем начальный тип элемента в соответствии с активной вкладкой (по умолчанию "Element")
	selectedType = ItemTypeElement

	// Создаем контейнер для выбора папки
	folderSelectionContainer := CreateFolderSelection(breadcrumbManager)

	// Создаем вертикальный контейнер для формы
	formContainer := container.NewVBox(
		tabs,
		createButton,
		widget.NewLabel("Save in . . ."),
		folderSelectionContainer,
	)

	// Оборачиваем в контейнер с отступами и фоном
	bgRect := canvas.NewRectangle(color.RGBA{0, 0, 0, 255})
	bgRect.CornerRadius = 5
	bgRect.StrokeColor = color.RGBA{48, 48, 255, 255}
	bgRect.StrokeWidth = 1
	bgRect.SetMinSize(fyne.NewSize(300, 300))

	outerContainer := container.NewStack(bgRect, container.NewPadded(formContainer))

	// Оборачиваем в прокрутку, если форма не влезает в окно
	scrollContainer := container.NewScroll(outerContainer)
	scrollContainer.SetMinSize(fyne.NewSize(320, 500))

	return scrollContainer
}

// createElementForm создает форму для элемента
func createElementForm(titleEntry *widget.Entry, descriptionEntry *widget.Entry, tagsEntry *widget.Entry, fileSelectorContainer fyne.CanvasObject, multiFileMode *bool) *fyne.Container {
	multiFileCheck := widget.NewCheck("Create separate items for each file", func(checked bool) {
		*multiFileMode = checked
	})

	showDescCheck := widget.NewCheck("Show description on card", func(checked bool) {
		showDescriptionOnCard = checked
	})
	showDescCheck.Checked = false

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Widget: titleEntry},
			{Widget: descriptionEntry},
			{Widget: tagsEntry},
			{Widget: showDescCheck},
			{Widget: multiFileCheck},
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
func saveNewItemExtended(title, description, tags string, selectedFiles *[]string, linkEntries []string, canvas fyne.Canvas, showDescription bool) error {
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

	// Use the selected folder ID from the global variable
	// If it's nil (meaning "Saved"), we use the current folder from breadcrumbs
	parentID := selectedFolder.ID

	// Determine the item type based on the selected tab
	var itemType models.ItemType
	if selectedType == ItemTypeFolder {
		itemType = models.ItemTypeFolder
	} else {
		itemType = models.ItemTypeElement
	}

	return CreateItem(title, description, tags, selectedFiles, linkEntries, parentID, itemType, window, showDescription)
}
