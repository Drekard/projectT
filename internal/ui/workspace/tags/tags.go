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

// favoritesService - глобальный экземпляр сервиса избранного
var favoritesService = favorites.NewService()

// tagsService - глобальный экземпляр сервиса тегов
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
	// Получаем все теги из базы данных
	var err error
	t.tags, err = tagsService.GetAllTags(context.Background())
	if err != nil {
		return container.NewVBox(
			widget.NewLabel("Ошибка загрузки тегов: " + err.Error()),
		)
	}

	// Создаем поле поиска
	t.searchBar = widget.NewEntry()
	t.searchBar.SetPlaceHolder("Поиск по тегам...")
	t.searchBar.OnChanged = func(text string) {
		t.filterTags(text)
	}
	searchContainer := container.NewGridWithColumns(2, t.searchBar)

	// Создаем таблицу
	t.table = t.createTable()

	// Создаем контейнер с поиском и таблицей
	return container.NewBorder(
		searchContainer,
		nil, nil, nil,
		t.table,
	)
}

func (t *UI) createTable() *widget.Table {
	// Создаем таблицу с двумя разными типами ячеек
	table := widget.NewTable(
		func() (int, int) {
			rows := len(t.tags)
			return rows, 6
		},
		func() fyne.CanvasObject {
			// Создаем базовый контейнер, который может содержать разные типы объектов
			return container.New(layout.NewHBoxLayout(), widget.NewLabel("placeholder"))
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			if id.Row >= len(t.tags) {
				return
			}

			tag := t.tags[id.Row]
			cellContainer := cell.(*fyne.Container)

			// Очищаем контейнер
			cellContainer.Objects = nil

			switch id.Col {
			case 0: // ID
				cellContainer.Add(widget.NewLabel(fmt.Sprintf("%d", tag.ID)))
			case 1: // Цвет
				circle := canvas.NewRectangle(parseHexColor(tag.Color))
				circle.CornerRadius = 17
				circle.SetMinSize(fyne.NewSize(35, 35))
				circle.StrokeColor = parseHexColor(tag.Color)

				// Оборачиваем круг в контейнер с кнопкой для обработки кликов
				clickBtn := widget.NewButton("", func() {
					t.changeTagColor(tag.ID)
				})
				clickBtn.Importance = widget.LowImportance
				clickBtn.Resize(fyne.NewSize(20, 20))
				clickBtn.Hide() // Скрываем кнопку, но она остается кликабельной

				// Создаем контейнер, в котором круг и кнопка будут находиться в одном месте
				// Используем container.New с StackLayout из пакета container
				stackContainer := container.New(layout.NewStackLayout(), circle, clickBtn)
				cellContainer.Add(stackContainer)
			case 2: // Имя
				cellContainer.Add(widget.NewLabel(tag.Name))
			case 3: // Количество
				cellContainer.Add(widget.NewLabel(fmt.Sprintf("%d", tag.ItemCount)))
			case 4: // Описание
				desc := tag.Description
				if desc == "" {
					desc = "— описание отсутствует —"
				}
				cellContainer.Add(widget.NewLabel(desc))
			case 5: // Кнопки действий
				editBtn := widget.NewButton("✏️", func() { t.editTag(tag.ID) })
				editBtn.Importance = widget.LowImportance

				deleteBtn := widget.NewButton("🗑", func() { t.deleteTag(tag.ID) })
				deleteBtn.Importance = widget.LowImportance

				// Проверяем, является ли тег избранным
				isFavorite, err := favoritesService.IsFavorite("tag", tag.ID)
				if err != nil {
					isFavorite = false
				}

				var favBtn *widget.Button
				if isFavorite {
					favBtn = widget.NewButton("✨", func() {
						err := favoritesService.RemoveFromFavorites("tag", tag.ID)
						if err != nil {
							return
						}

						// Обновляем таблицу для отражения нового состояния
						t.Refresh()
					})
				} else {
					favBtn = widget.NewButton("⭐️", func() {
						err := favoritesService.AddToFavorites("tag", tag.ID)
						if err != nil {
							return
						}

						// Обновляем таблицу для отражения нового состояния
						t.Refresh()
					})
				}
				favBtn.Importance = widget.LowImportance

				cellContainer.Add(favBtn)
				cellContainer.Add(editBtn)
				cellContainer.Add(deleteBtn)
			}
		},
	)

	// Устанавливаем ширину столбцов
	table.SetColumnWidth(0, 50)  // ID
	table.SetColumnWidth(1, 50)  // Цвет
	table.SetColumnWidth(2, 200) // Имя
	table.SetColumnWidth(3, 100) // Количество
	table.SetColumnWidth(4, 250) // Описание
	table.SetColumnWidth(5, 100) // Действия

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
		// В реальном приложении здесь должна быть обработка ошибок
		return
	}

	t.tags = filtered
	t.table.Refresh()
}

func (t *UI) editTag(tagID int) {
	// Получаем информацию о теге для редактирования
	tag, err := tagsService.GetTagByID(context.Background(), tagID)
	if err != nil {
		// В случае ошибки можно показать сообщение пользователю
		return
	}

	// Создаем диалоговое окно для редактирования тега
	w := fyne.CurrentApp().Driver().AllWindows()[0]
	var dialog *widget.PopUp

	// Поля для редактирования
	nameEntry := widget.NewEntry()
	nameEntry.SetText(tag.Name)
	descEntry := widget.NewEntry()
	descEntry.SetText(tag.Description)

	// Поле для редактирования цвета
	colorEntry := widget.NewEntry()
	colorEntry.SetText(tag.Color)

	content := container.NewVBox(
		widget.NewLabel("Редактирование тега"),
		widget.NewLabel("Название:"),
		nameEntry,
		widget.NewLabel("Описание:"),
		descEntry,
		widget.NewLabel("Цвет (в формате HEX, например #FF0000):"),
		colorEntry,
		container.NewHBox(
			widget.NewButton("Отмена", func() {
				dialog.Hide()
			}),
			widget.NewButton("Сохранить", func() {
				// Обновляем тег в базе данных
				tag.Name = nameEntry.Text
				tag.Description = descEntry.Text
				tag.Color = colorEntry.Text

				// Обновляем тег в базе данных через UpdateTag
				err := tagsService.UpdateTag(context.Background(), tag)
				if err != nil {
					// Обработка ошибки
					dialog.Hide()
					return
				}

				// Обновляем список тегов
				t.filterTags(t.searchBar.Text)
				dialog.Hide()
			}),
		),
	)

	dialog = widget.NewPopUp(content, w.Canvas())
	dialog.Show()
}

func (t *UI) deleteTag(tagID int) {
	// Подтверждение удаления
	w := fyne.CurrentApp().Driver().AllWindows()[0]
	var dialog *widget.PopUp
	content := container.NewVBox(
		widget.NewLabel("Удаление тега"),
		widget.NewLabel("Вы уверены, что хотите удалить этот тег?"),
		container.NewHBox(
			widget.NewButton("Отмена", func() {
				dialog.Hide()
			}),
			widget.NewButton("Удалить", func() {
				err := tagsService.DeleteTag(context.Background(), tagID)
				if err != nil {
					// Обработка ошибки
					dialog.Hide()
					return
				}

				// Обновляем список тегов
				t.filterTags(t.searchBar.Text)
				dialog.Hide()
			}),
		),
	)
	dialog = widget.NewPopUp(content, w.Canvas())
	dialog.Show()
}

// parseHexColor - остаётся без изменений
func parseHexColor(hex string) color.RGBA {
	if len(hex) == 0 {
		return color.RGBA{R: 255, G: 187, B: 0, A: 255}
	}

	hex = hex[1:] // Убираем #

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

// Refresh обновляет данные тегов
func (t *UI) Refresh() {
	// Загружаем свежие данные из базы данных
	var err error
	tags, err := tagsService.GetAllTags(context.Background())
	if err != nil {
		// В случае ошибки можно обновить с пустым списком или показать сообщение об ошибке
		tags = []*models.Tag{}
	}

	// Обновляем внутренний список тегов
	t.tags = tags

	// Обновляем таблицу
	t.table.Refresh()
}

// changeTagColor открывает диалог для изменения цвета тега и обновляет цвет тега в базе данных
func (t *UI) changeTagColor(tagID int) {
	// Получаем информацию о теге
	tag, err := tagsService.GetTagByID(context.Background(), tagID)
	if err != nil {
		return
	}

	// Создаем диалог для ввода нового цвета
	w := fyne.CurrentApp().Driver().AllWindows()[0]

	// Объявляем переменную для диалога до её использования
	var popUp *widget.PopUp

	// Поле ввода для цвета в формате HEX
	colorInput := widget.NewEntry()
	colorInput.SetText(tag.Color)

	// Контейнер для диалога
	content := container.NewVBox(
		widget.NewLabel("Введите новый цвет в формате HEX (например, #FF0000):"),
		colorInput,
		container.NewHBox(
			widget.NewButton("Отмена", func() {
				// Закрываем диалог
				popUp.Hide()
			}),
			widget.NewButton("Сохранить", func() {
				newColor := colorInput.Text
				// Проверяем формат цвета (упрощенная проверка)
				if len(newColor) >= 4 && len(newColor) <= 7 && newColor[0] == '#' {
					// Обновляем цвет в базе данных через UpdateTag
					tagToUpdate, err := tagsService.GetTagByID(context.Background(), tagID)
					if err != nil {
						return
					}
					tagToUpdate.Color = newColor
					err = tagsService.UpdateTag(context.Background(), tagToUpdate)
					if err != nil {
						return
					}

					// Обновляем список тегов
					t.Refresh()
				} else {
					// Показываем сообщение об ошибке
					var errorDlg *widget.PopUp
					errorDlg = widget.NewModalPopUp(
						container.NewVBox(
							widget.NewLabel("Неверный формат цвета. Используйте формат #RRGGBB"),
							widget.NewButton("OK", func() {
								errorDlg.Hide()
							}),
						),
						w.Canvas(),
					)
					errorDlg.Show()
				}

				// Закрываем диалог
				popUp.Hide()
			}),
		),
	)

	// Создаем и показываем диалог
	popUp = widget.NewModalPopUp(content, w.Canvas())
	popUp.Show()
}
