package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// AppTheme представляет собой тему приложения
type AppTheme struct {
	BackgroundColor     color.Color
	BorderColor         color.Color
	TextColor           color.Color
	PlaceHolderColor    color.Color
	PrimaryColor        color.Color
	HoverColor          color.Color
	FocusColor          color.Color
	ShadowColor         color.Color
	MenuBackgroundColor color.Color
	ButtonColor         color.Color
	DisabledButtonColor color.Color
	ScrollBarColor      color.Color
	SelectionColor      color.Color
	ForegroundColor     color.Color
}

// DarkTheme тема с темными цветами
var DarkTheme = AppTheme{
	BackgroundColor:     color.RGBA{R: 25, G: 25, B: 25, A: 255},    // Темно-серый фон
	BorderColor:         color.RGBA{R: 60, G: 60, B: 60, A: 255},    // Серая граница
	TextColor:           color.RGBA{R: 230, G: 230, B: 230, A: 255}, // Светло-серый текст
	PlaceHolderColor:    color.RGBA{R: 150, G: 150, B: 150, A: 255}, // Более темный плейсхолдер
	PrimaryColor:        color.RGBA{R: 70, G: 130, B: 180, A: 255},  // Стальной синий
	HoverColor:          color.RGBA{R: 50, G: 50, B: 50, A: 255},    // Темнее при наведении
	FocusColor:          color.RGBA{R: 100, G: 150, B: 200, A: 255}, // Цвет фокуса
	ShadowColor:         color.RGBA{R: 0, G: 0, B: 0, A: 100},       // Тень
	MenuBackgroundColor: color.RGBA{R: 30, G: 30, B: 30, A: 255},    // Фон меню
	ButtonColor:         color.RGBA{R: 60, G: 60, B: 60, A: 255},    // Кнопка
	DisabledButtonColor: color.RGBA{R: 80, G: 80, B: 80, A: 255},    // Отключенная кнопка
	ScrollBarColor:      color.RGBA{R: 100, G: 100, B: 100, A: 255}, // Скроллбар
	SelectionColor:      color.RGBA{R: 80, G: 120, B: 160, A: 255},  // Выделение
	ForegroundColor:     color.RGBA{R: 200, G: 200, B: 200, A: 255}, // Передний план
}

// LightTheme тема со светлыми цветами
var LightTheme = AppTheme{
	BackgroundColor:     color.RGBA{R: 245, G: 245, B: 245, A: 255}, // Светло-серый фон
	BorderColor:         color.RGBA{R: 200, G: 200, B: 200, A: 255}, // Светлая граница
	TextColor:           color.RGBA{R: 30, G: 30, B: 30, A: 255},    // Темный текст
	PlaceHolderColor:    color.RGBA{R: 120, G: 120, B: 120, A: 255}, // Темный плейсхолдер
	PrimaryColor:        color.RGBA{R: 70, G: 130, B: 180, A: 255},  // Стальной синий
	HoverColor:          color.RGBA{R: 230, G: 230, B: 230, A: 255}, // Светлее при наведении
	FocusColor:          color.RGBA{R: 100, G: 150, B: 200, A: 255}, // Цвет фокуса
	ShadowColor:         color.RGBA{R: 0, G: 0, B: 0, A: 50},        // Легкая тень
	MenuBackgroundColor: color.RGBA{R: 250, G: 250, B: 250, A: 255}, // Фон меню
	ButtonColor:         color.RGBA{R: 230, G: 230, B: 230, A: 255}, // Кнопка
	DisabledButtonColor: color.RGBA{R: 200, G: 200, B: 200, A: 255}, // Отключенная кнопка
	ScrollBarColor:      color.RGBA{R: 180, G: 180, B: 180, A: 255}, // Скроллбар
	SelectionColor:      color.RGBA{R: 180, G: 200, B: 220, A: 255}, // Выделение
	ForegroundColor:     color.RGBA{R: 50, G: 50, B: 50, A: 255},    // Передний план
}

// BlueTheme синяя тема
var BlueTheme = AppTheme{
	BackgroundColor:     color.RGBA{R: 15, G: 23, B: 42, A: 255},    // Темно-синий фон
	BorderColor:         color.RGBA{R: 30, G: 64, B: 100, A: 255},   // Темно-голубая граница
	TextColor:           color.RGBA{R: 220, G: 230, B: 240, A: 255}, // Светлый текст
	PlaceHolderColor:    color.RGBA{R: 150, G: 170, B: 190, A: 255}, // Светлый плейсхолдер
	PrimaryColor:        color.RGBA{R: 59, G: 130, B: 246, A: 255},  // Ярко-синий
	HoverColor:          color.RGBA{R: 30, G: 41, B: 59, A: 255},    // Темнее при наведении
	FocusColor:          color.RGBA{R: 99, G: 102, B: 241, A: 255},  // Фиолетово-синий фокус
	ShadowColor:         color.RGBA{R: 0, G: 0, B: 0, A: 100},       // Тень
	MenuBackgroundColor: color.RGBA{R: 20, G: 30, B: 48, A: 255},    // Фон меню
	ButtonColor:         color.RGBA{R: 30, G: 64, B: 100, A: 255},   // Кнопка
	DisabledButtonColor: color.RGBA{R: 50, G: 80, B: 120, A: 255},   // Отключенная кнопка
	ScrollBarColor:      color.RGBA{R: 70, G: 100, B: 150, A: 255},  // Скроллбар
	SelectionColor:      color.RGBA{R: 59, G: 130, B: 246, A: 100},  // Выделение
	ForegroundColor:     color.RGBA{R: 200, G: 220, B: 240, A: 255}, // Передний план
}

// GreenTheme зеленая тема
var GreenTheme = AppTheme{
	BackgroundColor:     color.RGBA{R: 16, G: 24, B: 20, A: 255},    // Темно-зеленый фон
	BorderColor:         color.RGBA{R: 34, G: 197, B: 94, A: 255},   // Ярко-зеленая граница
	TextColor:           color.RGBA{R: 220, G: 252, B: 231, A: 255}, // Светлый текст
	PlaceHolderColor:    color.RGBA{R: 160, G: 190, B: 170, A: 255}, // Светлый плейсхолдер
	PrimaryColor:        color.RGBA{R: 34, G: 197, B: 94, A: 255},   // Ярко-зеленый
	HoverColor:          color.RGBA{R: 22, G: 33, B: 28, A: 255},    // Темнее при наведении
	FocusColor:          color.RGBA{R: 74, G: 222, B: 128, A: 255},  // Светло-зеленый фокус
	ShadowColor:         color.RGBA{R: 0, G: 0, B: 0, A: 100},       // Тень
	MenuBackgroundColor: color.RGBA{R: 22, G: 33, B: 28, A: 255},    // Фон меню
	ButtonColor:         color.RGBA{R: 34, G: 197, B: 94, A: 255},   // Кнопка
	DisabledButtonColor: color.RGBA{R: 60, G: 150, B: 90, A: 255},   // Отключенная кнопка
	ScrollBarColor:      color.RGBA{R: 70, G: 150, B: 100, A: 255},  // Скроллбар
	SelectionColor:      color.RGBA{R: 34, G: 197, B: 94, A: 100},   // Выделение
	ForegroundColor:     color.RGBA{R: 200, G: 240, B: 220, A: 255}, // Передний план
}

// PurpleTheme фиолетовая тема на основе указанных цветов
var PurpleTheme = AppTheme{
	BackgroundColor:     color.RGBA{R: 25, G: 15, B: 35, A: 255},    // Темно-фиолетовый фон (на основе смешивания указанных цветов)
	BorderColor:         color.RGBA{R: 226, G: 55, B: 255, A: 255},  // Яркий розово-фиолетовый (первый указанный цвет)
	TextColor:           color.RGBA{R: 240, G: 220, B: 255, A: 255}, // Светло-фиолетовый текст
	PlaceHolderColor:    color.RGBA{R: 180, G: 150, B: 200, A: 255}, // Тусклый фиолетовый плейсхолдер
	PrimaryColor:        color.RGBA{R: 133, G: 55, B: 255, A: 255},  // Второй указанный цвет (фиолетовый)
	HoverColor:          color.RGBA{R: 45, G: 25, B: 55, A: 255},    // Темнее при наведении
	FocusColor:          color.RGBA{R: 200, G: 100, B: 230, A: 255}, // Светло-фиолетовый фокус (между двумя указанными цветами)
	ShadowColor:         color.RGBA{R: 0, G: 0, B: 0, A: 100},       // Тень
	MenuBackgroundColor: color.RGBA{R: 30, G: 20, B: 40, A: 255},    // Фон меню (темнее)
	ButtonColor:         color.RGBA{R: 144, G: 55, B: 255, A: 255},  // Кнопка с первым цветом
	DisabledButtonColor: color.RGBA{R: 120, G: 80, B: 150, A: 255},  // Отключенная кнопка
	ScrollBarColor:      color.RGBA{R: 150, G: 100, B: 180, A: 255}, // Скроллбар
	SelectionColor:      color.RGBA{R: 226, G: 55, B: 255, A: 100},  // Выделение с прозрачностью
	ForegroundColor:     color.RGBA{R: 220, G: 200, B: 240, A: 255}, // Передний план
}

// CurrentTheme текущая тема приложения
var CurrentTheme = PurpleTheme

// SetTheme устанавливает новую тему
func SetTheme(newTheme AppTheme) {
	CurrentTheme = newTheme
}

// GetTheme возвращает текущую тему
func GetTheme() AppTheme {
	return CurrentTheme
}

// FyneTheme адаптер для использования с Fyne
type FyneTheme struct{}

// Color возвращает цвет для указанного типа и варианта
func (t FyneTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return CurrentTheme.BackgroundColor
	case theme.ColorNameForeground:
		return CurrentTheme.ForegroundColor
	case theme.ColorNameInputBorder:
		return CurrentTheme.BorderColor
	case theme.ColorNamePrimary:
		return CurrentTheme.PrimaryColor
	case theme.ColorNameFocus:
		return CurrentTheme.FocusColor
	case theme.ColorNameHover:
		return CurrentTheme.HoverColor
	case theme.ColorNamePlaceHolder:
		return CurrentTheme.PlaceHolderColor
	case theme.ColorNameSelection:
		return CurrentTheme.SelectionColor
	case theme.ColorNameScrollBar:
		return CurrentTheme.ScrollBarColor
	case theme.ColorNameMenuBackground:
		return CurrentTheme.MenuBackgroundColor
	case theme.ColorNameButton:
		return CurrentTheme.ButtonColor
	case theme.ColorNameDisabledButton:
		return CurrentTheme.DisabledButtonColor
	case theme.ColorNameShadow:
		return CurrentTheme.ShadowColor
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

// Icon возвращает иконку для указанного имени
func (t FyneTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

// Font возвращает шрифт для указанного стиля
func (t FyneTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

// Size возвращает размер для указанного типа
func (t FyneTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

// GetFyneTheme возвращает FyneTheme для использования в приложении
func GetFyneTheme() fyne.Theme {
	return FyneTheme{}
}
