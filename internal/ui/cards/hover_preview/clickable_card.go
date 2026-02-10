package hover_preview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// ClickableCard - виджет, который обрабатывает клики
type ClickableCard struct {
	widget.BaseWidget
	content        fyne.CanvasObject
	onTapped       func()
	onDoubleTapped func()
	onTappedXY     func(fyne.Position)
}

// NewClickableCard создает новый кликабельный виджет
func NewClickableCard(content fyne.CanvasObject, onTapped func()) *ClickableCard {
	c := &ClickableCard{
		content:  content,
		onTapped: onTapped,
	}
	c.ExtendBaseWidget(c)
	return c
}

// NewClickableCardWithDoubleTap создает новый кликабельный виджет с поддержкой двойного клика
func NewClickableCardWithDoubleTap(content fyne.CanvasObject, onTapped func(), onDoubleTapped func()) *ClickableCard {
	c := &ClickableCard{
		content:        content,
		onTapped:       onTapped,
		onDoubleTapped: onDoubleTapped,
	}
	c.ExtendBaseWidget(c)
	return c
}

// CreateRenderer создает рендерер для виджета
func (c *ClickableCard) CreateRenderer() fyne.WidgetRenderer {
	return &clickableCardRenderer{
		card:    c,
		objects: []fyne.CanvasObject{c.content},
	}
}

// Tapped обрабатывает клик
func (c *ClickableCard) Tapped(*fyne.PointEvent) {
	if c.onTapped != nil {
		c.onTapped()
	}
}

// DoubleTapped обрабатывает двойной клик
func (c *ClickableCard) DoubleTapped(*fyne.PointEvent) {
	if c.onDoubleTapped != nil {
		c.onDoubleTapped()
	}
}

// TappedSecondary обрабатывает правый клик
func (c *ClickableCard) TappedSecondary(*fyne.PointEvent) {
	if c.onTapped != nil {
		c.onTapped()
	}
}

// clickableCardRenderer рендерер для ClickableCard
type clickableCardRenderer struct {
	card    *ClickableCard
	objects []fyne.CanvasObject
}

func (r *clickableCardRenderer) Layout(size fyne.Size) {
	r.objects[0].Resize(size)
}

func (r *clickableCardRenderer) MinSize() fyne.Size {
	return r.objects[0].MinSize()
}

func (r *clickableCardRenderer) Refresh() {
	r.objects[0].Refresh()
}

func (r *clickableCardRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *clickableCardRenderer) Destroy() {}
