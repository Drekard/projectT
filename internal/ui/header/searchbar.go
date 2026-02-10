package header

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type SearchHandler interface {
	SearchItems(query string) error
	ClearSearch() error
}

func CreateSearchBar(workspace SearchHandler) (*fyne.Container, *widget.Entry) {
	searchIcon := canvas.NewImageFromResource(theme.SearchIcon())
	searchIcon.SetMinSize(fyne.NewSize(24, 24))
	searchIcon.FillMode = canvas.ImageFillContain

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Поиск...")

	// Обработчик поиска с задержкой
	var searchTimer *time.Timer
	var lastQuery string

	searchEntry.OnChanged = func(text string) {
		// Отменяем предыдущий таймер, если он существует
		if searchTimer != nil {
			searchTimer.Stop()
		}

		// Если текст пустой, очищаем результаты поиска
		if text == "" && lastQuery != "" {
			if workspace != nil {
				workspace.ClearSearch()
			}
			lastQuery = ""
			return
		}

		// Устанавливаем таймер для задержки поиска
		searchTimer = time.AfterFunc(500*time.Millisecond, func() {
			if text != lastQuery {
				if workspace != nil {
					workspace.SearchItems(text)
				}
				lastQuery = text
			}
		})
	}

	// Вместо HBox используем Border с поиском в центре
	searchContainer := container.NewBorder(
		nil,
		nil,
		searchIcon, // слева иконка
		nil,
		searchEntry, // по центру поле ввода
	)

	// Добавляем отступы для красоты
	return container.NewPadded(searchContainer), searchEntry
}
