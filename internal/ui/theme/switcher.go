package theme

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ThemeSwitcher предоставляет интерфейс для переключения тем
type ThemeSwitcher struct {
	dropdown *widget.Select
	window   fyne.Window
}

// NewThemeSwitcher создает новый переключатель тем
func NewThemeSwitcher(window fyne.Window) *ThemeSwitcher {
	switcher := &ThemeSwitcher{
		window: window,
	}

	// Создаем выпадающий список с названиями тем
	themes := []string{"Dark Theme", "Light Theme", "Blue Theme", "Green Theme", "Purple Theme"}
	switcher.dropdown = widget.NewSelect(themes, func(themeName string) {
		switch themeName {
		case "Dark Theme":
			SetTheme(DarkTheme)
		case "Light Theme":
			SetTheme(LightTheme)
		case "Blue Theme":
			SetTheme(BlueTheme)
		case "Green Theme":
			SetTheme(GreenTheme)
		case "Purple Theme":
			SetTheme(PurpleTheme)
		}

		// Обновляем тему в настройках приложения
		if app := fyne.CurrentApp(); app != nil {
			app.Settings().SetTheme(GetFyneTheme())
		}
	})

	// Устанавливаем начальное значение в соответствии с текущей темой
	switch CurrentTheme {
	case DarkTheme:
		switcher.dropdown.SetSelected("Dark Theme")
	case LightTheme:
		switcher.dropdown.SetSelected("Light Theme")
	case BlueTheme:
		switcher.dropdown.SetSelected("Blue Theme")
	case GreenTheme:
		switcher.dropdown.SetSelected("Green Theme")
	case PurpleTheme:
		switcher.dropdown.SetSelected("Purple Theme")
	}

	return switcher
}

// GetWidget возвращает виджет переключателя тем для добавления в интерфейс
func (ts *ThemeSwitcher) GetWidget() fyne.CanvasObject {
	return container.NewHBox(
		widget.NewLabel("Theme:"),
		ts.dropdown,
	)
}

// GetCurrentTheme возвращает текущую тему
func (ts *ThemeSwitcher) GetCurrentTheme() AppTheme {
	return CurrentTheme
}
