// Package right содержит компоненты правой панели (профиль)
package right

import (
	"image/color"
	"os"

	"projectT/internal/services/p2p/network"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Panel представляет правую панель с профилем
type Panel struct {
	container                *fyne.Container
	profileAvatar            *canvas.Image
	profileName              *widget.Label
	profileStatus            *widget.Label
	characteristicsContainer *fyne.Container
	demoElementsContainer    *fyne.Container
	chatsUI                  UIProvider
}

// UIProvider интерфейс для доступа к функциям UI чатов
type UIProvider interface {
	GetP2PService() *network.UIP2P
}

// New создает новую правую панель
func New(chatsUI UIProvider) *Panel {
	p := &Panel{
		chatsUI: chatsUI,
	}
	p.container = p.createProfileArea()
	return p
}

// Container возвращает контейнер панели
func (p *Panel) Container() *fyne.Container {
	return p.container
}

// Refresh обновляет панель
func (p *Panel) Refresh() {
	if p.container != nil {
		p.container.Refresh()
	}
}

// UpdateProfile обновляет профиль собеседника
func (p *Panel) UpdateProfile(contact *models.Contact) {
	// Проверяем, локальный ли это чат
	if contact.IsLocalChat() {
		// Для локального чата показываем профиль текущего пользователя
		p.showUserProfile()
		return
	}

	// Обновляем имя
	if p.profileName != nil {
		p.profileName.SetText(contact.Username)
	}

	// Обновляем статус (текстовый, из профиля)
	if p.profileStatus != nil {
		p.profileStatus.SetText(contact.Title)
	}

	// Загружаем аватар
	p.loadAvatar(contact.AvatarPath)

	// Загружаем характеристики из профиля пира
	if contact.PeerID != "" && p.characteristicsContainer != nil {
		// Загружаем профиль из таблицы profiles по PeerID
		profile, err := queries.GetProfileByPeerID(contact.PeerID)
		if err == nil && profile != nil {
			if profile.ContentChar != "" {
				p.loadCharacteristics(profile.ContentChar)
			} else {
				// Если характеристик нет, очищаем контейнер
				p.characteristicsContainer.Objects = nil
				p.characteristicsContainer.Refresh()
			}
		}
	}

	// Загружаем demo элементы из профиля пира
	if contact.PeerID != "" && p.demoElementsContainer != nil {
		// Загружаем профиль из таблицы profiles по PeerID
		profile, err := queries.GetProfileByPeerID(contact.PeerID)
		if err == nil && profile != nil {
			if profile.PinnedUUIDs != "" {
				p.loadDemoElements(profile.PinnedUUIDs)
			} else {
				// Если demo элементов нет, очищаем контейнер
				p.demoElementsContainer.Objects = nil
				p.demoElementsContainer.Refresh()
			}
		}
	}

	// Обновляем UI
	if p.container != nil {
		p.container.Refresh()
	}
}

// showUserProfile показывает профиль текущего пользователя в правой панели
func (p *Panel) showUserProfile() {
	// Загружаем локальный профиль
	localProfile, err := queries.GetLocalProfile()
	if err != nil {
		return
	}

	// Создаём временный контакт с данными профиля
	tempContact := &models.Contact{
		Username:   localProfile.Username,
		Title:      localProfile.Title, // Титул из профиля
		AvatarPath: localProfile.AvatarPath,
		PeerID:     localProfile.PeerID,
	}

	// Обновляем правую панель с профилем пользователя
	p.UpdateProfile(tempContact)

	// Загружаем характеристики из профиля
	if localProfile.ContentChar != "" && p.characteristicsContainer != nil {
		p.loadCharacteristics(localProfile.ContentChar)
	}

	// Загружаем избранные элементы из pinned_uuids
	if localProfile.PinnedUUIDs != "" && p.demoElementsContainer != nil {
		p.loadDemoElements(localProfile.PinnedUUIDs)
	}
}

// createProfileArea создает правую панель с профилем собеседника
func (p *Panel) createProfileArea() *fyne.Container {
	// Аватар - изображение 100x100
	p.profileAvatar = canvas.NewImageFromResource(nil)
	p.profileAvatar.FillMode = canvas.ImageFillContain

	// Черный фон 100x100 под аватарку
	avatarBg := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 255})
	avatarBg.SetMinSize(fyne.NewSize(100, 100))

	// Аватарка поверх фона через Stack
	avatarStack := container.NewStack(avatarBg, p.profileAvatar)

	// Имя
	p.profileName = widget.NewLabel("")
	p.profileName.TextStyle = fyne.TextStyle{Bold: true}
	p.profileName.Alignment = fyne.TextAlignCenter

	// Текстовый статус пользователя (устанавливается вручную)
	p.profileStatus = widget.NewLabel("")
	p.profileStatus.TextStyle = fyne.TextStyle{Italic: true}
	p.profileStatus.Alignment = fyne.TextAlignCenter

	// Контейнер для аватара и имени
	headerContainer := container.NewVBox(
		container.NewCenter(avatarStack),
		p.profileName,
		p.profileStatus,
	)

	// Разделитель
	separator1 := canvas.NewRectangle(color.RGBA{R: 64, G: 64, B: 64, A: 255})
	separator1.SetMinSize(fyne.NewSize(200, 1))

	// Заголовок характеристик
	characteristicsTitle := widget.NewLabel("Характеристики")
	characteristicsTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Контейнер для характеристик
	p.characteristicsContainer = container.NewVBox()

	// Заголовок витрины элементов
	demoElementsTitle := widget.NewLabel("Витрина элементов")
	demoElementsTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Контейнер для demo элементов
	p.demoElementsContainer = container.NewVBox()

	// Основная информация (без внутренних прокруток)
	infoContainer := container.NewVBox(
		headerContainer,
		separator1,
		container.NewPadded(container.NewVBox(characteristicsTitle, p.characteristicsContainer)),
		separator1,
		container.NewPadded(container.NewVBox(demoElementsTitle, p.demoElementsContainer)),
	)

	// Оборачиваем всю панель в прокрутку
	scrollContainer := container.NewScroll(infoContainer)

	// Фон
	bg := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 255})
	bg.SetMinSize(fyne.NewSize(330, 1))

	p.container = container.NewStack(bg, scrollContainer)

	// Загружаем профиль текущего пользователя при инициализации
	p.showUserProfile()

	return p.container
}

// loadAvatar загружает аватар из локального хранилища
func (p *Panel) loadAvatar(avatarPath string) {
	if p.profileAvatar == nil {
		return
	}

	if avatarPath == "" {
		// Пустой аватар - скрываем изображение
		p.profileAvatar.Resource = nil
		p.profileAvatar.Refresh()
		return
	}

	// Проверяем существование файла
	if _, err := os.Stat(avatarPath); os.IsNotExist(err) {
		p.profileAvatar.Resource = nil
		p.profileAvatar.Refresh()
		return
	}

	// Загружаем изображение
	avatarImg, err := fyne.LoadResourceFromPath(avatarPath)
	if err != nil {
		p.profileAvatar.Resource = nil
		p.profileAvatar.Refresh()
		return
	}

	// Устанавливаем изображение
	p.profileAvatar.Resource = avatarImg
	p.profileAvatar.FillMode = canvas.ImageFillContain
	p.profileAvatar.Refresh()
}
