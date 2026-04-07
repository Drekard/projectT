package hover_preview

import (
	"net/url"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// LinkPopup компонент для отображения всплывающего окна со списком ссылок
type LinkPopup struct {
	popup *widget.PopUp
}

// NewLinkPopup создает новый компонент для отображения списка ссылок
func NewLinkPopup(links []string, trigger fyne.CanvasObject) *LinkPopup {
	lp := &LinkPopup{}

	// Создаем список ссылок
	var linkObjects []fyne.CanvasObject
	for _, link := range links {
		linkWidget := widget.NewHyperlink(link, ParseURL(link))
		linkObjects = append(linkObjects, linkWidget)
	}

	// Create content for the popup
	var content fyne.CanvasObject
	if len(links) <= 5 {
		// If 5 or fewer links, show without scrolling
		content = container.NewVBox(
			widget.NewLabel("All links:"),
			container.NewVBox(linkObjects...),
		)
	} else {
		// If more than 5 links, add scrolling
		content = container.NewVBox(
			widget.NewLabel("All links:"),
			container.NewVScroll(container.NewVBox(linkObjects...)),
		)
	}

	// Создаем всплывающее окно
	canvas := fyne.CurrentApp().Driver().CanvasForObject(trigger)
	lp.popup = widget.NewPopUp(content, canvas)

	// Set window size based on number of links
	maxHeight := float32(400)   // Maximum window height
	itemHeight := float32(30)   // Approximate height of one link
	headerHeight := float32(80) // Height of header and other elements

	calculatedHeight := headerHeight + float32(len(links))*itemHeight
	if calculatedHeight > maxHeight {
		// If calculated height exceeds maximum, use maximum and add scrolling
		lp.popup.Resize(fyne.NewSize(400, maxHeight))
	} else {
		// Otherwise use calculated height
		lp.popup.Resize(fyne.NewSize(400, calculatedHeight))
	}

	return lp
}

// Show displays the link popup
func (lp *LinkPopup) Show(pos fyne.Position) {
	lp.popup.ShowAtPosition(pos)
}

// Hide hides the popup
func (lp *LinkPopup) Hide() {
	lp.popup.Hide()
}

// IsVisible returns the visibility state of the popup
func (lp *LinkPopup) IsVisible() bool {
	return lp.popup.Visible()
}

// Helper function for parsing URLs
func ParseURL(urlStr string) *url.URL {
	// Add protocol if missing
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		// Return empty URL on error
		return &url.URL{}
	}

	return parsedURL
}
