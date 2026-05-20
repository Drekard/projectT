package profile

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"os"
	"sync"

	"projectT/internal/services/p2p/protocols/itemsync"
	"projectT/internal/services/p2p/protocols/transfer"
	network "projectT/internal/services/p2p/ui"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/ui/workspace/saved"

	"github.com/libp2p/go-libp2p/core/peer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// RemoteProfileUI представляет read-only профиль удалённого пользователя
type RemoteProfileUI struct {
	content                  fyne.CanvasObject
	profileAvatar            *canvas.Image
	profileName              *widget.Label
	profileTitle             *widget.Label
	characteristicsContainer *fyne.Container
	gridManager              *saved.GridManager
	peerID                   string
	isLocal                  bool
	p2pUI                    *network.UIP2P
	onOpenElements           func()

	// Loading state
	rightPanelContent *fyne.Container
	loadingIndicator  *widget.ProgressBar
	loadingLabel      *widget.Label
	loadingContainer  *fyne.Container
	currentBatchID    string
}

// NewRemoteProfileUI создаёт новый read-only профиль удалённого пользователя
func NewRemoteProfileUI(peerID string, p2pUI *network.UIP2P) *RemoteProfileUI {
	ui := &RemoteProfileUI{
		peerID: peerID,
		p2pUI:  p2pUI,
	}

	// Определяем тип профиля до создания UI, чтобы корректно отобразить кнопку
	profile, err := queries.GetProfileByPeerID(peerID)
	if err == nil && profile != nil {
		ui.isLocal = profile.OwnerType == "local"
	}

	ui.createView()
	ui.loadProfile()
	return ui
}

// Container возвращает контейнер
func (ui *RemoteProfileUI) Container() fyne.CanvasObject {
	return ui.content
}

// Refresh обновляет профиль
func (ui *RemoteProfileUI) Refresh() {
	ui.loadProfile()
	if ui.content != nil {
		ui.content.Refresh()
	}
}

// SetOnOpenElements устанавливает callback для открытия элементов
func (ui *RemoteProfileUI) SetOnOpenElements(callback func()) {
	ui.onOpenElements = callback
}

// GetPeerID возвращает peerID профиля
func (ui *RemoteProfileUI) GetPeerID() string {
	return ui.peerID
}

// GetProfileName возвращает имя профиля
func (ui *RemoteProfileUI) GetProfileName() string {
	if ui.profileName != nil {
		return ui.profileName.Text
	}
	return ""
}

func (ui *RemoteProfileUI) createView() {
	// Левая панель (аватар, имя, title, характеристики)
	leftPanel := ui.createLeftPanel()

	// Правая панель (pinned items)
	rightPanel := ui.createRightPanel()

	// Split
	split := container.NewHSplit(leftPanel, rightPanel)
	split.SetOffset(0.35)

	ui.content = split
}

func (ui *RemoteProfileUI) createLeftPanel() fyne.CanvasObject {
	// Аватар
	ui.profileAvatar = canvas.NewImageFromFile("storage/files/avatars/local/ProjctT_true.png")
	ui.profileAvatar.FillMode = canvas.ImageFillContain
	ui.profileAvatar.SetMinSize(fyne.NewSize(100, 100))

	avatarBg := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 255})
	avatarBg.SetMinSize(fyne.NewSize(100, 100))
	avatarStack := container.NewStack(avatarBg, ui.profileAvatar)

	// Имя (read-only)
	ui.profileName = widget.NewLabel("")
	ui.profileName.TextStyle = fyne.TextStyle{Bold: true}

	// Title (read-only)
	ui.profileTitle = widget.NewLabel("")
	ui.profileTitle.TextStyle = fyne.TextStyle{Italic: true}

	// Кнопка "Открыть элементы" — только для remote профилей
	openElementsBtn := widget.NewButton("Open Elements", func() {
		if ui.onOpenElements != nil {
			ui.onOpenElements()
		}
	})
	openElementsBtn.Importance = widget.HighImportance

	// Характеристики
	ui.characteristicsContainer = container.NewVBox()
	charScroll := container.NewScroll(ui.characteristicsContainer)
	charScroll.SetMinSize(fyne.NewSize(0, 150))

	// Разделитель
	separator := canvas.NewRectangle(color.Gray{Y: 128})
	separator.SetMinSize(fyne.NewSize(0, 1))

	leftContent := container.NewVBox(
		container.NewCenter(avatarStack),
		ui.profileName,
		ui.profileTitle,
	)

	// Для remote профилей добавляем кнопку "Open Elements"
	if !ui.isLocal {
		fmt.Printf("Creating openElementsBtn for remote profile: %s\n", ui.peerID)
		leftContent.Add(container.NewCenter(openElementsBtn))
	}

	leftContent.Add(separator)
	leftContent.Add(widget.NewLabel("Characteristics"))
	leftContent.Add(charScroll)

	return container.NewScroll(leftContent)
}

func (ui *RemoteProfileUI) createRightPanel() fyne.CanvasObject {
	// GridManager для pinned items (как в profile.go)
	ui.gridManager = saved.NewGridManager()
	ui.gridManager.SetColumnCount(2)

	pinnedGridContainer := ui.gridManager.GetContainer()

	// Loading indicator
	ui.loadingLabel = widget.NewLabel("")
	ui.loadingLabel.TextStyle = fyne.TextStyle{Italic: true}
	ui.loadingLabel.Alignment = fyne.TextAlignCenter

	ui.loadingIndicator = widget.NewProgressBar()
	ui.loadingIndicator.Hide()

	loadingContent := container.NewVBox(
		container.NewCenter(ui.loadingLabel),
		ui.loadingIndicator,
	)
	ui.loadingContainer = container.NewPadded(loadingContent)
	ui.loadingContainer.Hide()

	// Основной контент с overlay для загрузки
	ui.rightPanelContent = container.NewStack(
		container.NewBorder(
			widget.NewLabel("Showcase"),
			nil,
			nil,
			nil,
			pinnedGridContainer,
		),
		ui.loadingContainer,
	)

	return ui.rightPanelContent
}

func (ui *RemoteProfileUI) loadProfile() {
	// Загружаем профиль из БД (по peerID без фильтра owner_type)
	profile, err := queries.GetProfileByPeerID(ui.peerID)
	if err != nil {
		log.Printf("[RemoteProfile] Ошибка загрузки профиля %s: %v", ui.peerID[:min(10, len(ui.peerID))], err)
		return
	}

	if profile == nil {
		log.Printf("[RemoteProfile] Профиль %s не найден", ui.peerID[:min(10, len(ui.peerID))])
		return
	}

	// Определяем, локальный ли это профиль
	ui.isLocal = profile.OwnerType == "local"

	// Имя
	if ui.profileName != nil {
		ui.profileName.SetText(profile.Username)
	}

	// Title
	if ui.profileTitle != nil {
		ui.profileTitle.SetText(profile.Title)
	}

	// Аватар
	ui.loadAvatar(profile.AvatarPath)

	// Характеристики
	if profile.ContentChar != "" {
		ui.loadCharacteristics(profile.ContentChar)
	}

	// Pinned items
	ui.loadPinnedItems(profile)
}

func (ui *RemoteProfileUI) loadAvatar(avatarPath string) {
	if ui.profileAvatar == nil {
		return
	}

	if avatarPath == "" {
		ui.profileAvatar.Resource = nil
		ui.profileAvatar.Refresh()
		return
	}

	if _, err := os.Stat(avatarPath); os.IsNotExist(err) {
		ui.profileAvatar.Resource = nil
		ui.profileAvatar.Refresh()
		return
	}

	avatarImg, err := fyne.LoadResourceFromPath(avatarPath)
	if err != nil {
		ui.profileAvatar.Resource = nil
		ui.profileAvatar.Refresh()
		return
	}

	ui.profileAvatar.Resource = avatarImg
	ui.profileAvatar.FillMode = canvas.ImageFillContain
	ui.profileAvatar.Refresh()
}

func (ui *RemoteProfileUI) loadCharacteristics(contentChar string) {
	if ui.characteristicsContainer == nil {
		return
	}

	ui.characteristicsContainer.Objects = nil

	var characteristics []ContentCharacteristicItem
	if err := json.Unmarshal([]byte(contentChar), &characteristics); err != nil {
		log.Printf("[RemoteProfile] Ошибка парсинга характеристик: %v", err)
		return
	}

	if len(characteristics) == 0 {
		emptyLabel := widget.NewLabel("No characteristics")
		emptyLabel.TextStyle = fyne.TextStyle{Italic: true}
		ui.characteristicsContainer.Add(emptyLabel)
	} else {
		for _, ch := range characteristics {
			if ch.Title == "" {
				continue
			}
			row := ui.createCharacteristicItem(ch.Title, ch.Value)
			ui.characteristicsContainer.Objects = append(ui.characteristicsContainer.Objects, row)
		}
	}

	ui.characteristicsContainer.Refresh()
}

func (ui *RemoteProfileUI) createCharacteristicItem(title, value string) *fyne.Container {
	text := title + ": " + value
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	return container.NewVBox(label)
}

func (ui *RemoteProfileUI) parsePinnedUUIDs(pinnedUUIDsStr string) ([]string, error) {
	if pinnedUUIDsStr == "" || pinnedUUIDsStr == "[]" {
		return []string{}, nil
	}

	var uuids []string
	if err := json.Unmarshal([]byte(pinnedUUIDsStr), &uuids); err != nil {
		return nil, err
	}

	return uuids, nil
}

func (ui *RemoteProfileUI) loadPinnedItems(profile *models.Profile) {
	if ui.gridManager == nil {
		return
	}

	if ui.isLocal {
		// Локальный профиль — берём pinned items из БД
		pinnedItems, err := queries.GetPinnedItems()
		if err != nil {
			pinnedItems = []*models.Item{}
		}
		ui.gridManager.LoadItemsWithoutCreateElement(pinnedItems)
		return
	}

	// Remote профиль — запрашиваем pinned items у пира
	pinnedUUIDs, err := ui.parsePinnedUUIDs(profile.PinnedUUIDs)
	if err != nil {
		log.Printf("[RemoteProfile] Ошибка парсинга pinned_uuids: %v", err)
		ui.gridManager.LoadItemsWithoutCreateElement([]*models.Item{})
		return
	}

	if len(pinnedUUIDs) == 0 {
		ui.gridManager.LoadItemsWithoutCreateElement([]*models.Item{})
		return
	}

	// Сначала пытаемся загрузить из локальной БД (remote элементы)
	remoteItems, err := queries.GetRemoteItemsByPeer(ui.peerID)
	if err == nil && len(remoteItems) > 0 {
		uuidSet := make(map[string]bool)
		for _, uuid := range pinnedUUIDs {
			uuidSet[uuid] = true
		}

		var pinnedItems []*models.Item
		for _, item := range remoteItems {
			if uuidSet[item.ElementUUID] {
				pinnedItems = append(pinnedItems, item)
			}
		}

		if len(pinnedItems) > 0 {
			ui.gridManager.LoadItemsWithoutCreateElement(pinnedItems)
			return
		}
	}

	// Если элементов нет в БД — запрашиваем у пира
	go ui.requestPinnedItemsFromPeer(pinnedUUIDs)
}

func (ui *RemoteProfileUI) requestPinnedItemsFromPeer(pinnedUUIDs []string) {
	if ui.p2pUI == nil {
		log.Printf("[RemoteProfile] p2pUI не установлен, не могу запросить pinned items у пира")
		return
	}

	p, err := queries.GetProfileByPeerID(ui.peerID)
	if err != nil || p == nil {
		return
	}

	// Показываем индикатор загрузки
	ui.showLoading(len(pinnedUUIDs))

	// Создаём batch ID для трекинга прогресса
	batchID := fmt.Sprintf("receive_batch_%s_%d", ui.peerID[:8], len(pinnedUUIDs))
	ui.currentBatchID = batchID

	// Регистрируем батч в transfer service для отображения в sidebar
	transferSvc := ui.p2pUI.GetNetwork().Transfer()
	if transferSvc != nil {
		transferSvc.StartReceiveBatch(batchID, len(pinnedUUIDs), 0)
	}

	var loadedItems []*models.Item
	var mu sync.Mutex

	callbacks := itemsync.BatchRequestCallbacks{
		OnItem: func(item *models.Item, index int, total int) {
			mu.Lock()
			loadedItems = append(loadedItems, item)
			mu.Unlock()

			// Прогрессивно добавляем элемент в grid
			if err := ui.gridManager.AddItem(item); err != nil {
				log.Printf("[RemoteProfile] Ошибка добавления элемента в grid: %v", err)
			}

			// Обновляем прогресс батча
			if transferSvc != nil {
				transferSvc.UpdateReceiveBatchItem(batchID, item.Title, index+1, total)
			}

			// Обновляем локальный прогресс
			ui.updateLoadingProgress(index+1, total, item.Title)
		},
		OnProgress: func(completed int, total int, currentItem string) {
			ui.updateLoadingProgress(completed, total, currentItem)
			if transferSvc != nil {
				transferSvc.UpdateReceiveBatchItem(batchID, currentItem, completed, total)
			}
		},
		OnDone: func(items []*models.Item, doneErr error) {
			ui.hideLoading()

			if transferSvc != nil {
				if doneErr != nil {
					transferSvc.CompleteReceiveBatch(batchID, transfer.TransferStatusFailed, doneErr.Error())
				} else {
					transferSvc.CompleteReceiveBatch(batchID, transfer.TransferStatusCompleted, "")
				}
			}

			if doneErr != nil {
				log.Printf("[RemoteProfile] Ошибка запроса pinned items у пира %s: %v", ui.peerID[:min(10, len(ui.peerID))], doneErr)
				return
			}

			// Если элементов не было добавлено прогрессивно (fallback), загружаем все сразу
			mu.Lock()
			hasItems := len(loadedItems) > 0
			mu.Unlock()

			if hasItems && len(items) > 0 {
				// Элементы уже добавлены прогрессивно, просто убеждаемся что grid обновлён
				ui.gridManager.UpdateLayout()
			}
		},
	}

	ui.p2pUI.RequestBatchByUUIDsAsync(peer.ID(p.PeerID), pinnedUUIDs, callbacks)
}

func (ui *RemoteProfileUI) showLoading(total int) {
	if ui.loadingLabel == nil || ui.loadingIndicator == nil || ui.loadingContainer == nil {
		return
	}

	ui.loadingLabel.SetText(fmt.Sprintf("Loading %d elements...", total))
	ui.loadingIndicator.SetValue(0)
	ui.loadingIndicator.Show()
	ui.loadingContainer.Show()
	ui.rightPanelContent.Refresh()
}

func (ui *RemoteProfileUI) updateLoadingProgress(completed int, total int, currentItem string) {
	if ui.loadingLabel == nil || ui.loadingIndicator == nil {
		return
	}

	if total > 0 {
		progress := float64(completed) / float64(total)
		ui.loadingIndicator.SetValue(progress)
		ui.loadingLabel.SetText(fmt.Sprintf("Loading %d/%d: %s", completed, total, currentItem))
		ui.loadingContainer.Refresh()
	}
}

func (ui *RemoteProfileUI) hideLoading() {
	if ui.loadingContainer == nil {
		return
	}

	ui.loadingIndicator.Hide()
	ui.loadingLabel.SetText("")
	ui.loadingContainer.Hide()
	ui.rightPanelContent.Refresh()
}

// LoadPinnedItemsFromRemote загружает pinned items через ItemSync (вызывается извне)
func (ui *RemoteProfileUI) LoadPinnedItemsFromRemote(items []*models.Item) {
	if ui.gridManager != nil {
		ui.gridManager.LoadItemsWithoutCreateElement(items)
	}
}

// GetPinnedUUIDs возвращает список UUID закреплённых элементов
func (ui *RemoteProfileUI) GetPinnedUUIDs() ([]string, error) {
	profile, err := queries.GetProfileByPeerID(ui.peerID)
	if err != nil {
		return nil, err
	}

	if profile.PinnedUUIDs == "" || profile.PinnedUUIDs == "[]" {
		return []string{}, nil
	}

	var uuids []string
	if err := json.Unmarshal([]byte(profile.PinnedUUIDs), &uuids); err != nil {
		return nil, err
	}

	return uuids, nil
}
