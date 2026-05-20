package compilation

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"sync"

	"projectT/internal/services/p2p/protocols/itemsync"
	"projectT/internal/services/p2p/protocols/transfer"
	network "projectT/internal/services/p2p/ui"
	db_models "projectT/internal/storage/database/models"
	"projectT/internal/ui/workspace/saved"

	"github.com/libp2p/go-libp2p/core/peer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type CompilationUI struct {
	content          *fyne.Container
	gridManager      *saved.GridManager
	p2pUI            *network.UIP2P
	refreshBtn       *widget.Button
	statusLabel      *widget.Label
	loadingBar       *widget.ProgressBar
	loadingContainer *fyne.Container
	mainContent      *fyne.Container

	currentBatchID string
	loading        bool
	mu             sync.Mutex
}

func NewCompilationUI(p2pUI *network.UIP2P) *CompilationUI {
	ui := &CompilationUI{
		p2pUI: p2pUI,
	}
	ui.createView()
	return ui
}

func (ui *CompilationUI) createView() {
	ui.gridManager = saved.NewGridManager()
	ui.gridManager.SetColumnCount(3)

	ui.statusLabel = widget.NewLabel("Click Refresh to load random items from connected peers")
	ui.statusLabel.TextStyle = fyne.TextStyle{Italic: true}
	ui.statusLabel.Wrapping = fyne.TextWrapWord

	ui.loadingBar = widget.NewProgressBar()
	ui.loadingBar.Hide()

	ui.refreshBtn = widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		ui.LoadRandomCompilation()
	})
	ui.refreshBtn.Importance = widget.HighImportance

	header := container.NewBorder(
		nil,
		nil,
		ui.refreshBtn,
		nil,
		widget.NewLabel("Compilation"),
	)

	ui.loadingContainer = container.NewVBox(
		container.NewCenter(ui.statusLabel),
		ui.loadingBar,
	)
	ui.loadingContainer.Hide()

	ui.mainContent = container.NewStack(
		container.NewBorder(
			header,
			nil,
			nil,
			nil,
			ui.gridManager.GetContainer(),
		),
		ui.loadingContainer,
	)

	// Placeholder when empty
	placeholder := canvas.NewText("No items loaded", color.Gray{Y: 128})
	placeholder.Alignment = fyne.TextAlignCenter
	placeholderBg := canvas.NewRectangle(color.RGBA{R: 30, G: 30, B: 30, A: 50})
	placeholderContainer := container.NewStack(placeholderBg, container.NewCenter(placeholder))

	ui.content = container.NewStack(
		placeholderContainer,
		ui.mainContent,
	)
}

func (ui *CompilationUI) Container() fyne.CanvasObject {
	return ui.content
}

func (ui *CompilationUI) Refresh() {
	if ui.content != nil {
		ui.content.Refresh()
	}
}

func (ui *CompilationUI) LoadRandomCompilation() {
	ui.mu.Lock()
	if ui.loading {
		ui.mu.Unlock()
		return
	}
	ui.loading = true
	ui.mu.Unlock()

	connectedPeers := ui.p2pUI.GetConnectedPeers()
	if len(connectedPeers) == 0 {
		ui.statusLabel.SetText("No connected peers to load items from")
		ui.loadingContainer.Show()
		ui.mainContent.Refresh()
		ui.mu.Lock()
		ui.loading = false
		ui.mu.Unlock()
		return
	}

	ui.gridManager.Clear()

	ui.statusLabel.SetText(fmt.Sprintf("Loading from %d peers...", len(connectedPeers)))
	ui.loadingBar.SetValue(0)
	ui.loadingBar.Show()
	ui.loadingContainer.Show()
	ui.mainContent.Refresh()

	// Shuffle peers for randomness
	shuffled := make([]*network.PeerInfo, len(connectedPeers))
	copy(shuffled, connectedPeers)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	totalPeers := len(shuffled)
	itemsPerPeer := 5
	var loadedCount int
	var mu sync.Mutex

	batchID := fmt.Sprintf("compilation_batch_%d", rand.Intn(100000))
	ui.currentBatchID = batchID

	transferSvc := ui.p2pUI.GetNetwork().Transfer()
	if transferSvc != nil {
		transferSvc.StartReceiveBatch(batchID, totalPeers*itemsPerPeer, 0)
	}

	var wg sync.WaitGroup
	for _, peerInfo := range shuffled {
		wg.Add(1)
		go func(pi *network.PeerInfo) {
			defer wg.Done()

			p, err := peer.Decode(pi.PeerID)
			if err != nil {
				log.Printf("[Compilation] Ошибка декодирования peer ID %s: %v", pi.PeerID, err)
				return
			}

			callbacks := itemsync.BatchRequestCallbacks{
				OnItem: func(item *db_models.Item, index int, total int) {
					if err := ui.gridManager.AddItem(item); err != nil {
						log.Printf("[Compilation] Ошибка добавления элемента в grid: %v", err)
					}

					mu.Lock()
					loadedCount++
					currentLoaded := loadedCount
					mu.Unlock()

					progress := float64(currentLoaded) / float64(totalPeers*itemsPerPeer)
					ui.updateProgress(progress, currentLoaded, item.Title)

					if transferSvc != nil {
						transferSvc.UpdateReceiveBatchItem(batchID, item.Title, currentLoaded, totalPeers*itemsPerPeer)
					}
				},
				OnProgress: func(completed int, total int, currentItem string) {
					mu.Lock()
					loadedCount++
					currentLoaded := loadedCount
					mu.Unlock()

					progress := float64(currentLoaded) / float64(totalPeers*itemsPerPeer)
					ui.updateProgress(progress, currentLoaded, currentItem)

					if transferSvc != nil {
						transferSvc.UpdateReceiveBatchItem(batchID, currentItem, currentLoaded, totalPeers*itemsPerPeer)
					}
				},
				OnDone: func(items []*db_models.Item, err error) {
					if err != nil {
						log.Printf("[Compilation] Ошибка загрузки от пира %s: %v", pi.Username, err)
					}
				},
			}

			ui.p2pUI.RequestRandomItemsAsync(p, itemsPerPeer, callbacks)
		}(peerInfo)
	}

	go func() {
		wg.Wait()

		ui.mu.Lock()
		ui.loading = false
		ui.mu.Unlock()

		if transferSvc != nil {
			transferSvc.CompleteReceiveBatch(batchID, transfer.TransferStatusCompleted, "")
		}

		ui.loadingBar.Hide()
		ui.loadingContainer.Hide()

		mu.Lock()
		totalLoaded := loadedCount
		mu.Unlock()

		if totalLoaded > 0 {
			ui.statusLabel.SetText(fmt.Sprintf("Loaded %d items from %d peers", totalLoaded, totalPeers))
		} else {
			ui.statusLabel.SetText("No items found from connected peers")
		}

		ui.gridManager.UpdateLayout()
		ui.mainContent.Refresh()
	}()
}

func (ui *CompilationUI) updateProgress(value float64, count int, currentItem string) {
	if ui.loadingBar == nil || ui.statusLabel == nil {
		return
	}

	ui.loadingBar.SetValue(value)
	ui.statusLabel.SetText(fmt.Sprintf("Loading... %d items (%s)", count, currentItem))
	ui.loadingContainer.Refresh()
}
