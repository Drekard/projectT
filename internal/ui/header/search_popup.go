package header

import (
	"image/color"
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
	searchEntry.SetPlaceHolder("Search...")

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
				if swm.handler != nil {
					_ = swm.handler.SearchItems(text)
				}
				lastQuery = text
			}
		})
	}

	return swm
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

// createSearchContent создает содержимое popup поиска
func (swm *SearchWindowManager) createSearchContent() *fyne.Container {
	searchIcon := canvas.NewImageFromResource(theme.SearchIcon())
	searchIcon.SetMinSize(fyne.NewSize(32, 16))

	searchRow := container.NewBorder(nil, nil, searchIcon, nil, swm.searchEntry)

	bgRect := canvas.NewRectangle(color.RGBA{R: 44, G: 44, B: 44, A: 255})
	bgRect.CornerRadius = 8
	bgRect.StrokeColor = color.RGBA{R: 80, G: 80, B: 80, A: 255}
	bgRect.StrokeWidth = 1
	bgRect.SetMinSize(fyne.NewSize(280, 40))

	outerContainer := container.NewStack(bgRect, container.NewPadded(searchRow))

	return outerContainer
}

// GetSearchEntry возвращает поле поиска
func (swm *SearchWindowManager) GetSearchEntry() *widget.Entry {
	return swm.searchEntry
}
