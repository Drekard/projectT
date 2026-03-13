package chats

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"os"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// ContentCharacteristicItem представляет элемент характеристики
type ContentCharacteristicItem struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Value string `json:"value"`
}

// createProfileArea создает правую панель с профилем собеседника
func (ui *UI) createProfileArea() *fyne.Container {
	// Аватар - изображение 100x100
	ui.profileAvatar = canvas.NewImageFromResource(nil)
	ui.profileAvatar.FillMode = canvas.ImageFillContain

	// Черный фон 100x100 под аватарку
	avatarBg := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 255})
	avatarBg.SetMinSize(fyne.NewSize(100, 100))

	// Аватарка поверх фона через Stack
	avatarStack := container.NewStack(avatarBg, ui.profileAvatar)

	// Имя
	ui.profileName = widget.NewLabel("")
	ui.profileName.TextStyle = fyne.TextStyle{Bold: true}
	ui.profileName.Alignment = fyne.TextAlignCenter

	// Текстовый статус пользователя (устанавливается вручную)
	ui.profileStatus = widget.NewLabel("")
	ui.profileStatus.TextStyle = fyne.TextStyle{Italic: true}
	ui.profileStatus.Alignment = fyne.TextAlignCenter

	// Отступы сверху и снизу
	spacerTop := canvas.NewRectangle(color.Transparent)
	spacerTop.SetMinSize(fyne.NewSize(0, 20))
	spacerBottom := canvas.NewRectangle(color.Transparent)
	spacerBottom.SetMinSize(fyne.NewSize(0, 20))

	// Контейнер для аватара и имени
	headerContainer := container.NewVBox(
		spacerTop,
		container.NewCenter(avatarStack),
		spacerBottom,
		ui.profileName,
		ui.profileStatus,
		layout.NewSpacer(),
	)

	// Разделитель
	separator1 := canvas.NewRectangle(color.RGBA{R: 64, G: 64, B: 64, A: 255})
	separator1.SetMinSize(fyne.NewSize(200, 1))

	// Заголовок характеристик
	characteristicsTitle := widget.NewLabel("Характеристики")
	characteristicsTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Контейнер для характеристик
	ui.characteristicsContainer = container.NewVBox()
	characteristicsScroll := container.NewScroll(ui.characteristicsContainer)
	characteristicsScroll.SetMinSize(fyne.NewSize(0, 200))

	// Основная информация
	infoContainer := container.NewVBox(
		container.NewPadded(headerContainer),
		separator1,
		container.NewPadded(container.NewVBox(characteristicsTitle, characteristicsScroll)),
	)

	// Фон
	bg := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 255})

	ui.profileArea = container.NewStack(bg, infoContainer)

	// Загружаем профиль текущего пользователя при инициализации
	ui.showUserProfile()

	return ui.profileArea
}

// showUserProfile показывает профиль текущего пользователя в правой панели
func (ui *UI) showUserProfile() {
	// Загружаем локальный профиль
	localProfile, err := queries.GetLocalProfile()
	if err != nil {
		log.Printf("Ошибка загрузки локального профиля: %v", err)
		return
	}

	// Создаём временный контакт с данными профиля
	tempContact := &models.Contact{
		Username:   localProfile.Username,
		Status:     localProfile.Status, // Текстовый статус из профиля
		AvatarPath: localProfile.AvatarPath,
		PeerID:     localProfile.PeerID,
	}

	// Обновляем правую панель с профилем пользователя
	ui.updateProfile(tempContact)

	// Загружаем характеристики из профиля
	if localProfile.ContentChar != "" && ui.characteristicsContainer != nil {
		ui.loadCharacteristics(localProfile.ContentChar)
	}
}

// updateProfile обновляет профиль собеседника
func (ui *UI) updateProfile(contact *models.Contact) {
	// Обновляем имя
	if ui.profileName != nil {
		ui.profileName.SetText(contact.Username)
	}

	// Обновляем статус (текстовый, устанавливается пользователем)
	if ui.profileStatus != nil {
		ui.profileStatus.SetText(contact.Status)
	}

	// Загружаем аватар
	if ui.profileAvatar != nil {
		ui.loadAvatar(contact.AvatarPath)
	}

	// Загружаем характеристики из профиля пира
	if contact.PeerID != "" && ui.characteristicsContainer != nil {
		// Загружаем профиль из таблицы profiles по PeerID
		profile, err := queries.GetProfileByPeerID(contact.PeerID)
		if err == nil && profile != nil {
			if profile.ContentChar != "" {
				ui.loadCharacteristics(profile.ContentChar)
			} else {
				// Если характеристик нет, очищаем контейнер
				ui.characteristicsContainer.Objects = nil
				ui.characteristicsContainer.Refresh()
			}
		}
	}

	// Обновляем UI
	if ui.profileArea != nil {
		ui.profileArea.Refresh()
	}
}

// loadAvatar загружает аватар из локального хранилища
func (ui *UI) loadAvatar(avatarPath string) {
	if ui.profileAvatar == nil {
		return
	}

	if avatarPath == "" {
		// Пустой аватар - скрываем изображение
		ui.profileAvatar.Resource = nil
		ui.profileAvatar.Refresh()
		return
	}

	// Проверяем существование файла
	if _, err := os.Stat(avatarPath); os.IsNotExist(err) {
		log.Printf("Аватар не найден: %s", avatarPath)
		ui.profileAvatar.Resource = nil
		ui.profileAvatar.Refresh()
		return
	}

	// Загружаем изображение
	avatarImg, err := fyne.LoadResourceFromPath(avatarPath)
	if err != nil {
		log.Printf("Ошибка загрузки аватара: %v", err)
		ui.profileAvatar.Resource = nil
		ui.profileAvatar.Refresh()
		return
	}

	// Устанавливаем изображение
	ui.profileAvatar.Resource = avatarImg
	ui.profileAvatar.FillMode = canvas.ImageFillContain
	ui.profileAvatar.Refresh()
}

// loadCharacteristics загружает характеристики из JSON
func (ui *UI) loadCharacteristics(jsonStr string) {
	if ui.characteristicsContainer == nil {
		return
	}

	ui.characteristicsContainer.Objects = nil

	var characteristics []ContentCharacteristicItem
	if jsonStr != "" {
		err := json.Unmarshal([]byte(jsonStr), &characteristics)
		if err != nil {
			log.Printf("Ошибка парсинга характеристик: %v", err)
			return
		}
	}

	if len(characteristics) == 0 {
		emptyLabel := widget.NewLabel("Нет характеристик")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.characteristicsContainer.Add(emptyLabel)
	} else {
		for _, item := range characteristics {
			characteristicItem := ui.createCharacteristicItem(item.Title, item.Value)
			ui.characteristicsContainer.Add(characteristicItem)
		}
	}

	ui.characteristicsContainer.Refresh()
}

// createCharacteristicItem создает элемент характеристики (название: значение в одну строку)
func (ui *UI) createCharacteristicItem(title, value string) *fyne.Container {
	// Форматируем как "Название: Значение"
	text := fmt.Sprintf("%s: %s", title, value)
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	return container.NewVBox(label)
}
