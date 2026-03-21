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

	// Если не найдено рядом с exe, пробуем assets/icons/ (для go run)
	if _, err := os.Stat(iconPath); os.IsNotExist(err) {
		iconPath = filepath.Join("assets", "icons", "ProjctT.png")
	}

	img := canvas.NewImageFromFile(iconPath)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(32, 32))
	return container.NewCenter(img)
}
