package profile

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"projectT/internal/services/background"
	"projectT/internal/storage/database/queries"
	appTheme "projectT/internal/ui/theme"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (p *UI) CreateView() fyne.CanvasObject {
	return p.content
}

func (p *UI) GetContent() fyne.CanvasObject {
	return p.content
}

// GetAvatarPath returns the current avatar path
func (p *UI) GetAvatarPath() string {
	return p.avatarPath
}

// SetWindow sets the window for UI
func (p *UI) SetWindow(window fyne.Window) {
	p.window = window
}

// SetBackgroundUpdater sets the background updater for changing background color
func (p *UI) SetBackgroundUpdater(updater BackgroundUpdater) {
	p.backgroundUpdater = updater
}

func (p *UI) SetCustomField(index int, title, value string) {
	if index >= 0 && index < len(p.customFields) {
		p.customFields[index].titleLabel.SetText(title + ":")
		p.customFields[index].valueEntry.SetText(value)
	}
}

// AddCharacteristic adds a new characteristic to the interface
func (p *UI) AddCharacteristic() {
	row := &fieldRow{}

	// Assign unique UUID (will be set on load or save)
	row.elementUUID = ""
	p.nextID++ // increment counter for next element

	row.titleLabel = widget.NewLabel(":")
	row.titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	row.titleEntry = widget.NewEntry()
	row.titleEntry.PlaceHolder = "Name"

	// Add text change handler for auto-save
	row.titleEntry.OnChanged = func(text string) {
		p.scheduleAutoSave(row)
	}

	entryWrapper := canvas.NewRectangle(color.Transparent)
	entryWrapper.SetMinSize(fyne.NewSize(140, 40)) // 165 + button width
	entryContainer := container.NewStack(entryWrapper, row.titleEntry)

	row.valueEntry = widget.NewEntry()
	row.valueEntry.PlaceHolder = "Value"

	// Add text change handler for auto-save
	row.valueEntry.OnChanged = func(text string) {
		p.scheduleAutoSave(row)
	}

	// Delete button
	row.removeButton = widget.NewButton("❌", func() {
		p.RemoveCharacteristic(row)
	})
	row.removeButton.Importance = widget.LowImportance

	// Create container with stretchable elements
	row.container = container.NewBorder(
		nil,
		nil,
		container.NewHBox(entryContainer, row.titleLabel),
		row.removeButton,
		row.valueEntry,
	)

	p.characteristicsContainer.Add(row.container)

	// Add to field list for later saving
	p.customFields = append(p.customFields, row)
}

// scheduleAutoSave schedules field auto-save after 2 seconds
func (p *UI) scheduleAutoSave(row *fieldRow) {
	// Cancel previous timer if it exists
	if row.timer != nil {
		row.timer.Stop()
	}

	// Create new 2-second timer
	row.timer = time.AfterFunc(2*time.Second, func() {
		p.autoSaveField(row)
	})
}

// scheduleProfileAutoSave schedules profile auto-save (name and title) after 2 seconds
func (p *UI) scheduleProfileAutoSave() {
	// Cancel previous timers if they exist
	if p.userNameTimer != nil {
		p.userNameTimer.Stop()
	}
	if p.userTitleTimer != nil {
		p.userTitleTimer.Stop()
	}

	// Create new 2-second timer
	p.userNameTimer = time.AfterFunc(2*time.Second, func() {
		p.autoSaveProfile()
	})
	p.userTitleTimer = p.userNameTimer
}

// autoSaveProfile saves profile name and status to database
func (p *UI) autoSaveProfile() {
	p.saveCharacteristicsToDB()
}

// autoSaveField saves field to database
func (p *UI) autoSaveField(row *fieldRow) {
	p.saveCharacteristicsToDB()
}

// saveCharacteristicsToDB saves all characteristics to database
func (p *UI) saveCharacteristicsToDB() {
	// Get current profile
	profile, err := queries.GetLocalProfile()
	if err != nil {
		return
	}

	// Update main profile fields
	profile.Username = p.userNameEntry.Text
	profile.Title = p.userTitleEntry.Text

	// Update characteristics in JSON format
	characteristicsJSON, err := p.SaveCharacteristicsToJSON()
	if err != nil {
		return
	}
	profile.ContentChar = characteristicsJSON

	// Save changes to database
	err = queries.UpdateLocalProfile(profile)
	if err != nil {
		return
	}
}

// RemoveCharacteristic removes a characteristic from the interface
func (p *UI) RemoveCharacteristic(row *fieldRow) {

	// Cancel timer if it exists
	if row.timer != nil {
		row.timer.Stop()
	}

	p.characteristicsContainer.Remove(row.container)

	// Remove from list if needed to save references
	for i, r := range p.customFields {
		if r == row {
			p.customFields = append(p.customFields[:i], p.customFields[i+1:]...)
			break
		}
	}

	// Save changes to database
	p.saveCharacteristicsToDB()
}

// LoadCharacteristicsFromJSON loads characteristics from JSON string
func (p *UI) LoadCharacteristicsFromJSON(jsonStr string) {
	var characteristics []ContentCharacteristicItem
	if jsonStr != "" {
		err := json.Unmarshal([]byte(jsonStr), &characteristics)
		if err != nil {
			// On error, just exit
			return
		}
	}

	// Clear current container
	p.characteristicsContainer.Objects = nil
	p.characteristicsContainer.Refresh()

	// Add characteristics to interface
	for _, item := range characteristics {
		p.AddCharacteristic()
		// Set values for the last added element
		if len(p.customFields) > 0 {
			lastRow := p.customFields[len(p.customFields)-1]
			lastRow.elementUUID = item.ElementUUID // ✅ Save ElementUUID
			lastRow.titleEntry.SetText(item.Title)
			lastRow.valueEntry.SetText(item.Value)
			// Update title label too for view mode
			lastRow.titleLabel.SetText(":")
		}
	}
}

// SaveCharacteristicsToJSON saves characteristics to JSON string
func (p *UI) SaveCharacteristicsToJSON() (string, error) {
	var characteristics []ContentCharacteristicItem

	// Collect all characteristics from interface
	for _, row := range p.customFields {
		// Get title from input field
		title := row.titleEntry.Text

		// Get value from input field
		value := row.valueEntry.Text

		characteristics = append(characteristics, ContentCharacteristicItem{
			ElementUUID: row.elementUUID, // ✅ Save ElementUUID
			Title:       title,
			Value:       value,
		})
	}

	jsonData, err := json.Marshal(characteristics)
	if err != nil {
		return "", err
	}

	jsonStr := string(jsonData)

	return jsonStr, nil
}

// showBackgroundDialog shows a dialog for managing background image
func (p *UI) showBackgroundDialog() {
	p.showImageDialog("Background", "storage/files/background", "No saved backgrounds", "Upload background", "Delete background", func(imagePath string) error {
		backgroundService := background.NewService()
		err := backgroundService.SetBackground(imagePath)
		if err != nil {
			return fmt.Errorf("error setting background: %v", err)
		}
		p.backgroundPath = imagePath
		return nil
	}, func() {
		backgroundService := background.NewService()
		_ = backgroundService.ClearBackground() //nolint:errcheck
		p.backgroundPath = ""
	})
}

// showAvatarDialog shows a dialog for managing avatar
func (p *UI) showAvatarDialog() {
	p.showImageDialog("Avatar", "storage/files/avatars/local", "No saved avatars", "Upload avatar", "", func(imagePath string) error {
		p.avatarPath = imagePath
		return nil
	}, func() {
		p.avatarPath = ""
	})
}

// showImageDialog shows an image selection dialog
func (p *UI) showImageDialog(
	title string,
	assetsDir string,
	noImagesLabel string,
	loadButtonLabel string,
	deleteButtonLabel string,
	onSelect func(imagePath string) error,
	onDelete func(),
) {
	// Create container for thumbnails
	thumbnailsContainer := container.NewVBox()

	// Get list of files from directory
	files, err := os.ReadDir(assetsDir)
	if err != nil {
		// If directory doesn't exist or an error occurs, create empty container
		thumbnailsContainer.Add(widget.NewLabel(noImagesLabel))
	} else {
		// Filter only image files
		imageExtensions := map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
			".gif":  true,
			".bmp":  true,
		}

		hasImages := false
		for _, file := range files {
			if !file.IsDir() {
				ext := strings.ToLower(filepath.Ext(file.Name()))
				if imageExtensions[ext] {
					imagePath := filepath.Join(assetsDir, file.Name())

					// Create image thumbnail
					imageThumb := canvas.NewImageFromFile(imagePath)
					imageThumb.FillMode = canvas.ImageFillContain
					imageThumb.SetMinSize(fyne.NewSize(100, 100))

					// Create container for thumbnail with file name
					fileLabel := widget.NewLabel(file.Name())
					fileLabel.Alignment = fyne.TextAlignCenter

					thumbContainer := container.NewVBox(imageThumb, fileLabel)

					// Add double-click selection capability
					clickableThumb := NewThumbnailClickable(thumbContainer, func() {
						err := onSelect(imagePath)
						if err != nil {
							dialog.ShowError(err, p.window)
							return
						}

						// Save to DB
						p.saveToDatabase()

						// Update profile UI
						p.createView()

						// Close dialog and show notification
						dialog.ShowInformation("Success", fmt.Sprintf("%s successfully set", strings.ToLower(title)), p.window)
					})

					thumbnailsContainer.Add(clickableThumb)
					hasImages = true
				}
			}
		}

		if !hasImages {
			thumbnailsContainer.Add(widget.NewLabel(noImagesLabel))
		}
	}

	// Create buttons
	loadButton := widget.NewButton(loadButtonLabel, func() {
		if title == "Background" {
			p.selectBackgroundImage()
		} else {
			p.selectAvatarImage()
		}
	})

	// Create container for buttons
	var buttonsContainer fyne.CanvasObject
	if deleteButtonLabel != "" {
		deleteButton := widget.NewButton(deleteButtonLabel, func() {
			onDelete()
			p.saveToDatabase()

			// Update profile UI
			p.createView()
		})
		buttonsContainer = container.NewHBox(loadButton, deleteButton)
	} else {
		buttonsContainer = container.NewHBox(loadButton)
	}

	// Create main dialog container
	dialogContent := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Select %s:", strings.ToLower(title))),
		thumbnailsContainer,
		container.NewCenter(buttonsContainer),
	)

	// Show dialog
	dialog.ShowCustom(title, "Close", dialogContent, p.window)
}

// showThemeDialog shows a dialog for selecting application theme and background color
func (p *UI) showThemeDialog() {
	themeNames := []string{"Dark", "Light", "Blue", "Green", "Purple"}
	themes := []appTheme.AppTheme{
		appTheme.DarkTheme,
		appTheme.LightTheme,
		appTheme.BlueTheme,
		appTheme.GreenTheme,
		appTheme.PurpleTheme,
	}

	currentTheme := appTheme.GetTheme()
	currentThemeName := "Purple"
	for i, t := range themes {
		if currentTheme == t {
			currentThemeName = themeNames[i]
			break
		}
	}

	themeRadio := widget.NewRadioGroup(themeNames, func(selected string) {
		for i, name := range themeNames {
			if name == selected {
				appTheme.SetTheme(themes[i])
				if app := fyne.CurrentApp(); app != nil {
					app.Settings().SetTheme(appTheme.GetFyneTheme())
				}
				// Сохраняем тему в конфиг
				if p.config != nil {
					p.config.UISettings.Theme = selected
					if p.onSave != nil {
						p.onSave()
					}
				}
				break
			}
		}
	})
	themeRadio.Horizontal = false
	themeRadio.SetSelected(currentThemeName)

	bgRadio := widget.NewRadioGroup([]string{"Black", "White"}, func(selected string) {
		if p.backgroundUpdater != nil {
			switch selected {
			case "Black":
				p.backgroundUpdater.SetBackgroundColor(color.RGBA{R: 0, G: 0, B: 0, A: 255})
			case "White":
				p.backgroundUpdater.SetBackgroundColor(color.RGBA{R: 255, G: 255, B: 255, A: 255})
			}
		}
	})
	bgRadio.Horizontal = false
	bgRadio.SetSelected("Black")

	leftCol := container.NewVBox(
		widget.NewLabel("Theme"),
		themeRadio,
	)

	rightCol := container.NewVBox(
		widget.NewLabel("Background"),
		bgRadio,
	)

	dialogContent := container.NewHBox(leftCol, rightCol)

	dialog.ShowCustom("Settings", "Close", dialogContent, p.window)
}
