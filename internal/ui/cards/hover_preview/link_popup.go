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

	// Создаем контент для всплывающего окна
	var content fyne.CanvasObject
	if len(links) <= 5 {
		// Если ссылок 5 или меньше, показываем без прокрутки
		content = container.NewVBox(
			widget.NewLabel("Все ссылки:"),
			container.NewVBox(linkObjects...),
		)
	} else {
		// Если больше 5 ссылок, добавляем прокрутку
		content = container.NewVBox(
			widget.NewLabel("Все ссылки:"),
			container.NewVScroll(container.NewVBox(linkObjects...)),
		)
	}

	// Создаем всплывающее окно
	canvas := fyne.CurrentApp().Driver().CanvasForObject(trigger)
	lp.popup = widget.NewPopUp(content, canvas)

	// Устанавливаем размер окна в зависимости от количества ссылок
	maxHeight := float32(400)   // Максимальная высота окна
	itemHeight := float32(30)   // Приблизительная высота одной ссылки
	headerHeight := float32(80) // Высота заголовка и других элементов

	calculatedHeight := headerHeight + float32(len(links))*itemHeight
	if calculatedHeight > maxHeight {
		// Если расчетная высота больше максимальной, используем максимальную и добавляем прокрутку
		lp.popup.Resize(fyne.NewSize(400, maxHeight))
	} else {
		// Иначе используем расчетную высоту
		lp.popup.Resize(fyne.NewSize(400, calculatedHeight))
	}

	return lp
}

// Show показывает всплывающее окно с ссылками
func (lp *LinkPopup) Show(pos fyne.Position) {
	lp.popup.ShowAtPosition(pos)
}

// Hide скрывает всплывающее окно
func (lp *LinkPopup) Hide() {
	lp.popup.Hide()
}

// IsVisible возвращает видимость всплывающего окна
func (lp *LinkPopup) IsVisible() bool {
	return lp.popup.Visible()
}

// Вспомогательная функция для парсинга URL
func ParseURL(urlStr string) *url.URL {
	// Добавляем протокол, если его нет
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		// Возвращаем пустой URL в случае ошибки
		return &url.URL{}
	}

	return parsedURL
}
