package tags

import (
	"context"
	"fmt"
	"image/color"
	"projectT/internal/services"
	"projectT/internal/services/favorites"
	"projectT/internal/storage/database/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// favoritesService - global favorite service instance
var favoritesService = favorites.NewService()

// tagsService - global tags service instance
var tagsService = services.NewTagsService()

type UI struct {
	content   fyne.CanvasObject
	table     *widget.Table
	tags      []*models.Tag
	searchBar *widget.Entry
}

func New() *UI {
	ui := &UI{}
	ui.content = ui.createView()
	return ui
}

func (t *UI) createView() fyne.CanvasObject {
	// Get all tags from database
	var err error
	t.tags, err = tagsService.GetAllTags(context.Background())
	if err != nil {
		return container.NewVBox(
			widget.NewLabel("Error loading tags: " + err.Error()),
		)
	}

	// Create search field
	t.searchBar = widget.NewEntry()
	t.searchBar.SetPlaceHolder("Search tags...")
	t.searchBar.OnChanged = func(text string) {
		t.filterTags(text)
	}
	searchContainer := container.NewGridWithColumns(2, t.searchBar)

	// Create table
	t.table = t.createTable()

	// Create container with search and table
	return container.NewBorder(
		searchContainer,
		nil, nil, nil,
		t.table,
	)
}

func (t *UI) createTable() *widget.Table {
	// Create table with two different cell types
	table := widget.NewTable(
		func() (int, int) {
			rows := len(t.tags)
			return rows, 6
		},
		func() fyne.CanvasObject {
			// Create base container that can contain different object types
			return container.New(layout.NewHBoxLayout(), widget.NewLabel("placeholder"))
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			if id.Row >= len(t.tags) {
				return
			}

			tag := t.tags[id.Row]
			cellContainer := cell.(*fyne.Container)

			// Clear container
			cellContainer.Objects = nil

			switch id.Col {
			case 0: // ID
				cellContainer.Add(widget.NewLabel(fmt.Sprintf("%d", tag.ID)))
			case 1: // Color
				circle := canvas.NewRectangle(parseHexColor(tag.Color))
				circle.CornerRadius = 17
				circle.SetMinSize(fyne.NewSize(35, 35))
				circle.StrokeColor = parseHexColor(tag.Color)

				// Wrap circle in container with button for click handling
				clickBtn := widget.NewButton("", func() {
					t.changeTagColor(tag.ID)
				})
				clickBtn.Importance = widget.LowImportance
				clickBtn.Resize(fyne.NewSize(20, 20))
				clickBtn.Hide() // Hide button but it remains clickable

				// Create container where circle and button are in the same place
				// Use container.New with StackLayout from container package
				stackContainer := container.New(layout.NewStackLayout(), circle, clickBtn)
				cellContainer.Add(stackContainer)
			case 2: // Name
				cellContainer.Add(widget.NewLabel(tag.Name))
			case 3: // Count
				cellContainer.Add(widget.NewLabel(fmt.Sprintf("%d", tag.ItemCount)))
			case 4: // Description
				desc := tag.Description
				if desc == "" {
					desc = "— no description —"
				}
				cellContainer.Add(widget.NewLabel(desc))
			case 5: // Action buttons
				editBtn := widget.NewButton("✏️", func() { t.editTag(tag.ID) })
				editBtn.Importance = widget.LowImportance

				deleteBtn := widget.NewButton("🗑", func() { t.deleteTag(tag.ID) })
				deleteBtn.Importance = widget.LowImportance

				isFavorite, err := favoritesService.IsFavorite("tag", tag.TagUUID)
				if err != nil {
					isFavorite = false
				}

				var favBtn *widget.Button
				favBtn = widget.NewButton("⭐️", func() {
					if isFavorite {
						err := favoritesService.RemoveFromFavorites("tag", tag.TagUUID)
						if err != nil {
							return
						}
					} else {
						err := favoritesService.AddToFavorites("tag", tag.TagUUID)
						if err != nil {
							return
						}
					}

					newIsFavorite, _ := favoritesService.IsFavorite("tag", tag.TagUUID)
					if newIsFavorite {
						favBtn.SetText("✨")
					} else {
						favBtn.SetText("⭐️")
					}

					t.Refresh()
				})
				favBtn.Importance = widget.LowImportance

				if isFavorite {
					favBtn.SetText("✨")
				}

				cellContainer.Add(favBtn)
				cellContainer.Add(editBtn)
				cellContainer.Add(deleteBtn)
			}
		},
	)

	// Set column widths
	table.SetColumnWidth(0, 50)  // ID
	table.SetColumnWidth(1, 50)  // Color
	table.SetColumnWidth(2, 150) // Name
	table.SetColumnWidth(3, 60)  // Count
	table.SetColumnWidth(4, 200) // Description
	table.SetColumnWidth(5, 100) // Actions

	return table
}

func (t *UI) filterTags(searchText string) {
	var filtered []*models.Tag
	var err error

	if searchText == "" {
		filtered, err = tagsService.GetAllTags(context.Background())
	} else {
		filtered, err = tagsService.SearchTagsByName(context.Background(), searchText)
	}

	if err != nil {
		// In real application, error handling should be here
		return
	}

	t.tags = filtered
	t.table.Refresh()
}

func (t *UI) editTag(tagID int) {
	// Get tag information for editing
	tag, err := tagsService.GetTagByID(context.Background(), tagID)
	if err != nil {
		// In case of error, can show message to user
		return
	}

	// Create dialog for editing tag
	w := fyne.CurrentApp().Driver().AllWindows()[0]
	var dialog *widget.PopUp

	// Fields for editing
	nameEntry := widget.NewEntry()
	nameEntry.SetText(tag.Name)
	descEntry := widget.NewEntry()
	descEntry.SetText(tag.Description)

	// Field for editing color
	colorEntry := widget.NewEntry()
	colorEntry.SetText(tag.Color)

	content := container.NewVBox(
		widget.NewLabel("Edit Tag"),
		widget.NewLabel("Name:"),
		nameEntry,
		widget.NewLabel("Description:"),
		descEntry,
		widget.NewLabel("Color (in HEX format, e.g. #FF0000):"),
		colorEntry,
		container.NewHBox(
			widget.NewButton("Cancel", func() {
				dialog.Hide()
			}),
			widget.NewButton("Save", func() {
				// Update tag in database
				tag.Name = nameEntry.Text
				tag.Description = descEntry.Text
				tag.Color = colorEntry.Text

				// Update tag in database via UpdateTag
				err := tagsService.UpdateTag(context.Background(), tag)
				if err != nil {
					// Error handling
					dialog.Hide()
					return
				}

				// Refresh tag list
				t.filterTags(t.searchBar.Text)
				dialog.Hide()
			}),
		),
	)

	dialog = widget.NewPopUp(content, w.Canvas())
	dialog.Show()
}

func (t *UI) deleteTag(tagID int) {
	// Confirmation dialog
	w := fyne.CurrentApp().Driver().AllWindows()[0]
	var dialog *widget.PopUp
	content := container.NewVBox(
		widget.NewLabel("Delete Tag"),
		widget.NewLabel("Are you sure you want to delete this tag?"),
		container.NewHBox(
			widget.NewButton("Cancel", func() {
				dialog.Hide()
			}),
			widget.NewButton("Delete", func() {
				err := tagsService.DeleteTag(context.Background(), tagID)
				if err != nil {
					// Error handling
					dialog.Hide()
					return
				}

				// Refresh tag list
				t.filterTags(t.searchBar.Text)
				dialog.Hide()
			}),
		),
	)
	dialog = widget.NewPopUp(content, w.Canvas())
	dialog.Show()
}

// parseHexColor - remains unchanged
func parseHexColor(hex string) color.RGBA {
	if len(hex) == 0 {
		return color.RGBA{R: 255, G: 187, B: 0, A: 255}
	}

	hex = hex[1:] // Remove #

	var r, g, b uint8
	if len(hex) == 6 {
		r = uint8((parseHexChar(hex[0]) << 4) + parseHexChar(hex[1]))
		g = uint8((parseHexChar(hex[2]) << 4) + parseHexChar(hex[3]))
		b = uint8((parseHexChar(hex[4]) << 4) + parseHexChar(hex[5]))
	} else if len(hex) == 3 {
		r = uint8(parseHexChar(hex[0]) * 17)
		g = uint8(parseHexChar(hex[1]) * 17)
		b = uint8(parseHexChar(hex[2]) * 17)
	}

	return color.RGBA{R: r, G: g, B: b, A: 255}
}

func parseHexChar(c byte) uint8 {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

func (t *UI) GetContent() fyne.CanvasObject {
	return t.content
}

// Refresh updates tags data
func (t *UI) Refresh() {
	// Load fresh data from database
	var err error
	tags, err := tagsService.GetAllTags(context.Background())
	if err != nil {
		// In case of error, can update with empty list or show error message
		tags = []*models.Tag{}
	}

	// Update internal tags list
	t.tags = tags

	// Update table
	t.table.Refresh()
}

// changeTagColor opens a dialog to change tag color and updates the tag color in the database
func (t *UI) changeTagColor(tagID int) {
	// Get tag information
	tag, err := tagsService.GetTagByID(context.Background(), tagID)
	if err != nil {
		return
	}

	// Create a dialog for entering new color
	w := fyne.CurrentApp().Driver().AllWindows()[0]

	// Declare variable for dialog before using it
	var popUp *widget.PopUp

	// Input field for color in HEX format
	colorInput := widget.NewEntry()
	colorInput.SetText(tag.Color)

	// Container for dialog
	content := container.NewVBox(
		widget.NewLabel("Enter new color in HEX format (e.g., #FF0000):"),
		colorInput,
		container.NewHBox(
			widget.NewButton("Cancel", func() {
				// Close dialog
				popUp.Hide()
			}),
			widget.NewButton("Save", func() {
				newColor := colorInput.Text
				// Check color format (simplified check)
				if len(newColor) >= 4 && len(newColor) <= 7 && newColor[0] == '#' {
					// Update color in database via UpdateTag
					tagToUpdate, err := tagsService.GetTagByID(context.Background(), tagID)
					if err != nil {
						return
					}
					tagToUpdate.Color = newColor
					err = tagsService.UpdateTag(context.Background(), tagToUpdate)
					if err != nil {
						return
					}

					// Refresh tag list
					t.Refresh()
				} else {
					// Show error message
					var errorDlg *widget.PopUp
					errorDlg = widget.NewModalPopUp(
						container.NewVBox(
							widget.NewLabel("Invalid color format. Use format #RRGGBB"),
							widget.NewButton("OK", func() {
								errorDlg.Hide()
							}),
						),
						w.Canvas(),
					)
					errorDlg.Show()
				}

				// Close dialog
				popUp.Hide()
			}),
		),
	)

	// Create and show dialog
	popUp = widget.NewModalPopUp(content, w.Canvas())
	popUp.Show()
}
