package chats

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// UI represents the chats interface
type UI struct {
	content fyne.CanvasObject
}

// New creates and returns a new chats UI
func New() *UI {
	ui := &UI{}
	ui.content = ui.createViewContent()
	return ui
}

// createViewContent creates and returns the visual representation of the chats UI
func (c *UI) createViewContent() fyne.CanvasObject {
	// Create black background area on the left
	leftPanel := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 255}) // Pure black
	leftPanel.SetMinSize(fyne.NewSize(300, 0)) // Fixed width for the left panel

	// Create separator line
	separator := canvas.NewRectangle(color.RGBA{R: 64, G: 64, B: 64, A: 255}) // Dark gray separator
	separator.SetMinSize(fyne.NewSize(1, 0)) // 1 pixel wide separator

	// Create main content area (currently empty as placeholder)
	mainContent := canvas.NewRectangle(color.RGBA{R: 32, G: 32, B: 32, A: 255}) // Slightly lighter gray background for contrast

	// Arrange the elements horizontally: left panel + separator + main content
	return container.NewHBox(leftPanel, separator, mainContent)
}

// createView обновляет визуальное представление UI чатов
func (c *UI) createView() {
	c.content = c.createViewContent()
}

// CreateView returns the canvas object for the chats view
func (c *UI) CreateView() fyne.CanvasObject {
	return c.content
}

// GetContent returns the content canvas object
func (c *UI) GetContent() fyne.CanvasObject {
	return c.content
}

// Refresh обновляет содержимое UI
func (c *UI) Refresh() {
	// В текущей реализации нет необходимости в обновлении, но метод предусмотрен для совместимости
	// При необходимости здесь можно обновить содержимое чатов
	c.content = c.createViewContent() // Пересоздаем представление
	if c.content != nil {
		c.content.Refresh()
	}
}
