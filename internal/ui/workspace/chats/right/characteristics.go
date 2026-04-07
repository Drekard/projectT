// Package right contains right panel components (profile)
package right

import (
	"encoding/json"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ContentCharacteristicItem represents a characteristic element
type ContentCharacteristicItem struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Value string `json:"value"`
}

// loadCharacteristics loads characteristics from JSON
func (p *Panel) loadCharacteristics(jsonStr string) {
	if p.characteristicsContainer == nil {
		return
	}

	p.characteristicsContainer.Objects = nil

	var characteristics []ContentCharacteristicItem
	if jsonStr != "" {
		err := json.Unmarshal([]byte(jsonStr), &characteristics)
		if err != nil {
			return
		}
	}

	if len(characteristics) == 0 {
		emptyLabel := widget.NewLabel("No characteristics")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		p.characteristicsContainer.Add(emptyLabel)
	} else {
		for _, item := range characteristics {
			characteristicItem := p.createCharacteristicItem(item.Title, item.Value)
			p.characteristicsContainer.Add(characteristicItem)
		}
	}

	p.characteristicsContainer.Refresh()
}

// createCharacteristicItem creates a characteristic item (name: value on one line)
func (p *Panel) createCharacteristicItem(title, value string) *fyne.Container {
	// Format as "Name: Value"
	text := fmt.Sprintf("%s: %s", title, value)
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	return container.NewVBox(label)
}
