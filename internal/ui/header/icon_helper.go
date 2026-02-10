package header

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

func LoadAppIcon() fyne.CanvasObject {
	img := canvas.NewImageFromFile("./assets/icons/ProjctT.png")
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(32, 32))
	return container.NewCenter(img)
}
