package hover_preview

import (
	"context"
	"fmt"
	"image/color"
	"projectT/internal/services"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/edit_item"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// SearchHandler определяет интерфейс для обработки поисковых запросов
type SearchHandler interface {
	SetSearchQuery(query string)
}

// favoritesService - глобальный экземпляр сервиса избранного
var favoritesService = services.NewFavoritesService()

// globalSearchEntry глобальная ссылка на поисковую строку
var globalSearchEntry *widget.Entry

// MenuManager менеджер меню действий
type MenuManager struct {
	searchEntry *widget.Entry
}

// SetGlobalSearchEntry устанавливает глобальную ссылку на поисковую строку
func SetGlobalSearchEntry(entry *widget.Entry) {
	globalSearchEntry = entry
}

// NewMenuManager создает новый менеджер меню
func NewMenuManager() *MenuManager {
	return &MenuManager{}
}

// SetSearchEntry устанавливает ссылку на поисковую строку
func (mm *MenuManager) SetSearchEntry(entry *widget.Entry) {
	mm.searchEntry = entry
}

// ShowSimpleMenu показывает простое меню действий
func (mm *MenuManager) ShowSimpleMenu(item *models.Item, cont fyne.CanvasObject, onClose func()) {
	window := fyne.CurrentApp().Driver().CanvasForObject(cont)
	if window == nil {
		return
	}

	// Создаем переменную для попапа, чтобы была возможность его закрыть из обработчика кнопки
	var popup *widget.PopUp

	var children []fyne.CanvasObject

	children = append(children,
		widget.NewRichTextFromMarkdown(getTitleForItem(item)),
	)

	if item.Type == models.ItemTypeElement && item.ContentMeta != "" && item.Description != "" {
		children = append(children, widget.NewLabel(getDescriptionForItem(item)))
	}

	children = append(children,
		getTagsContainer(item, mm),
		widget.NewLabel("Создан: "+item.CreatedAt.Format("02.01.2006 15:04")),
		widget.NewLabel("Изменен: "+item.UpdatedAt.Format("02.01.2006 15:04")),
		container.NewBorder(
			nil, nil, nil,
			func() fyne.CanvasObject {
				buttons := []fyne.CanvasObject{
					widget.NewButton("✏️ Редактировать", func() {
						appWindows := fyne.CurrentApp().Driver().AllWindows()
						if len(appWindows) > 0 {
							edit_item.ShowCreateItemModalForEdit(appWindows[0], item.ID)
						}
					}),
					widget.NewButton("🗑 Удалить", func() {
						appWindow := fyne.CurrentApp().Driver().AllWindows()[0]
						dialog.ShowConfirm("Подтверждение удаления",
							fmt.Sprintf("Вы уверены, что хотите удалить элемент \"%s\"?", item.Title),
							func(confirmed bool) {
								if confirmed {
									if err := mm.deleteItem(item); err != nil {
										dialog.ShowError(fmt.Errorf("Ошибка при удалении элемента: %v", err), appWindow)
									} else {
										popup.Hide()
										if onClose != nil {
											onClose()
										}
									}
								}
							}, appWindow)
					}),
				}

				// Добавляем кнопку избранного только для папок
				if item.Type == models.ItemTypeFolder {
					isFavorite, err := favoritesService.IsFavorite("folder", item.ID)
					if err != nil {
						isFavorite = false
					}

					// Создаем кнопку избранного с правильным начальным состоянием
					var favButton *widget.Button

					// Для корректной работы с замыканиями создаем функцию вне блока
					// чтобы избежать проблем с областью видимости
					var createFavHandler func(currentState bool) func()

					// Определяем функцию обработчика
					createFavHandler = func(currentState bool) func() {
						if currentState {
							// Если сейчас в избранном - делаем обработчик для удаления
							return func() {
								err := favoritesService.RemoveFromFavorites("folder", item.ID)
								if err != nil {
									return
								}
								// Обновляем текст кнопки
								favButton.SetText("⭐️")
								// Устанавливаем новый обработчик для следующего клика
								favButton.OnTapped = createFavHandler(false)
							}
						} else {
							// Если сейчас не в избранном - делаем обработчик для добавления
							return func() {
								err := favoritesService.AddToFavorites("folder", item.ID)
								if err != nil {
									return
								}
								// Обновляем текст кнопки
								favButton.SetText("✨")
								// Устанавливаем новый обработчик для следующего клика
								favButton.OnTapped = createFavHandler(true)
							}
						}
					}

					// Создаем кнопку с правильным начальным текстом и обработчиком
					if isFavorite {
						favButton = widget.NewButton("✨", createFavHandler(true))
					} else {
						favButton = widget.NewButton("⭐️", createFavHandler(false))
					}

					// Вставляем кнопку избранного первой в список кнопок
					buttons = append([]fyne.CanvasObject{favButton}, buttons...)
				}

				// Добавляем кнопку закрепления для всех типов элементов
				isPinned, err := queries.IsItemPinned(item.ID)
				if err != nil {
					isPinned = false
				}

				// Создаем кнопку закрепления с правильным начальным состоянием
				var pinButton *widget.Button

				// Для корректной работы с замыканиями создаем функцию вне блока
				// чтобы избежать проблем с областью видимости
				var createPinHandler func(currentState bool) func()

				// Определяем функцию обработчика
				createPinHandler = func(currentState bool) func() {
					if currentState {
						// Если сейчас закреплено - делаем обработчик для открепления
						return func() {
							err := queries.UnpinItem(item.ID)
							if err != nil {
								return
							}
							// Обновляем текст кнопки
							pinButton.SetText("📌")
							// Устанавливаем новый обработчик для следующего клика
							pinButton.OnTapped = createPinHandler(false)
						}
					} else {
						// Если сейчас не закреплено - делаем обработчик для закрепления
						return func() {
							err := queries.PinItem(item.ID)
							if err != nil {
								return
							}
							// Обновляем текст кнопки
							pinButton.SetText("✅📌")
							// Устанавливаем новый обработчик для следующего клика
							pinButton.OnTapped = createPinHandler(true)
						}
					}
				}

				// Создаем кнопку с правильным начальным текстом и обработчиком
				if isPinned {
					pinButton = widget.NewButton("✅📌", createPinHandler(true))
				} else {
					pinButton = widget.NewButton("📌", createPinHandler(false))
				}

				// Вставляем кнопку закрепления перед кнопками редактирования и удаления
				buttons = append([]fyne.CanvasObject{pinButton}, buttons...)

				return container.NewHBox(buttons...)
			}(),
		),
	)

	content := container.NewVBox(children...)

	popup = widget.NewPopUp(content, window)

	// Позиция карточки
	cardPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(cont)

	// Показываем прямо под карточкой
	menuPos := fyne.NewPos(
		cardPos.X,
		cardPos.Y+50,
	)

	// Проверяем, не выходит ли за нижнюю границу окна
	popupSize := popup.MinSize()
	windowSize := window.Size()

	if menuPos.Y+popupSize.Height > windowSize.Height {
		// Если выходит, показываем над карточкой
		menuPos.Y = cardPos.Y - popupSize.Height - 5
	}

	popup.ShowAtPosition(menuPos)

	// Вызываем колбэк при закрытии
	if onClose != nil {
		go func() {
			// Периодически проверяем, закрыт ли попап, чтобы не нагружать CPU
			for popup.Visible() {
				time.Sleep(100 * time.Millisecond) // Ждем 100 мс перед следующей проверкой
			}
			onClose()
		}()
	}
}

// deleteItem удаляет элемент и все вложенные элементы, если это папка
func (mm *MenuManager) deleteItem(item *models.Item) error {
	// Если элемент - папка, удаляем все вложенные элементы рекурсивно
	if item.Type == models.ItemTypeFolder {
		// Получаем все элементы в папке
		items, err := queries.GetItemsByParent(item.ID)
		if err != nil {
			return fmt.Errorf("ошибка получения вложенных элементов: %v", err)
		}

		// Рекурсивно удаляем все вложенные элементы
		for _, childItem := range items {
			if err := mm.deleteItem(childItem); err != nil {
				return fmt.Errorf("ошибка удаления вложенного элемента: %v", err)
			}
		}
	}

	// Удаляем сам элемент
	if err := queries.DeleteItem(item.ID); err != nil {
		return fmt.Errorf("ошибка удаления элемента: %v", err)
	}

	return nil
}

// SetSearchQuery устанавливает поисковый запрос
func (mm *MenuManager) SetSearchQuery(query string) {
	// Сначала проверяем локальную ссылку
	if mm.searchEntry != nil {
		mm.searchEntry.SetText(query)
		return
	}

	// Затем проверяем глобальную ссылку
	if globalSearchEntry != nil {
		globalSearchEntry.SetText(query)
		return
	}

	fmt.Printf("Попытка установить поисковый запрос '%s', но поисковая строка не инициализирована\n", query)
}

// getDescriptionForItem возвращает описание элемента или сообщение, если описание отсутствует
func getDescriptionForItem(item *models.Item) string {
	if item.Description == "" {
		return "--описание отсутствует--"
	}
	return item.Description
}

func getTitleForItem(item *models.Item) string {
	if item.Title == "" {
		return "--заголовок отсутствует--"
	}
	return "**" + item.Title + "**"
}

// getTagsForItem возвращает строку с тегами для элемента
func getTagsForItem(item *models.Item) string {
	tags, err := queries.GetTagsForItem(context.Background(), item.ID)
	if err != nil || len(tags) == 0 {
		return "--теги отсутствуют--"
	}

	var tagNames []string
	for _, tag := range tags {
		tagNames = append(tagNames, tag.Name)
	}

	return strings.Join(tagNames, ", ")
}

// parseHexColor преобразует HEX цвет в RGBA
func parseHexColor(hex string) (color.RGBA, error) {
	// Убираем символ # если он есть
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}

	var r, g, b uint8
	var a uint8 = 255 // по умолчанию непрозрачный

	switch len(hex) {
	case 3: // формат #RGB
		var ir, ig, ib int
		n, err := fmt.Sscanf(hex, "%1x%1x%1x", &ir, &ig, &ib)
		if n != 3 || err != nil {
			return color.RGBA{}, fmt.Errorf("неправильный формат HEX цвета: %s", hex)
		}
		r, g, b = uint8(ir*17), uint8(ig*17), uint8(ib*17) // 17 = 255/15
	case 6: // формат #RRGGBB
		var ir, ig, ib int
		n, err := fmt.Sscanf(hex, "%02x%02x%02x", &ir, &ig, &ib)
		if n != 3 || err != nil {
			return color.RGBA{}, fmt.Errorf("неправильный формат HEX цвета: %s", hex)
		}
		r, g, b = uint8(ir), uint8(ig), uint8(ib)
	default:
		return color.RGBA{}, fmt.Errorf("неправильная длина HEX цвета: %s", hex)
	}

	return color.RGBA{R: r, G: g, B: b, A: a}, nil
}

// getContrastColor возвращает контрастный цвет (черный или белый) в зависимости от фона
func getContrastColor(backgroundColor color.RGBA) color.Color {
	// Вычисляем яркость фона по формуле
	luminance := (0.299*float64(backgroundColor.R) + 0.587*float64(backgroundColor.G) + 0.114*float64(backgroundColor.B)) / 255.0

	if luminance > 0.5 {
		// Светлый фон - используем черный текст
		return color.RGBA{R: 0, G: 0, B: 0, A: 255}
	} else {
		// Темный фон - используем белый текст
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
}

// TagButton - виджет для отображения тега с цветным фоном
type TagButton struct {
	widget.BaseWidget
	text      string
	bgColor   color.RGBA
	textColor color.Color
	onClick   func()
}

// NewTagButton создает новый тег-баттон
func NewTagButton(text string, bgColor color.RGBA, textColor color.Color, onClick func()) *TagButton {
	tb := &TagButton{
		text:      text,
		bgColor:   bgColor,
		textColor: textColor,
		onClick:   onClick,
	}
	tb.ExtendBaseWidget(tb)
	return tb
}

// CreateRenderer создает рендерер для TagButton
func (tb *TagButton) CreateRenderer() fyne.WidgetRenderer {
	textObj := canvas.NewText(tb.text, tb.textColor)
	textObj.TextSize = 12
	textObj.Alignment = fyne.TextAlignCenter

	bgRect := canvas.NewRectangle(tb.bgColor)
	bgRect.CornerRadius = 15
	bgRect.StrokeColor = color.RGBA{48, 48, 48, 255}
	bgRect.StrokeWidth = 1

	// Центрируем текст
	centerContainer := container.NewCenter(textObj)

	stack := container.NewStack(bgRect, centerContainer)

	return &TagButtonRenderer{
		tagButton: tb,
		bgRect:    bgRect,
		textObj:   textObj,
		container: stack,
		objects:   []fyne.CanvasObject{stack},
	}
}

// MinSize возвращает минимальный размер
func (tb *TagButton) MinSize() fyne.Size {
	return fyne.NewSize(60, 30)
}

// Tapped обрабатывает клик
func (tb *TagButton) Tapped(_ *fyne.PointEvent) {
	if tb.onClick != nil {
		tb.onClick()
	}
}

// TagButtonRenderer - рендерер для TagButton
type TagButtonRenderer struct {
	tagButton *TagButton
	bgRect    *canvas.Rectangle
	textObj   *canvas.Text
	container fyne.CanvasObject // Изменил тип на fyne.CanvasObject
	objects   []fyne.CanvasObject
}

// Layout распологает элементы
func (r *TagButtonRenderer) Layout(size fyne.Size) {
	r.container.Resize(size)
}

// MinSize возвращает минимальный размер
func (r *TagButtonRenderer) MinSize() fyne.Size {
	return r.tagButton.MinSize()
}

// Refresh обновляет отображение
func (r *TagButtonRenderer) Refresh() {
	r.bgRect.FillColor = r.tagButton.bgColor
	r.textObj.Color = r.tagButton.textColor
	r.textObj.Text = r.tagButton.text
	canvas.Refresh(r.tagButton)
}

// Objects возвращает объекты для рендеринга
func (r *TagButtonRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

// Destroy освобождает ресурсы
func (r *TagButtonRenderer) Destroy() {}

// getTagsContainer возвращает контейнер с цветными кнопками тегов для элемента
func getTagsContainer(item *models.Item, handler SearchHandler) fyne.CanvasObject {
	tags, err := queries.GetTagsForItem(context.Background(), item.ID)
	if err != nil || len(tags) == 0 {
		return container.NewHBox(widget.NewLabel("--теги отсутствуют--"))
	}

	var tagButtons []fyne.CanvasObject
	for _, tag := range tags {
		hexColor := tag.Color
		if hexColor == "" {
			hexColor = "#808080"
		}

		rgba, err := parseHexColor(hexColor)
		if err != nil {
			rgba = color.RGBA{R: 128, G: 128, B: 128, A: 255}
		}

		textColor := getContrastColor(rgba)

		tagBtn := NewTagButton(
			tag.Name,
			rgba,
			textColor,
			func(tagName string) func() {
				return func() {
					if handler != nil {
						handler.SetSearchQuery(tagName)
					}
				}
			}(tag.Name),
		)

		tagButtons = append(tagButtons, tagBtn)
	}

	return container.NewHBox(tagButtons...)
}
