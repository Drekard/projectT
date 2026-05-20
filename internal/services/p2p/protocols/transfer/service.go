package transfer

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
)

// Service сервис для передачи файлов между пирами
type Service struct {
	host            host.Host
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	activeTransfers map[string]*ActiveTransfer
	progressChan    chan *TransferProgress
	batchProgress   chan *BatchProgress
	batchItemProg   chan *BatchItemProgress
	activeBatches   map[string]*BatchProgress
}

// NewService создаёт сервис передачи файлов
func NewService(host host.Host) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		host:            host,
		ctx:             ctx,
		cancel:          cancel,
		activeTransfers: make(map[string]*ActiveTransfer),
		progressChan:    make(chan *TransferProgress, 100),
		batchProgress:   make(chan *BatchProgress, 50),
		batchItemProg:   make(chan *BatchItemProgress, 200),
		activeBatches:   make(map[string]*BatchProgress),
	}
}

// Start запускает сервис передачи файлов
func (ts *Service) Start() error {
	ts.host.SetStreamHandler(ProtocolID, ts.handleTransferRequest)
	ts.host.SetStreamHandler(BatchProtocolID, ts.handleBatchTransferRequest)
	return nil
}

// Stop останавливает сервис
func (ts *Service) Stop() error {
	ts.cancel()
	return nil
}

// GetProgress возвращает прогресс активной передачи
func (ts *Service) GetProgress(transferID string) *TransferProgress {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if transfer, exists := ts.activeTransfers[transferID]; exists {
		return transfer.GetProgress()
	}

	return &TransferProgress{
		TransferID: transferID,
		Status:     TransferStatusPending,
	}
}

// CancelTransfer отменяет активную передачу
func (ts *Service) CancelTransfer(transferID string) error {
	ts.mu.RLock()
	transfer, exists := ts.activeTransfers[transferID]
	ts.mu.RUnlock()

	if !exists {
		return fmt.Errorf("передача не найдена")
	}

	if transfer.Cancel != nil {
		transfer.Cancel()
	}

	transfer.UpdateProgress(transfer.Progress.Transferred, TransferStatusCancelled, "")
	log.Printf("Передача %s отменена", transferID)
	return nil
}

// ProgressChan возвращает канал для получения уведомлений о прогрессе
func (ts *Service) ProgressChan() <-chan *TransferProgress {
	return ts.progressChan
}

// RemoveTransfer удаляет информацию о завершённой передаче
func (ts *Service) RemoveTransfer(transferID string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	delete(ts.activeTransfers, transferID)
}

// BatchProgressChan возвращает канал для получения уведомлений о прогрессе батчей
func (ts *Service) BatchProgressChan() <-chan *BatchProgress {
	return ts.batchProgress
}

// BatchItemProgressChan возвращает канал для получения уведомлений о прогрессе элементов батча
func (ts *Service) BatchItemProgressChan() <-chan *BatchItemProgress {
	return ts.batchItemProg
}

// GetBatchProgress возвращает прогресс батча
func (ts *Service) GetBatchProgress(batchID string) *BatchProgress {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if batch, exists := ts.activeBatches[batchID]; exists {
		return &BatchProgress{
			BatchID:          batch.BatchID,
			Type:             batch.Type,
			TotalItems:       batch.TotalItems,
			Completed:        batch.Completed,
			Failed:           batch.Failed,
			TotalBytes:       batch.TotalBytes,
			TransferredBytes: batch.TransferredBytes,
			OverallPercent:   batch.OverallPercent,
			Status:           batch.Status,
			CurrentItem:      batch.CurrentItem,
		}
	}
	return &BatchProgress{
		BatchID: batchID,
		Status:  TransferStatusPending,
	}
}

// RemoveBatch удаляет информацию о завершённом батче
func (ts *Service) RemoveBatch(batchID string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.activeBatches, batchID)
}

// StartReceiveBatch начинает трекинг входящего батча
func (ts *Service) StartReceiveBatch(batchID string, totalItems int, totalBytes int64) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	batchProg := &BatchProgress{
		BatchID:    batchID,
		Type:       "receive",
		TotalItems: totalItems,
		Status:     TransferStatusInProgress,
		TotalBytes: totalBytes,
	}
	ts.activeBatches[batchID] = batchProg

	// Отправляем начальное состояние
	select {
	case ts.batchProgress <- batchProg:
	default:
	}
}

// UpdateReceiveBatchItem обновляет прогресс по одному элементу входящего батча
func (ts *Service) UpdateReceiveBatchItem(batchID string, itemTitle string, completed int, failed int) {
	ts.mu.Lock()
	batchProg, exists := ts.activeBatches[batchID]
	if !exists {
		ts.mu.Unlock()
		return
	}

	batchProg.Completed = completed
	batchProg.Failed = failed
	batchProg.CurrentItem = itemTitle
	if batchProg.TotalItems > 0 {
		batchProg.OverallPercent = float64(completed) / float64(batchProg.TotalItems) * 100
	}

	// Копируем для отправки
	progCopy := &BatchProgress{
		BatchID:          batchProg.BatchID,
		Type:             batchProg.Type,
		TotalItems:       batchProg.TotalItems,
		Completed:        batchProg.Completed,
		Failed:           batchProg.Failed,
		TotalBytes:       batchProg.TotalBytes,
		TransferredBytes: batchProg.TransferredBytes,
		OverallPercent:   batchProg.OverallPercent,
		Status:           batchProg.Status,
		CurrentItem:      batchProg.CurrentItem,
	}
	ts.mu.Unlock()

	select {
	case ts.batchProgress <- progCopy:
	default:
	}
}

// CompleteReceiveBatch завершает трекинг входящего батча
func (ts *Service) CompleteReceiveBatch(batchID string, status TransferStatus, errMsg string) {
	ts.mu.Lock()
	batchProg, exists := ts.activeBatches[batchID]
	if !exists {
		ts.mu.Unlock()
		return
	}

	batchProg.Status = status
	batchProg.Error = errMsg
	if status == TransferStatusCompleted {
		batchProg.Completed = batchProg.TotalItems
		batchProg.OverallPercent = 100
	}

	progCopy := &BatchProgress{
		BatchID:          batchProg.BatchID,
		Type:             batchProg.Type,
		TotalItems:       batchProg.TotalItems,
		Completed:        batchProg.Completed,
		Failed:           batchProg.Failed,
		TotalBytes:       batchProg.TotalBytes,
		TransferredBytes: batchProg.TransferredBytes,
		OverallPercent:   batchProg.OverallPercent,
		Status:           batchProg.Status,
		CurrentItem:      batchProg.CurrentItem,
		Error:            batchProg.Error,
	}
	ts.mu.Unlock()

	select {
	case ts.batchProgress <- progCopy:
	default:
	}
}
