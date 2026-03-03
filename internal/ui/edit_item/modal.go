package edit_item

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
)

// ShowCreateItemModalForEdit показывает модальное окно для редактирования элемента
func ShowCreateItemModalForEdit(parentWindow fyne.Window, itemID int) {
	// Создаем ViewModel для редактирования
	viewModel, err := NewCreateItemViewModelForEdit(itemID)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Ошибка загрузки элемента для редактирования: %v", err), parentWindow)
		return
	}

	rightColumn, formWidgets := CreateRightColumn(viewModel)
	leftColumn := CreateLeftColumn(parentWindow, viewModel, formWidgets)

	// Главный контейнер с двумя колонками
	mainContainer := container.NewHSplit(leftColumn, rightColumn)
	mainContainer.Offset = 0.3 // Левая колонка занимает 30% ширины

	// Создаем контейнер с содержимым
	modalContent := container.NewVBox(mainContainer)

	// Устанавливаем минимальный размер для диалога
	modalContent.Resize(fyne.NewSize(750, 300))

	// Создаем диалог, чтобы получить контроль над ним
	d := dialog.NewCustom("Редактирование элемента", "Закрыть", modalContent, parentWindow)

	// Устанавливаем функцию закрытия диалога
	formWidgets.CloseDialog = d.Hide

	// Показываем диалог
	d.Show()
}
