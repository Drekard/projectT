package edit_item

import (
	"projectT/internal/storage/database/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// FormWidgets хранит ссылки на виджеты формы
type FormWidgets struct {
	TitleEntry       *widget.Entry
	DescriptionEntry *widget.Entry
	TagsEntry        *widget.Entry
	LinksContainer   *fyne.Container
	LinkEntries      []*widget.Entry
	AddLinkButton    *widget.Button
	Tabs             *container.AppTabs
	ImageUploadArea  *fyne.Container // Область загрузки изображений
	FileUploadArea   *fyne.Container // Область загрузки файлов
}

// CreateRightColumn создает правую колонку с привязкой к ViewModel
func CreateRightColumn(viewModel *CreateItemViewModel) (*fyne.Container, *FormWidgets) {
	widgets := &FormWidgets{}

	// Создаем поля ввода
	widgets.TitleEntry = widget.NewEntry()
	widgets.TitleEntry.PlaceHolder = "Введите название"
	// Устанавливаем начальное значение из ViewModel
	widgets.TitleEntry.SetText(viewModel.Title)

	widgets.DescriptionEntry = widget.NewMultiLineEntry()
	widgets.DescriptionEntry.PlaceHolder = "Введите описание"
	// Устанавливаем начальное значение из ViewModel
	widgets.DescriptionEntry.SetText(viewModel.Description)

	widgets.TagsEntry = widget.NewEntry()
	widgets.TagsEntry.PlaceHolder = "Введите теги (через запятую)"
	// Устанавливаем начальное значение из ViewModel
	widgets.TagsEntry.SetText(viewModel.Tags)

	// Создаем контейнер для ссылок
	widgets.LinksContainer = container.NewVBox()
	widgets.LinksContainer.Objects = []fyne.CanvasObject{
		widget.NewLabel("Ссылки:"),
	}

	// Добавляем существующие ссылки из ViewModel
	for _, link := range viewModel.Links {
		if link != "" {
			// Добавляем пустое поле ввода
			addLinkEntry(widgets)
			// Устанавливаем текст в последнее добавленное поле ввода
			if len(widgets.LinkEntries) > 0 {
				widgets.LinkEntries[len(widgets.LinkEntries)-1].SetText(link)
			}
		}
	}

	// Кнопка добавления ссылки
	widgets.AddLinkButton = widget.NewButton("+ Добавить ссылку", func() {
		addLinkEntry(widgets)
	})
	widgets.AddLinkButton.Importance = widget.LowImportance

	// Привязываем к ViewModel
	widgets.TitleEntry.OnChanged = func(text string) {
		viewModel.Title = text
	}

	widgets.DescriptionEntry.OnChanged = func(text string) {
		viewModel.Description = text
	}

	widgets.TagsEntry.OnChanged = func(text string) {
		viewModel.Tags = text
	}

	// Создаем вкладки
	widgets.Tabs = container.NewAppTabs(
		container.NewTabItem("Элемент", createElementForm(widgets)),
		container.NewTabItem("Папка", createFolderForm(widgets)),
	)
	widgets.Tabs.SetTabLocation(container.TabLocationTop)

	// Обработчик смены вкладки
	widgets.Tabs.OnSelected = func(tab *container.TabItem) {
		if tab.Text == "Элемент" {
			viewModel.ItemType = models.ItemTypeElement
		} else {
			viewModel.ItemType = models.ItemTypeFolder
		}

		// Обновляем видимость элементов формы при смене вкладки
		UpdateFormVisibility(widgets, viewModel.ItemType)
	}

	// Если это режим редактирования папки, выбираем соответствующую вкладку
	if viewModel.EditMode && viewModel.ItemType == models.ItemTypeFolder {
		widgets.Tabs.SelectIndex(1) // Выбираем вкладку "Папка" (индекс 1)
	} else {
		// Для элемента или при создании выбираем вкладку "Элемент"
		widgets.Tabs.SelectIndex(0) // Выбираем вкладку "Элемент" (индекс 0)
	}

	// Обновляем видимость элементов формы в зависимости от типа элемента
	UpdateFormVisibility(widgets, viewModel.ItemType)

	return container.NewVBox(widgets.Tabs), widgets
}

func createElementForm(widgets *FormWidgets) *fyne.Container {
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Название", Widget: widgets.TitleEntry},
			{Text: "Описание", Widget: widgets.DescriptionEntry},
			{Text: "Теги", Widget: widgets.TagsEntry},
		},
	}
	return container.NewPadded(container.NewVBox(form, widgets.LinksContainer, widgets.AddLinkButton))
}

func createFolderForm(widgets *FormWidgets) *fyne.Container {
	// Для папки используем те же поля ввода, но скрываем поля для ссылок
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Название", Widget: widgets.TitleEntry},
			{Text: "Описание", Widget: widgets.DescriptionEntry},
			{Text: "Теги", Widget: widgets.TagsEntry},
		},
	}

	// Создаем контейнер для папки - без ссылок
	return container.NewPadded(container.NewVBox(form))
}

// UpdateFormVisibility обновляет видимость элементов формы в зависимости от типа элемента
func UpdateFormVisibility(widgets *FormWidgets, itemType models.ItemType) {
	if itemType == models.ItemTypeFolder {
		// Для папки скрываем поля ссылок
		widgets.LinksContainer.Hide()
		widgets.AddLinkButton.Hide()

		// Показываем область загрузки изображений (для обложки папки)
		if widgets.ImageUploadArea != nil {
			widgets.ImageUploadArea.Show()
		}

		// Скрываем область загрузки файлов для папок
		if widgets.FileUploadArea != nil {
			widgets.FileUploadArea.Hide()
		}
	} else {
		// Для других типов элементов показываем поля ссылок и области загрузки
		widgets.LinksContainer.Show()
		widgets.AddLinkButton.Show()

		// Показываем области загрузки файлов и изображений, если они существуют
		if widgets.ImageUploadArea != nil {
			widgets.ImageUploadArea.Show()
		}
		if widgets.FileUploadArea != nil {
			widgets.FileUploadArea.Show()
		}
	}
}

func addLinkEntry(widgets *FormWidgets) {
	entry := widget.NewEntry()
	entry.PlaceHolder = "Введите ссылку..."

	// Отключить перенос текста для лучшего растяжения
	entry.Wrapping = fyne.TextWrapOff

	// Использовать контейнер с растягивающимся полем ввода
	var stretchContainer *fyne.Container // Объявляем переменную до использования

	removeButton := widget.NewButton("❌", func() {
		// Удаляем элемент из интерфейса
		widgets.LinksContainer.Remove(stretchContainer)

		// Удаляем элемент из списка
		for i, linkEntry := range widgets.LinkEntries {
			if linkEntry == entry {
				widgets.LinkEntries = append(widgets.LinkEntries[:i], widgets.LinkEntries[i+1:]...)
				break
			}
		}
	})
	removeButton.Importance = widget.LowImportance

	// Создаем контейнер где entry будет растягиваться
	stretchContainer = container.NewBorder(
		nil, nil,
		nil,
		removeButton,
		entry,
	)

	widgets.LinksContainer.Add(stretchContainer)
	widgets.LinkEntries = append(widgets.LinkEntries, entry)
}

// addLinkEntryWithText добавляет поле ввода со ссылкой
func addLinkEntryWithText(widgets *FormWidgets, text string) {
	addLinkEntry(widgets)
	if len(widgets.LinkEntries) > 0 {
		widgets.LinkEntries[len(widgets.LinkEntries)-1].SetText(text)
	}
}
