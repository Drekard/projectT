package header

import (
	"fyne.io/fyne/v2/widget"
)

// CreateMenuButton создает кнопку "бургер" для открытия/сворачивания левой панели
func CreateMenuButton(sidebarVisible *bool, onToggle func()) *widget.Button {
	button := widget.NewButton("", func() {
		*sidebarVisible = !*sidebarVisible
		onToggle()
	})
	button.Importance = widget.LowImportance
	// Иконка будет установлена в основном приложении, так как тема недоступна в этом пакете

	return button
}
