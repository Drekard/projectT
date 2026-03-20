package header

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

func LoadAppIcon() fyne.CanvasObject {
	exePath, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	iconPath := filepath.Join(exePath, "ProjctT.png")
	img := canvas.NewImageFromFile(iconPath)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(32, 32))
	return container.NewCenter(img)
}
