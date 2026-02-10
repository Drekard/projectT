package edit_item

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
)

var createItemModalWindow fyne.Window

// ShowCreateItemModalForEdit показывает модальное окно для редактирования элемента
func ShowCreateItemModalForEdit(parentWindow fyne.Window, itemID int) {
	// Проверяем, не открыто ли уже окно
	if createItemModalWindow != nil {
		// Если окно уже открыто, просто делаем его активным
		createItemModalWindow.RequestFocus()
		return
	}

	// Создаем ViewModel для редактирования
	viewModel, err := NewCreateItemViewModelForEdit(itemID)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Ошибка загрузки элемента для редактирования: %v", err), parentWindow)
		return
	}

	// Создаем новое окно
	window := fyne.CurrentApp().NewWindow("Редактирование элемента")
	createItemModalWindow = window // Сохраняем ссылку на окно

	rightColumn, formWidgets := CreateRightColumn(viewModel)
	leftColumn := CreateLeftColumn(window, viewModel, formWidgets)

	// Главный контейнер с двумя колонками
	mainContainer := container.NewHSplit(leftColumn, rightColumn)
	mainContainer.Offset = 0.3 // Левая колонка занимает 30% ширины

	// Создаем контейнер с содержимым
	modalContent := container.NewVBox(mainContainer)

	// Устанавливаем контент и размеры
	window.SetContent(modalContent)
	window.Resize(fyne.NewSize(750, 300))

	// При закрытии окна сбрасываем ссылку
	window.SetOnClosed(func() {
		createItemModalWindow = nil
	})

	window.Show()
}
