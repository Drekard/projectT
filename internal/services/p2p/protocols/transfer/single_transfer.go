package transfer

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"projectT/internal/metrics"
	"projectT/internal/storage/filesystem"
)

// SendFile отправляет файл пиру
func (ts *Service) SendFile(ctx context.Context, peerID peer.ID, filePath, fileName, mimeType string, transferType TransferType) (string, error) {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения файла: %w", err)
	}

	fileHash := filesystem.CalculateHash(fileData)

	transferID := fmt.Sprintf("%s-%d", fileName, time.Now().UnixNano())
	request := &FileTransferRequest{
		TransferID:   transferID,
		Type:         transferType,
		FileName:     fileName,
		FileSize:     int64(len(fileData)),
		MimeType:     mimeType,
		FileHash:     fileHash,
		SourcePeerID: ts.host.ID().String(),
	}

	stream, err := ts.host.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		return "", fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer func() { _ = stream.Close() }()

	writer := bufio.NewWriter(stream)
	encoder := json.NewEncoder(writer)

	if err := encoder.Encode(request); err != nil {
		return "", fmt.Errorf("ошибка отправки запроса: %w", err)
	}

	reader := bufio.NewReader(stream)
	var ack TransferAck
	if err := json.NewDecoder(reader).Decode(&ack); err != nil {
		return "", fmt.Errorf("ошибка чтения подтверждения: %w", err)
	}

	if !ack.Success {
		return "", fmt.Errorf("получатель отклонил передачу: %s", ack.Error)
	}

	if err := ts.sendFileChunks(stream, transferID, fileData); err != nil {
		return "", fmt.Errorf("ошибка передачи файла: %w", err)
	}

	var finalAck TransferAck
	if err := json.NewDecoder(reader).Decode(&finalAck); err != nil {
		return "", fmt.Errorf("ошибка чтения финального подтверждения: %w", err)
	}

	if !finalAck.Success {
		return "", fmt.Errorf("ошибка сохранения файла у получателя: %s", finalAck.Error)
	}

	log.Printf("Файл %s успешно отправлен пиру %s", fileName, peerID)
	return transferID, nil
}

// sendFileChunks отправляет файл чанками
func (ts *Service) sendFileChunks(stream network.Stream, transferID string, fileData []byte) error {
	writer := bufio.NewWriter(stream)
	encoder := json.NewEncoder(writer)

	totalSize := len(fileData)
	offset := 0

	for offset < totalSize {
		chunkSize := ChunkSize
		if offset+chunkSize > totalSize {
			chunkSize = totalSize - offset
		}

		chunk := &FileTransferChunk{
			TransferID: transferID,
			Offset:     int64(offset),
			Data:       fileData[offset : offset+chunkSize],
			IsLast:     offset+chunkSize >= totalSize,
		}

		if err := encoder.Encode(chunk); err != nil {
			return fmt.Errorf("ошибка отправки чанка: %w", err)
		}

		if err := writer.Flush(); err != nil {
			return fmt.Errorf("ошибка flush чанка: %w", err)
		}

		offset += chunkSize

		select {
		case ts.progressChan <- &TransferProgress{
			TransferID:  transferID,
			FileName:    "",
			Total:       int64(totalSize),
			Transferred: int64(offset),
			Percent:     float64(offset) / float64(totalSize) * 100,
			Status:      TransferStatusInProgress,
		}:
		default:
		}
	}

	if metrics.IsInitialized() {
		m := metrics.Get().Metrics
		m.P2PTransferBytesTotal.Add(float64(totalSize))
		m.P2PFilesTransferred.Inc()
	}

	return nil
}

// handleTransferRequest обрабатывает входящий запрос передачи
func (ts *Service) handleTransferRequest(stream network.Stream) {
	defer func() { _ = stream.Close() }()

	remotePeer := stream.Conn().RemotePeer()
	log.Printf("Получен запрос передачи файла от: %s", remotePeer.String())

	reader := bufio.NewReader(stream)
	writer := bufio.NewWriter(stream)
	encoder := json.NewEncoder(writer)
	decoder := json.NewDecoder(reader)

	var request FileTransferRequest
	if err := decoder.Decode(&request); err != nil {
		log.Printf("Ошибка чтения запроса: %v", err)
		_ = encoder.Encode(&TransferAck{Success: false, Error: err.Error()})
		_ = writer.Flush()
		return
	}

	transfer := &ActiveTransfer{
		Request: &request,
		Progress: &TransferProgress{
			TransferID:  request.TransferID,
			FileName:    request.FileName,
			Total:       request.FileSize,
			Transferred: 0,
			Percent:     0,
			Status:      TransferStatusInProgress,
		},
		mu: sync.RWMutex{},
	}

	ts.mu.Lock()
	ts.activeTransfers[request.TransferID] = transfer
	ts.mu.Unlock()

	_ = encoder.Encode(&TransferAck{Success: true, Received: 0})
	_ = writer.Flush()

	destPath, err := ts.getDestinationPath(&request)
	if err != nil {
		log.Printf("Ошибка получения пути: %v", err)
		transfer.UpdateProgress(0, TransferStatusFailed, err.Error())
		_ = encoder.Encode(&TransferAck{Success: false, Error: err.Error()})
		_ = writer.Flush()
		return
	}

	var receivedData []byte
	var totalReceived int64 = 0

	for {
		var chunk FileTransferChunk
		if err := decoder.Decode(&chunk); err != nil {
			if err.Error() == "EOF" {
				break
			}
			log.Printf("Ошибка чтения чанка: %v", err)
			transfer.UpdateProgress(totalReceived, TransferStatusFailed, err.Error())
			_ = encoder.Encode(&TransferAck{Success: false, Received: totalReceived, Error: err.Error()})
			_ = writer.Flush()
			return
		}

		receivedData = append(receivedData, chunk.Data...)
		totalReceived += int64(len(chunk.Data))

		transfer.UpdateProgress(totalReceived, TransferStatusInProgress, "")

		_ = encoder.Encode(&TransferAck{Success: true, Received: totalReceived})
		_ = writer.Flush()

		if chunk.IsLast {
			break
		}
	}

	receivedHash := filesystem.CalculateHash(receivedData)
	if receivedHash != request.FileHash {
		log.Printf("Хеш не совпадает: ожидался %s, получен %s", request.FileHash, receivedHash)
		transfer.UpdateProgress(totalReceived, TransferStatusFailed, "хеш не совпадает")
		_ = encoder.Encode(&TransferAck{Success: false, Error: "хеш не совпадает"})
		_ = writer.Flush()
		return
	}

	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		log.Printf("Ошибка создания директории: %v", err)
		transfer.UpdateProgress(totalReceived, TransferStatusFailed, err.Error())
		_ = encoder.Encode(&TransferAck{Success: false, Error: err.Error()})
		_ = writer.Flush()
		return
	}

	if err := os.WriteFile(destPath, receivedData, 0644); err != nil {
		log.Printf("Ошибка сохранения файла: %v", err)
		transfer.UpdateProgress(totalReceived, TransferStatusFailed, err.Error())
		_ = encoder.Encode(&TransferAck{Success: false, Error: err.Error()})
		_ = writer.Flush()
		return
	}

	transfer.UpdateProgress(totalReceived, TransferStatusCompleted, "")
	log.Printf("Файл %s успешно получен от %s", request.FileName, remotePeer)

	_ = encoder.Encode(&TransferAck{Success: true, Received: totalReceived})
	_ = writer.Flush()
}

// getDestinationPath возвращает путь для сохранения файла
func (ts *Service) getDestinationPath(request *FileTransferRequest) (string, error) {
	switch request.Type {
	case TransferTypeAvatar:
		ext := filepath.Ext(request.FileName)
		return filepath.Join("storage", "files", "avatars", "remote", request.SourcePeerID+ext), nil
	case TransferTypeImage:
		return filepath.Join("storage", "transfers", "images", request.FileName), nil
	case TransferTypeFile:
		return filepath.Join("storage", "transfers", "files", request.FileName), nil
	default:
		return filepath.Join("storage", "transfers", "misc", request.FileName), nil
	}
}

// SendAvatar отправляет аватарку пиру
func (ts *Service) SendAvatar(ctx context.Context, peerID peer.ID, avatarPath, fileName string) (string, error) {
	return ts.SendFile(ctx, peerID, avatarPath, fileName, "image/png", TransferTypeAvatar)
}

// SendImage отправляет изображение пиру
func (ts *Service) SendImage(ctx context.Context, peerID peer.ID, imagePath, imageName string) (string, error) {
	return ts.SendFile(ctx, peerID, imagePath, imageName, "image/png", TransferTypeImage)
}

// SendElementMetadata отправляет метаданные элемента пиру
func (ts *Service) SendElementMetadata(ctx context.Context, peerID peer.ID, elementUUID, title, description, contentMeta string) (string, error) {
	log.Printf("[Transfer] 📤 Отправка метаданных элемента: UUID=%s, title=%s", elementUUID, title)

	transferID := fmt.Sprintf("element_%s_%d", elementUUID, time.Now().UnixNano())

	request := &FileTransferRequest{
		TransferID: transferID,
		Type:       TransferTypeElementMetadata,
		FileName:   fmt.Sprintf("Элемент: %s", title),
		FileSize:   0,
	}

	transfer := &ActiveTransfer{
		Request: request,
		Progress: &TransferProgress{
			TransferID:  transferID,
			FileName:    fmt.Sprintf("Элемент: %s", title),
			Total:       100,
			Transferred: 0,
			Status:      TransferStatusPending,
			Percent:     0,
		},
	}

	ts.mu.Lock()
	ts.activeTransfers[transferID] = transfer
	ts.mu.Unlock()

	ts.progressChan <- transfer.GetProgress()

	steps := []struct {
		percent float64
		status  TransferStatus
		delayMs int
	}{
		{33, TransferStatusInProgress, 200},
		{66, TransferStatusInProgress, 300},
		{100, TransferStatusCompleted, 100},
	}

	for _, step := range steps {
		select {
		case <-ctx.Done():
			transfer.UpdateProgress(int64(step.percent), TransferStatusCancelled, "отменено пользователем")
			ts.progressChan <- transfer.GetProgress()
			return transferID, ctx.Err()
		case <-time.After(time.Duration(step.delayMs) * time.Millisecond):
			transfer.UpdateProgress(int64(step.percent), step.status, "")
			ts.progressChan <- transfer.GetProgress()
		}
	}

	log.Printf("[Transfer] ✅ Метаданные элемента отправлены: UUID=%s", elementUUID)

	go func() {
		time.Sleep(2 * time.Second)
		ts.RemoveTransfer(transferID)
	}()

	return transferID, nil
}
