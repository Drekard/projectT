package header

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// SearchWindowManager управляет popup поиска
type SearchWindowManager struct {
	popup       *widget.PopUp
	searchEntry *widget.Entry
	handler     HeaderSearchHandler
}

// NewSearchWindowManager создает новый менеджер popup поиска
func NewSearchWindowManager(handler HeaderSearchHandler) *SearchWindowManager {
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search... (free text searches title/description)")

	swm := &SearchWindowManager{
		searchEntry: searchEntry,
		handler:     handler,
	}

	var searchTimer *time.Timer
	var lastQuery string

	searchEntry.OnChanged = func(text string) {
		if searchTimer != nil {
			searchTimer.Stop()
		}

		if text == "" && lastQuery != "" {
			if swm.handler != nil {
				_ = swm.handler.ClearSearch()
			}
			lastQuery = ""
			return
		}

		searchTimer = time.AfterFunc(500*time.Millisecond, func() {
			if text != lastQuery {
				query := buildSearchQuery(text)
				if swm.handler != nil {
					_ = swm.handler.SearchItems(query)
				}
				lastQuery = text
			}
		})
	}

	return swm
}

// buildSearchQuery преобразует пользовательский ввод в структурированный запрос
// Свободный текст автоматически помечается как Text:
func buildSearchQuery(input string) string {
	if input == "" {
		return ""
	}

	parts := strings.Fields(input)
	var queryParts []string

	for _, part := range parts {
		lower := strings.ToLower(part)
		if strings.HasPrefix(lower, "before:") || strings.HasPrefix(lower, "after:") ||
			strings.HasPrefix(lower, "on:") || strings.HasPrefix(lower, "tags:") {
			queryParts = append(queryParts, part)
		} else {
			queryParts = append(queryParts, "Text:"+part)
		}
	}

	return strings.Join(queryParts, " ")
}

// ShowSearchPopup показывает popup поиска под кнопкой
func (swm *SearchWindowManager) ShowSearchPopup(trigger fyne.CanvasObject) {
	window := fyne.CurrentApp().Driver().CanvasForObject(trigger)
	if window == nil {
		return
	}

	content := swm.createSearchContent()

	swm.popup = widget.NewPopUp(content, window)

	triggerPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(trigger)

	menuPos := fyne.NewPos(
		triggerPos.X+(trigger.Size().Width-swm.popup.MinSize().Width)/2,
		triggerPos.Y+trigger.Size().Height,
	)

	popupSize := swm.popup.MinSize()
	windowSize := window.Size()

	if menuPos.Y+popupSize.Height > windowSize.Height {
		menuPos.Y = triggerPos.Y - popupSize.Height - 5
	}

	swm.popup.ShowAtPosition(menuPos)
	fyne.CurrentApp().Driver().CanvasForObject(trigger).Focus(swm.searchEntry)
}

// createSearchContent создает содержимое popup поиска с фильтрами
func (swm *SearchWindowManager) createSearchContent() *fyne.Container {
	searchIcon := canvas.NewImageFromResource(theme.SearchIcon())
	searchIcon.SetMinSize(fyne.NewSize(32, 16))

	clearBtn := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
		swm.searchEntry.SetText("")
	})
	clearBtn.Importance = widget.LowImportance

	searchRow := container.NewBorder(nil, nil, searchIcon, clearBtn, swm.searchEntry)

	// Filter buttons
	filterButtons := container.NewHBox(
		swm.createFilterButton("before:", "Before date"),
		swm.createFilterButton("after:", "After date"),
		swm.createFilterButton("on:", "On date"),
		swm.createFilterButton("tags:", "Tags"),
	)

	bgRect := canvas.NewRectangle(color.RGBA{R: 44, G: 44, B: 44, A: 255})
	bgRect.CornerRadius = 8
	bgRect.StrokeColor = color.RGBA{R: 80, G: 80, B: 80, A: 255}
	bgRect.StrokeWidth = 1
	bgRect.SetMinSize(fyne.NewSize(400, 80))

	outerContainer := container.NewStack(bgRect, container.NewPadded(
		container.NewVBox(searchRow, filterButtons),
	))

	return outerContainer
}

// createFilterButton создает кнопку фильтра, которая вставляет префикс в поисковую строку
func (swm *SearchWindowManager) createFilterButton(prefix, _ string) *widget.Button {
	btn := widget.NewButton(prefix, func() {
		currentText := swm.searchEntry.Text
		if currentText != "" && !strings.HasSuffix(currentText, " ") {
			currentText += " "
		}
		swm.searchEntry.SetText(currentText + prefix)
		fyne.CurrentApp().Driver().CanvasForObject(swm.searchEntry).Focus(swm.searchEntry)
	})
	btn.Importance = widget.LowImportance
	return btn
}

// GetSearchEntry возвращает поле поиска
func (swm *SearchWindowManager) GetSearchEntry() *widget.Entry {
	return swm.searchEntry
}

// FormatDateFilter форматирует фильтр даты для отображения
func FormatDateFilter(filterType, date string) string {
	return fmt.Sprintf("%s:%s", filterType, date)
}
