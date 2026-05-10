package hover_preview

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// parseHexColor преобразует HEX цвет в RGBA
func parseHexColor(hex string) (color.RGBA, error) {
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}

	var r, g, b uint8
	var a uint8 = 255

	switch len(hex) {
	case 3:
		var ir, ig, ib int
		n, err := fmt.Sscanf(hex, "%1x%1x%1x", &ir, &ig, &ib)
		if n != 3 || err != nil {
			return color.RGBA{}, fmt.Errorf("неправильный формат HEX цвета: %s", hex)
		}
		r, g, b = uint8(ir*17), uint8(ig*17), uint8(ib*17)
	case 6:
		var ir, ig, ib int
		n, err := fmt.Sscanf(hex, "%02x%02x%02x", &ir, &ig, &ib)
		if n != 3 || err != nil {
			return color.RGBA{}, fmt.Errorf("неправильный формат HEX цвета: %s", hex)
		}
		r, g, b = uint8(ir), uint8(ig), uint8(ib)
	default:
		return color.RGBA{}, fmt.Errorf("неправильная длина HEX цвета: %s", hex)
	}

	return color.RGBA{R: r, G: g, B: b, A: a}, nil
}

// getContrastColor возвращает контрастный цвет (черный или белый) в зависимости от фона
func getContrastColor(backgroundColor color.RGBA) color.Color {
	luminance := (0.299*float64(backgroundColor.R) + 0.587*float64(backgroundColor.G) + 0.114*float64(backgroundColor.B)) / 255.0

	if luminance > 0.5 {
		return color.RGBA{R: 0, G: 0, B: 0, A: 255}
	}
	return color.RGBA{R: 255, G: 255, B: 255, A: 255}
}

// TagButton - виджет для отображения тега с цветным фоном
type TagButton struct {
	widget.BaseWidget
	text          string
	bgColor       color.RGBA
	textColor     color.Color
	onSingleClick func()
	OnMouseIn     func()
	OnMouseOut    func()
	OnTapped      func()
}

// NewTagButton создает новый тег-баттон
func NewTagButton(text string, bgColor color.RGBA, textColor color.Color, onSingleClick func()) *TagButton {
	tb := &TagButton{
		text:          text,
		bgColor:       bgColor,
		textColor:     textColor,
		onSingleClick: onSingleClick,
	}
	tb.ExtendBaseWidget(tb)
	return tb
}

// MouseIn вызывается при наведении курсора
func (tb *TagButton) MouseIn(_ *fyne.PointEvent) {
	if tb.OnMouseIn != nil {
		tb.OnMouseIn()
	}
}

// MouseOut вызывается при уходе курсора
func (tb *TagButton) MouseOut() {
	if tb.OnMouseOut != nil {
		tb.OnMouseOut()
	}
}

// CreateRenderer создает рендерер для TagButton
func (tb *TagButton) CreateRenderer() fyne.WidgetRenderer {
	textObj := canvas.NewText(tb.text, tb.textColor)
	textObj.TextSize = 12
	textObj.Alignment = fyne.TextAlignCenter

	bgRect := canvas.NewRectangle(tb.bgColor)
	bgRect.CornerRadius = 15
	bgRect.StrokeColor = color.RGBA{48, 48, 48, 255}
	bgRect.StrokeWidth = 1

	centerContainer := container.NewCenter(textObj)

	stack := container.NewStack(bgRect, centerContainer)

	return &TagButtonRenderer{
		tagButton: tb,
		bgRect:    bgRect,
		textObj:   textObj,
		container: stack,
		objects:   []fyne.CanvasObject{stack},
	}
}

// MinSize возвращает минимальный размер
func (tb *TagButton) MinSize() fyne.Size {
	return fyne.NewSize(60, 30)
}

// Tapped обрабатывает одинарный клик
func (tb *TagButton) Tapped(_ *fyne.PointEvent) {
	if tb.onSingleClick != nil {
		tb.onSingleClick()
	}
}

// DoubleTapped обрабатывает двойной клик
func (tb *TagButton) DoubleTapped(_ *fyne.PointEvent) {
	if tb.OnTapped != nil {
		tb.OnTapped()
	}
}

// TagButtonRenderer - рендерер для TagButton
type TagButtonRenderer struct {
	tagButton *TagButton
	bgRect    *canvas.Rectangle
	textObj   *canvas.Text
	container fyne.CanvasObject
	objects   []fyne.CanvasObject
}

// Layout распологает элементы
func (r *TagButtonRenderer) Layout(size fyne.Size) {
	r.container.Resize(size)
}

// MinSize возвращает минимальный размер
func (r *TagButtonRenderer) MinSize() fyne.Size {
	return r.tagButton.MinSize()
}

// Refresh обновляет отображение
func (r *TagButtonRenderer) Refresh() {
	r.bgRect.FillColor = r.tagButton.bgColor
	r.textObj.Color = r.tagButton.textColor
	r.textObj.Text = r.tagButton.text
	canvas.Refresh(r.tagButton)
}

// Objects возвращает объекты для рендеринга
func (r *TagButtonRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

// Destroy освобождает ресурсы
func (r *TagButtonRenderer) Destroy() {}
