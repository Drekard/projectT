package transfer

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"projectT/internal/metrics"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/storage/filesystem"
)

// SendBatch отправляет пакет элементов пиру
func (ts *Service) SendBatch(ctx context.Context, peerID peer.ID, elementUUIDs []string, batchType TransferType) (string, error) {
	if len(elementUUIDs) == 0 {
		return "", fmt.Errorf("пустой список элементов")
	}

	batchID := fmt.Sprintf("batch_%d", time.Now().UnixNano())

	var items []*models.Item
	for _, uuid := range elementUUIDs {
		item, err := queries.GetItemByElementUUID(uuid)
		if err != nil {
			log.Printf("[Batch] ⚠️ Элемент %s не найден: %v", uuid[:8], err)
			continue
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		return "", fmt.Errorf("не найдено ни одного элемента")
	}

	var totalSize int64
	for _, item := range items {
		if item.ContentMeta != "" {
			fileHash := extractFileHashFromContentMeta(item.ContentMeta)
			if fileHash != "" {
				fileInfo, err := filesystem.GetFileInfo(fileHash)
				if err == nil {
					totalSize += fileInfo.Size
				}
			}
		}
	}

	batchReq := &BatchTransferRequest{
		BatchID:      batchID,
		Type:         batchType,
		ElementUUIDs: elementUUIDs,
		TotalItems:   len(items),
		TotalSize:    totalSize,
		SourcePeerID: ts.host.ID().String(),
	}

	batchProg := &BatchProgress{
		BatchID:    batchID,
		Type:       string(batchType),
		TotalItems: len(items),
		Status:     TransferStatusPending,
	}
	ts.mu.Lock()
	ts.activeBatches[batchID] = batchProg
	ts.mu.Unlock()

	go ts.executeBatchSend(ctx, peerID, batchReq, items)

	return batchID, nil
}

// SendFolder отправляет все элементы папки пиру
func (ts *Service) SendFolder(ctx context.Context, peerID peer.ID, parentUUID string) (string, error) {
	items, err := queries.GetItemsByParentUUID(parentUUID)
	if err != nil {
		return "", fmt.Errorf("ошибка получения элементов папки: %w", err)
	}

	if len(items) == 0 {
		return "", fmt.Errorf("папка пуста")
	}

	var uuids []string
	for _, item := range items {
		uuids = append(uuids, item.ElementUUID)
	}

	return ts.SendBatch(ctx, peerID, uuids, TransferTypeBatchFolder)
}

// SendPinnedItems отправляет закреплённые элементы пиру
func (ts *Service) SendPinnedItems(ctx context.Context, peerID peer.ID) (string, error) {
	uuids, err := queries.GetPinnedItemUUIDs()
	if err != nil {
		return "", fmt.Errorf("ошибка получения закреплённых элементов: %w", err)
	}

	if len(uuids) == 0 {
		return "", fmt.Errorf("нет закреплённых элементов")
	}

	return ts.SendBatch(ctx, peerID, uuids, TransferTypeBatchPinned)
}

// SendSelection отправляет выбранные элементы пиру
func (ts *Service) SendSelection(ctx context.Context, peerID peer.ID, elementUUIDs []string) (string, error) {
	return ts.SendBatch(ctx, peerID, elementUUIDs, TransferTypeBatchSelection)
}

// executeBatchSend выполняет параллельную отправку элементов батча
func (ts *Service) executeBatchSend(ctx context.Context, peerID peer.ID, batchReq *BatchTransferRequest, items []*models.Item) {
	batchID := batchReq.BatchID

	ts.updateBatchStatus(batchID, TransferStatusInProgress, "")

	stream, err := ts.host.NewStream(ctx, peerID, BatchProtocolID)
	if err != nil {
		ts.updateBatchStatus(batchID, TransferStatusFailed, fmt.Sprintf("ошибка создания стрима: %v", err))
		return
	}
	defer func() { _ = stream.Close() }()

	writer := bufio.NewWriter(stream)
	encoder := json.NewEncoder(writer)
	reader := bufio.NewReader(stream)

	if err := encoder.Encode(batchReq); err != nil {
		ts.updateBatchStatus(batchID, TransferStatusFailed, fmt.Sprintf("ошибка отправки запроса: %v", err))
		return
	}
	if err := writer.Flush(); err != nil {
		ts.updateBatchStatus(batchID, TransferStatusFailed, fmt.Sprintf("ошибка flush: %v", err))
		return
	}

	var ack TransferAck
	if err := json.NewDecoder(reader).Decode(&ack); err != nil {
		ts.updateBatchStatus(batchID, TransferStatusFailed, fmt.Sprintf("ошибка чтения подтверждения: %v", err))
		return
	}
	if !ack.Success {
		ts.updateBatchStatus(batchID, TransferStatusFailed, fmt.Sprintf("получатель отклонил: %s", ack.Error))
		return
	}

	var completed, failed int
	var totalBytes, transferredBytes int64

	for i, item := range items {
		select {
		case <-ctx.Done():
			ts.updateBatchStatus(batchID, TransferStatusCancelled, "отменено")
			return
		default:
		}

		itemProg := &BatchItemProgress{
			BatchID:     batchID,
			ElementUUID: item.ElementUUID,
			Title:       item.Title,
			Index:       i,
			Total:       len(items),
			Status:      TransferStatusInProgress,
		}
		ts.sendBatchItemProgress(itemProg)

		itemResp := ts.itemToResponse(item)
		if err := encoder.Encode(itemResp); err != nil {
			log.Printf("[Batch] ❌ Ошибка отправки элемента %s: %v", item.ElementUUID[:8], err)
			itemProg.Status = TransferStatusFailed
			itemProg.Error = err.Error()
			ts.sendBatchItemProgress(itemProg)
			failed++
			continue
		}
		if err := writer.Flush(); err != nil {
			log.Printf("[Batch] ❌ Ошибка flush элемента %s: %v", item.ElementUUID[:8], err)
			itemProg.Status = TransferStatusFailed
			itemProg.Error = err.Error()
			ts.sendBatchItemProgress(itemProg)
			failed++
			continue
		}

		var itemAck TransferAck
		if err := json.NewDecoder(reader).Decode(&itemAck); err != nil {
			log.Printf("[Batch] ❌ Ошибка чтения подтверждения элемента %s: %v", item.ElementUUID[:8], err)
			itemProg.Status = TransferStatusFailed
			itemProg.Error = err.Error()
			ts.sendBatchItemProgress(itemProg)
			failed++
			continue
		}

		if itemAck.Success {
			itemProg.Status = TransferStatusCompleted
			itemProg.Transferred = itemAck.Received
			itemProg.FileSize = itemAck.Received
			totalBytes += itemAck.Received
			transferredBytes += itemAck.Received
		} else {
			itemProg.Status = TransferStatusFailed
			itemProg.Error = itemAck.Error
			failed++
		}
		ts.sendBatchItemProgress(itemProg)
		completed++
	}

	if failed == 0 {
		ts.updateBatchStatus(batchID, TransferStatusCompleted, "")
	} else if completed == 0 {
		ts.updateBatchStatus(batchID, TransferStatusFailed, fmt.Sprintf("все элементы не отправлены: %d ошибок", failed))
	} else {
		ts.updateBatchStatus(batchID, TransferStatusCompleted, fmt.Sprintf("частичный успех: %d успешно, %d ошибок", completed, failed))
	}

	_ = encoder.Encode(&TransferAck{Success: true})
	_ = writer.Flush()

	if metrics.IsInitialized() {
		m := metrics.Get().Metrics
		m.P2PTransferBytesTotal.Add(float64(transferredBytes))
		m.P2PFilesTransferred.Add(float64(completed))
	}

	log.Printf("[Batch] ✅ Батч %s завершён: %d успешно, %d ошибок", batchID, completed, failed)

	go func() {
		time.Sleep(5 * time.Second)
		ts.RemoveBatch(batchID)
	}()
}

// handleBatchTransferRequest обрабатывает входящий запрос пакетной передачи
func (ts *Service) handleBatchTransferRequest(stream network.Stream) {
	defer func() { _ = stream.Close() }()

	remotePeer := stream.Conn().RemotePeer()
	log.Printf("[Batch] 📥 Получен запрос батча от: %s", remotePeer.String())

	reader := bufio.NewReader(stream)
	writer := bufio.NewWriter(stream)
	encoder := json.NewEncoder(writer)

	var batchReq BatchTransferRequest
	if err := json.NewDecoder(reader).Decode(&batchReq); err != nil {
		log.Printf("[Batch] ❌ Ошибка чтения запроса: %v", err)
		_ = encoder.Encode(&TransferAck{Success: false, Error: err.Error()})
		_ = writer.Flush()
		return
	}

	log.Printf("[Batch] 📋 Батч %s: %d элементов, тип=%s", batchReq.BatchID, batchReq.TotalItems, batchReq.Type)

	_ = encoder.Encode(&TransferAck{Success: true})
	_ = writer.Flush()

	batchProg := &BatchProgress{
		BatchID:    batchReq.BatchID,
		Type:       string(batchReq.Type),
		TotalItems: batchReq.TotalItems,
		Status:     TransferStatusInProgress,
	}
	ts.mu.Lock()
	ts.activeBatches[batchReq.BatchID] = batchProg
	ts.mu.Unlock()

	var completed, failed int
	for i := 0; i < batchReq.TotalItems; i++ {
		var itemResp ItemResponse
		if err := json.NewDecoder(reader).Decode(&itemResp); err != nil {
			log.Printf("[Batch] ❌ Ошибка чтения элемента #%d: %v", i, err)
			failed++
			break
		}

		err := ts.saveBatchItem(remotePeer.String(), &itemResp)
		if err != nil {
			log.Printf("[Batch] ❌ Ошибка сохранения элемента %s: %v", itemResp.ElementUUID[:8], err)
			failed++
		} else {
			completed++
			log.Printf("[Batch] ✅ Элемент %s сохранён (%d/%d)", itemResp.ElementUUID[:8], completed, batchReq.TotalItems)
		}

		fileSize := int64(0)
		if itemResp.FileData != nil {
			fileSize = itemResp.FileData.Size
		}
		_ = encoder.Encode(&TransferAck{Success: err == nil, Received: fileSize})
		_ = writer.Flush()

		batchProg.Completed = completed
		batchProg.Failed = failed
		batchProg.OverallPercent = float64(completed+failed) / float64(batchReq.TotalItems) * 100
		ts.batchProgress <- batchProg
	}

	var finalAck TransferAck
	_ = json.NewDecoder(reader).Decode(&finalAck)

	if failed == 0 {
		batchProg.Status = TransferStatusCompleted
		log.Printf("[Batch] ✅ Батч %s от %s завершён: %d элементов", batchReq.BatchID, remotePeer, completed)
	} else {
		batchProg.Status = TransferStatusCompleted
		log.Printf("[Batch] ⚠️ Батч %s от %s: %d успешно, %d ошибок", batchReq.BatchID, remotePeer, completed, failed)
	}

	ts.batchProgress <- batchProg

	// Уведомляем UI о завершении получения батча
	go ts.NotifyBatchComplete(remotePeer.String())
}

// saveBatchItem сохраняет элемент из батча
func (ts *Service) saveBatchItem(sourcePeerID string, resp *ItemResponse) error {
	exists, err := queries.HasRemoteItem(sourcePeerID, resp.ElementUUID)
	if err != nil {
		return fmt.Errorf("ошибка проверки: %w", err)
	}

	if exists {
		existing, err := queries.GetRemoteItemByElementUUID(sourcePeerID, resp.ElementUUID)
		if err == nil && existing != nil {
			item := &models.Item{
				ID:           existing.ID,
				ElementUUID:  resp.ElementUUID,
				Hash:         resp.Hash,
				OwnerType:    models.OwnerTypeRemote,
				SourcePeerID: &sourcePeerID,
				Type:         resp.Type,
				Title:        resp.Title,
				Description:  resp.Description,
				ContentMeta:  resp.ContentMeta,
				ParentUUID:   resp.ParentUUID,
				Signature:    resp.Signature,
				Version:      existing.Version + 1,
				Status:       models.ItemStatusPreview,
				CreatedAt:    existing.CreatedAt,
				UpdatedAt:    time.Now(),
			}
			if err := queries.UpdateRemoteItem(item); err != nil {
				return fmt.Errorf("ошибка обновления: %w", err)
			}
		}
	} else {
		item := &models.Item{
			ElementUUID:  resp.ElementUUID,
			Hash:         resp.Hash,
			OwnerType:    models.OwnerTypeRemote,
			SourcePeerID: &sourcePeerID,
			Type:         resp.Type,
			Title:        resp.Title,
			Description:  resp.Description,
			ContentMeta:  resp.ContentMeta,
			ParentUUID:   resp.ParentUUID,
			Signature:    resp.Signature,
			Version:      1,
			Status:       models.ItemStatusPreview,
		}
		if err := queries.CreateRemoteItem(item); err != nil {
			return fmt.Errorf("ошибка создания: %w", err)
		}
	}

	if resp.FileData != nil && len(resp.FileData.Content) > 0 {
		_, err := filesystem.SaveFileWithOriginalName(resp.FileData.Content, "")
		if err != nil {
			log.Printf("[Batch] ⚠️ Не удалось сохранить файл: %v", err)
		}
	}

	return nil
}
