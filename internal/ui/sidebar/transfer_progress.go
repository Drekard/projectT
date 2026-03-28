// Package sidebar содержит компоненты боковой панели приложения
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

// TransferProgressWidget виджет для отображения прогресса передачи файлов
type TransferProgressWidget struct {
	container       *fyne.Container
	progressBar     *widget.ProgressBar
	statusLabel     *widget.Label
	cancelButton    *widget.Button
	icon            *widget.Icon
	currentTransfer *transfer.TransferProgress
	transferSvc     *transfer.Service
	lastTransferID  string
}

// NewTransferProgressWidget создаёт новый виджет прогресса передачи
func NewTransferProgressWidget(transferSvc *transfer.Service) *TransferProgressWidget {
	tpw := &TransferProgressWidget{
		transferSvc: transferSvc,
	}

	// Progress bar
	tpw.progressBar = widget.NewProgressBar()
	tpw.progressBar.Hide() // Скрыт по умолчанию

	// Статус
	tpw.statusLabel = widget.NewLabel("")
	tpw.statusLabel.TextStyle = fyne.TextStyle{Italic: true}
	tpw.statusLabel.Truncation = fyne.TextTruncateEllipsis

	// Иконка статуса
	tpw.icon = widget.NewIcon(nil)
	tpw.icon.Hide()

	// Кнопка отмены
	tpw.cancelButton = widget.NewButtonWithIcon("Отмена", theme.CancelIcon(), func() {
		if tpw.currentTransfer != nil && tpw.currentTransfer.TransferID != "" {
			if tpw.transferSvc != nil {
				_ = tpw.transferSvc.CancelTransfer(tpw.currentTransfer.TransferID)
			}
		}
	})
	tpw.cancelButton.Hide()
	tpw.cancelButton.Importance = widget.DangerImportance
	tpw.cancelButton.Alignment = widget.ButtonAlignTrailing

	// Компонуем прогресс и кнопку отмены в одну строку
	progressRow := container.NewBorder(
		nil,
		nil,
		nil,
		tpw.cancelButton,
		tpw.progressBar,
	)

	// Контейнер с фоном
	bg := canvas.NewRectangle(color.RGBA{R: 30, G: 30, B: 30, A: 255})
	bg.SetMinSize(fyne.NewSize(0, 60))

	// Основной контент
	content := container.NewVBox(
		container.NewHBox(tpw.icon, tpw.statusLabel),
		progressRow,
	)

	tpw.container = container.NewStack(bg, container.NewPadded(content))

	// Подписка на обновления прогресса
	if transferSvc != nil {
		go tpw.monitorProgress()
	}

	return tpw
}

// monitorProgress отслеживает прогресс передач
func (tpw *TransferProgressWidget) monitorProgress() {
	if tpw.transferSvc == nil {
		return
	}

	progressChan := tpw.transferSvc.ProgressChan()
	for progress := range progressChan {
		tpw.updateProgress(progress)
	}
}

// updateProgress обновляет UI
func (tpw *TransferProgressWidget) updateProgress(progress *transfer.TransferProgress) {
	// Пропускаем дубликаты
	if tpw.lastTransferID == progress.TransferID &&
		progress.Status == transfer.TransferStatusInProgress {
		return
	}

	tpw.lastTransferID = progress.TransferID
	tpw.currentTransfer = progress

	// Обновляем UI (в Fyne v2 обновления безопасны из любого потока)
	tpw.updateUI(progress)
}

// updateUI обновляет элементы интерфейса
func (tpw *TransferProgressWidget) updateUI(progress *transfer.TransferProgress) {
	// Показываем виджет
	tpw.progressBar.Show()
	tpw.statusLabel.Show()
	tpw.icon.Show()

	// Устанавливаем иконку в зависимости от статуса
	switch progress.Status {
	case transfer.TransferStatusInProgress:
		tpw.icon.SetResource(theme.DownloadIcon())
		tpw.cancelButton.Show()
		tpw.progressBar.SetValue(progress.Percent / 100.0)

		statusText := fmt.Sprintf("Передача: %s (%.1f%%)", progress.FileName, progress.Percent)
		tpw.statusLabel.SetText(statusText)

	case transfer.TransferStatusCompleted:
		tpw.icon.SetResource(theme.ConfirmIcon())
		tpw.cancelButton.Hide()
		tpw.progressBar.SetValue(1.0)

		statusText := fmt.Sprintf("✓ Завершено: %s", progress.FileName)
		tpw.statusLabel.SetText(statusText)

		// Скрыть через 3 секунды
		tpw.scheduleHide()

	case transfer.TransferStatusFailed:
		tpw.icon.SetResource(theme.ErrorIcon())
		tpw.cancelButton.Hide()

		statusText := fmt.Sprintf("✗ Ошибка: %s", progress.FileName)
		if progress.Error != "" {
			statusText += fmt.Sprintf(" (%s)", progress.Error)
		}
		tpw.statusLabel.SetText(statusText)

		// Скрыть через 5 секунд
		tpw.scheduleHide()

	case transfer.TransferStatusCancelled:
		tpw.icon.SetResource(theme.CancelIcon())
		tpw.cancelButton.Hide()

		statusText := fmt.Sprintf("✗ Отменено: %s", progress.FileName)
		tpw.statusLabel.SetText(statusText)

		// Скрыть через 3 секунды
		tpw.scheduleHide()

	case transfer.TransferStatusPending:
		tpw.icon.SetResource(theme.DownloadIcon())
		tpw.cancelButton.Show()

		statusText := fmt.Sprintf("Ожидание: %s", progress.FileName)
		tpw.statusLabel.SetText(statusText)
	}

	tpw.container.Refresh()
}

// scheduleHide планирует скрытие виджета через 3 секунды
func (tpw *TransferProgressWidget) scheduleHide() {
	time.AfterFunc(3*time.Second, func() {
		tpw.progressBar.Hide()
		tpw.statusLabel.SetText("")
		tpw.icon.Hide()
		tpw.container.Refresh()
	})
}

// Container возвращает контейнер виджета
func (tpw *TransferProgressWidget) Container() *fyne.Container {
	return tpw.container
}

// SetVisible устанавливает видимость виджета
func (tpw *TransferProgressWidget) SetVisible(visible bool) {
	if visible {
		tpw.container.Show()
	} else {
		tpw.container.Hide()
	}
}

// SetTransferService устанавливает сервис передачи
func (tpw *TransferProgressWidget) SetTransferService(transferSvc *transfer.Service) {
	tpw.transferSvc = transferSvc
	if transferSvc != nil {
		go tpw.monitorProgress()
	}
}
