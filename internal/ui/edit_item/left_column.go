package edit_item

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// CreateLeftColumn создает левую колонку с кнопками загрузки
func CreateLeftColumn(modalWindow fyne.Window, viewModel *CreateItemViewModel,
	formWidgets *FormWidgets) *fyne.Container {

	// Создаем состояния для изображений и файлов
	imageState := &FileUploadState{
		SelectedFiles: &viewModel.Images,
		UpdateDisplay: func() {},
	}

	fileState := &FileUploadState{
		SelectedFiles: &viewModel.Files,
		UpdateDisplay: func() {},
	}

	// Создаем область для загрузки изображений
	imageConfig := FileUploadConfig{
		Label:           "🖼️ Добавить изображение/обложку",
		Filter:          []string{".png", ".jpg", ".jpeg", ".gif", ".bmp"},
		BackgroundColor: color.RGBA{R: 30, G: 30, B: 30, A: 25},
		MinSize:         fyne.NewSize(150, 100),
		UploadType:      ImageUpload,
	}

	imageUploadArea := CreateFileUploadArea(imageConfig, imageState, modalWindow)
	formWidgets.ImageUploadArea = imageUploadArea

	// Создаем область для загрузки файлов
	fileConfig := FileUploadConfig{
		Label:           "📎 Добавить файл",
		Filter:          nil,
		BackgroundColor: color.RGBA{R: 30, G: 30, B: 30, A: 25},
		MinSize:         fyne.NewSize(150, 60),
		UploadType:      FileUpload,
	}

	fileUploadArea := CreateFileUploadArea(fileConfig, fileState, modalWindow)
	formWidgets.FileUploadArea = fileUploadArea

	// Кнопка "Создать" или "Сохранить изменения"
	buttonText := "Создать"
	if viewModel.EditMode {
		buttonText = "Сохранить изменения"
	}

	createButton := widget.NewButton(buttonText, func() {
		SaveItem(viewModel, formWidgets, modalWindow)
	})

	createButton.Importance = widget.HighImportance

	// Компоновка левой колонки
	leftContent := container.NewVBox(
		formWidgets.ImageUploadArea,
		widget.NewSeparator(),
		formWidgets.FileUploadArea,
		widget.NewSeparator(),
		container.NewPadded(container.NewCenter(createButton)),
	)

	return leftContent
}
