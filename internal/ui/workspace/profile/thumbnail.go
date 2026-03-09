package profile

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// ThumbnailClickable - виджет миниатюры с поддержкой двойного клика
type ThumbnailClickable struct {
	widget.BaseWidget
	content        fyne.CanvasObject
	onDoubleTapped func()
}

// NewThumbnailClickable создает новый виджет миниатюры с поддержкой двойного клика
func NewThumbnailClickable(content fyne.CanvasObject, onDoubleTapped func()) *ThumbnailClickable {
	c := &ThumbnailClickable{
		content:        content,
		onDoubleTapped: onDoubleTapped,
	}
	c.ExtendBaseWidget(c)
	return c
}

// CreateRenderer создает рендерер для виджета
func (c *ThumbnailClickable) CreateRenderer() fyne.WidgetRenderer {
	return &thumbnailClickableRenderer{
		card:    c,
		objects: []fyne.CanvasObject{c.content},
	}
}

// DoubleTapped обрабатывает двойной клик
func (c *ThumbnailClickable) DoubleTapped(*fyne.PointEvent) {
	if c.onDoubleTapped != nil {
		c.onDoubleTapped()
	}
}

// thumbnailClickableRenderer рендерер для ThumbnailClickable
type thumbnailClickableRenderer struct {
	card    *ThumbnailClickable
	objects []fyne.CanvasObject
}

func (r *thumbnailClickableRenderer) Layout(size fyne.Size) {
	r.objects[0].Resize(size)
}

func (r *thumbnailClickableRenderer) MinSize() fyne.Size {
	return r.objects[0].MinSize()
}

func (r *thumbnailClickableRenderer) Refresh() {
	r.objects[0].Refresh()
}

func (r *thumbnailClickableRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *thumbnailClickableRenderer) Destroy() {}
