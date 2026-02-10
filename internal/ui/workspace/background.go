package workspace

import (
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// Кастомный виджет для фона с масштабированием и обрезкой
type ScaledBackground struct {
	widget.BaseWidget
	image      *canvas.Image
	overlay    *canvas.Rectangle
	blackRect  *canvas.Rectangle
	blackRect2 *canvas.Rectangle
	imagePath  string
}

func NewScaledBackground(imagePath string) *ScaledBackground {
	sb := &ScaledBackground{
		imagePath: imagePath,
	}
	sb.ExtendBaseWidget(sb)
	return sb
}

func (sb *ScaledBackground) CreateRenderer() fyne.WidgetRenderer {
	if sb.image == nil {
		sb.image = canvas.NewImageFromFile(sb.imagePath)
		sb.image.FillMode = canvas.ImageFillContain
		sb.overlay = canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 128})   // 50% затемнение
		sb.blackRect = canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 255})  // Черный прямоугольник
		sb.blackRect2 = canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 255}) // Второй черный прямоугольник
	}

	return &scaledBackgroundRenderer{
		sb:      sb,
		objects: []fyne.CanvasObject{sb.image, sb.overlay, sb.blackRect, sb.blackRect2},
	}
}

type scaledBackgroundRenderer struct {
	sb      *ScaledBackground
	objects []fyne.CanvasObject
}

func (r *scaledBackgroundRenderer) Layout(size fyne.Size) {
	// Получаем оригинальный размер изображения
	imgSize := r.sb.image.Size()

	if imgSize.Width == 0 || imgSize.Height == 0 {
		// Если размер неизвестен, используем FillContain
		r.sb.image.FillMode = canvas.ImageFillContain
		r.sb.image.Resize(size)
	} else {
		// Рассчитываем масштаб для заполнения контейнера с сохранением пропорций
		widthRatio := float64(size.Width) / float64(imgSize.Width)
		heightRatio := float64(size.Height) / float64(imgSize.Height)

		// Используем больший масштаб для заполнения (Fill)
		scale := math.Max(widthRatio, heightRatio)

		newWidth := float32(float64(imgSize.Width) * scale)
		newHeight := float32(float64(imgSize.Height) * scale)

		r.sb.image.FillMode = canvas.ImageFillStretch
		r.sb.image.Resize(fyne.NewSize(newWidth, newHeight))

		// Центрируем изображение
		x := (size.Width - newWidth) / 2
		y := (size.Height - newHeight) / 2
		r.sb.image.Move(fyne.NewPos(x, y))
	}

	// Оверлей занимает весь контейнер
	r.sb.overlay.Resize(fyne.NewSize(size.Width, size.Height+20))
	r.sb.overlay.Move(fyne.NewPos(0, -20))

	// Размещаем черный прямоугольник высотой 200 на всю ширину окна над оверлеем
	r.sb.blackRect.Resize(fyne.NewSize(1110, 200))
	r.sb.blackRect.Move(fyne.NewPos(-150, -200)) // Позиционируем в верхней части окна

	r.sb.blackRect2.Resize(fyne.NewSize(5, 600))
	r.sb.blackRect2.Move(fyne.NewPos(-150, 0))
}

func (r *scaledBackgroundRenderer) MinSize() fyne.Size {
	return fyne.NewSize(100, 100)
}

func (r *scaledBackgroundRenderer) Refresh() {
	r.sb.image.Refresh()
	r.sb.overlay.Refresh()
	r.sb.blackRect.Refresh()
}

func (r *scaledBackgroundRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *scaledBackgroundRenderer) Destroy() {}
