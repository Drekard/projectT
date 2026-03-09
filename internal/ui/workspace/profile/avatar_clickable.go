package profile

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// AvatarClickableImage - виджет изображения аватара с поддержкой двойного клика
type AvatarClickableImage struct {
	widget.BaseWidget
	content        fyne.CanvasObject
	onDoubleTapped func()
}

// NewAvatarClickableImage создает новый виджет аватара с поддержкой двойного клика
func NewAvatarClickableImage(content fyne.CanvasObject, onDoubleTapped func()) *AvatarClickableImage {
	c := &AvatarClickableImage{
		content:        content,
		onDoubleTapped: onDoubleTapped,
	}
	c.ExtendBaseWidget(c)
	return c
}

// CreateRenderer создает рендерер для виджета
func (c *AvatarClickableImage) CreateRenderer() fyne.WidgetRenderer {
	return &avatarClickableImageRenderer{
		card:    c,
		objects: []fyne.CanvasObject{c.content},
	}
}

// DoubleTapped обрабатывает двойной клик
func (c *AvatarClickableImage) DoubleTapped(*fyne.PointEvent) {
	if c.onDoubleTapped != nil {
		c.onDoubleTapped()
	}
}

// avatarClickableImageRenderer рендерер для AvatarClickableImage
type avatarClickableImageRenderer struct {
	card    *AvatarClickableImage
	objects []fyne.CanvasObject
}

func (r *avatarClickableImageRenderer) Layout(size fyne.Size) {
	r.objects[0].Resize(size)
}

func (r *avatarClickableImageRenderer) MinSize() fyne.Size {
	return r.objects[0].MinSize()
}

func (r *avatarClickableImageRenderer) Refresh() {
	r.objects[0].Refresh()
}

func (r *avatarClickableImageRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *avatarClickableImageRenderer) Destroy() {}
