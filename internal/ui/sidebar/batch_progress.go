package sidebar

import (
	"fmt"
	"image/color"
	"time"

	"projectT/internal/services/p2p/protocols/transfer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// BatchProgressWidget виджет для отображения прогресса пакетной передачи
type BatchProgressWidget struct {
	container   *fyne.Container
	overallBar  *widget.ProgressBar
	statusLabel *widget.Label
	itemList    *widget.List
	transferSvc *transfer.Service
	activeBatch *transfer.BatchProgress
	batchItems  map[string]*transfer.BatchItemProgress
	lastBatchID string
	itemKeys    []string
}

// NewBatchProgressWidget создаёт виджет прогресса батча
func NewBatchProgressWidget(transferSvc *transfer.Service) *BatchProgressWidget {
	bpw := &BatchProgressWidget{
		transferSvc: transferSvc,
		batchItems:  make(map[string]*transfer.BatchItemProgress),
	}

	// Общий прогресс-бар
	bpw.overallBar = widget.NewProgressBar()
	bpw.overallBar.Hide()

	// Статус
	bpw.statusLabel = widget.NewLabel("")
	bpw.statusLabel.TextStyle = fyne.TextStyle{Italic: true}
	bpw.statusLabel.Truncation = fyne.TextTruncateEllipsis

	// Список элементов
	bpw.itemList = widget.NewList(
		func() int { return len(bpw.itemKeys) },
		func() fyne.CanvasObject {
			iconText := canvas.NewText("○", color.Gray{Y: 128})
			iconText.TextSize = 14
			progressBar := widget.NewProgressBar()
			percentLabel := widget.NewLabel("0%")
			titleLabel := widget.NewLabel("")
			titleLabel.Truncation = fyne.TextTruncateEllipsis
			return container.NewHBox(iconText, titleLabel, progressBar, percentLabel)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(bpw.itemKeys) {
				return
			}
			key := bpw.itemKeys[id]
			item, exists := bpw.batchItems[key]
			if !exists {
				return
			}

			row := obj.(*fyne.Container)
			iconText := row.Objects[0].(*canvas.Text)
			titleLabel := row.Objects[1].(*widget.Label)
			progressBar := row.Objects[2].(*widget.ProgressBar)
			percentLabel := row.Objects[3].(*widget.Label)

			titleLabel.SetText(item.Title)

			switch item.Status {
			case transfer.TransferStatusCompleted:
				iconText.Text = "✓"
				iconText.Color = color.RGBA{R: 0, G: 200, B: 0, A: 255}
				progressBar.SetValue(1.0)
				percentLabel.SetText("100%")
			case transfer.TransferStatusFailed:
				iconText.Text = "✗"
				iconText.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255}
				progressBar.SetValue(0)
				percentLabel.SetText("err")
			case transfer.TransferStatusInProgress:
				iconText.Text = "⟳"
				iconText.Color = color.RGBA{R: 255, G: 200, B: 0, A: 255}
				pct := float64(0)
				if item.FileSize > 0 {
					pct = float64(item.Transferred) / float64(item.FileSize)
				}
				progressBar.SetValue(pct)
				percentLabel.SetText(fmt.Sprintf("%.0f%%", pct*100))
			default:
				iconText.Text = "○"
				iconText.Color = color.Gray{Y: 128}
				progressBar.SetValue(0)
				percentLabel.SetText("0%")
			}

			iconText.Refresh()
		},
	)
	bpw.itemList.Hide()

	// Компоновка
	content := container.NewVBox(
		container.NewHBox(widget.NewIcon(theme.DownloadIcon()), bpw.statusLabel),
		bpw.overallBar,
		bpw.itemList,
	)

	bg := canvas.NewRectangle(color.RGBA{R: 30, G: 30, B: 30, A: 255})
	bg.SetMinSize(fyne.NewSize(0, 80))

	bpw.container = container.NewStack(bg, container.NewPadded(content))
	bpw.container.Hide() // Скрыт по умолчанию

	// Подписка на прогресс батча
	if transferSvc != nil {
		go bpw.monitorBatchProgress()
		go bpw.monitorBatchItemProgress()
	}

	return bpw
}

// monitorBatchProgress отслеживает прогресс батчей
func (bpw *BatchProgressWidget) monitorBatchProgress() {
	if bpw.transferSvc == nil {
		return
	}

	ch := bpw.transferSvc.BatchProgressChan()
	for progress := range ch {
		bpw.updateBatchProgress(progress)
	}
}

// monitorBatchItemProgress отслеживает прогресс элементов батча
func (bpw *BatchProgressWidget) monitorBatchItemProgress() {
	if bpw.transferSvc == nil {
		return
	}

	ch := bpw.transferSvc.BatchItemProgressChan()
	for itemProg := range ch {
		bpw.updateBatchItemProgress(itemProg)
	}
}

// updateBatchProgress обновляет общий прогресс батча
func (bpw *BatchProgressWidget) updateBatchProgress(progress *transfer.BatchProgress) {
	fyne.CurrentApp().Driver().DoFromGoroutine(func() {
		if bpw.lastBatchID == progress.BatchID && progress.Status == transfer.TransferStatusInProgress {
			return
		}

		bpw.lastBatchID = progress.BatchID
		bpw.activeBatch = progress

		bpw.container.Show()
		bpw.overallBar.Show()
		bpw.statusLabel.Show()

		switch progress.Status {
		case transfer.TransferStatusInProgress:
			bpw.overallBar.SetValue(progress.OverallPercent / 100.0)
			statusText := fmt.Sprintf("Batch: %s (%d/%d, %.0f%%)", progress.Type, progress.Completed, progress.TotalItems, progress.OverallPercent)
			bpw.statusLabel.SetText(statusText)

		case transfer.TransferStatusCompleted:
			bpw.overallBar.SetValue(1.0)
			statusText := fmt.Sprintf("✓ Batch completed: %d/%d", progress.Completed, progress.TotalItems)
			if progress.Failed > 0 {
				statusText += fmt.Sprintf(" (%d errors)", progress.Failed)
			}
			bpw.statusLabel.SetText(statusText)
			bpw.scheduleHide()

		case transfer.TransferStatusFailed:
			bpw.overallBar.SetValue(0)
			bpw.statusLabel.SetText(fmt.Sprintf("✗ Batch failed: %s", progress.Error))
			bpw.scheduleHide()

		case transfer.TransferStatusCancelled:
			bpw.overallBar.SetValue(0)
			bpw.statusLabel.SetText("✗ Batch cancelled")
			bpw.scheduleHide()
		}

		bpw.container.Refresh()
	}, false)
}

// updateBatchItemProgress обновляет прогресс элемента
func (bpw *BatchProgressWidget) updateBatchItemProgress(item *transfer.BatchItemProgress) {
	fyne.CurrentApp().Driver().DoFromGoroutine(func() {
		key := fmt.Sprintf("%s_%s", item.BatchID, item.ElementUUID)
		bpw.batchItems[key] = item

		// Обновляем ключи списка
		bpw.itemKeys = make([]string, 0, len(bpw.batchItems))
		for k := range bpw.batchItems {
			bpw.itemKeys = append(bpw.itemKeys, k)
		}

		// Показываем список если есть элементы
		if len(bpw.itemKeys) > 0 {
			bpw.itemList.Show()
		}

		bpw.itemList.Refresh()
	}, false)
}

// scheduleHide планирует скрытие
func (bpw *BatchProgressWidget) scheduleHide() {
	time.AfterFunc(5*time.Second, func() {
		fyne.CurrentApp().Driver().DoFromGoroutine(func() {
			bpw.overallBar.Hide()
			bpw.statusLabel.SetText("")
			bpw.itemList.Hide()
			bpw.batchItems = make(map[string]*transfer.BatchItemProgress)
			bpw.itemKeys = nil
			bpw.container.Hide()
			bpw.container.Refresh()
		}, false)
	})
}

// Container возвращает контейнер
func (bpw *BatchProgressWidget) Container() *fyne.Container {
	return bpw.container
}

// SetVisible устанавливает видимость
func (bpw *BatchProgressWidget) SetVisible(visible bool) {
	if visible {
		bpw.container.Show()
	} else {
		bpw.container.Hide()
	}
}
