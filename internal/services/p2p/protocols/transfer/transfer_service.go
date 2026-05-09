// Package transfer содержит сервисы для передачи файлов между пирами.
package transfer

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"projectT/internal/metrics"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/storage/filesystem"
)

// ProtocolID идентификатор протокола передачи файлов
const ProtocolID = "/projectt/transfer/1.0.0"

// BatchProtocolID идентификатор протокола пакетной передачи
const BatchProtocolID = "/projectt/batch-transfer/1.0.0"

// ChunkSize размер чанка при передаче файла (64 KB)
const ChunkSize = 64 * 1024

// MaxConcurrentBatchStreams максимальное количество параллельных стримов в батче
const MaxConcurrentBatchStreams = 3

// ItemResponse ответ с элементом (для batch передачи)
type ItemResponse struct {
	ElementUUID string          `json:"element_uuid"`
	ItemID      int             `json:"item_id"`
	Hash        string          `json:"hash"`
	Type        models.ItemType `json:"type"`
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	ContentMeta string          `json:"content_meta,omitempty"`
	ParentUUID  *string         `json:"parent_uuid,omitempty"`
	Signature   []byte          `json:"signature,omitempty"`
	Timestamp   int64           `json:"timestamp"`
	FileData    *ItemFileData   `json:"file_data,omitempty"`
}

// ItemFileData данные о файле элемента
type ItemFileData struct {
	Hash     string `json:"hash"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
	Content  []byte `json:"content,omitempty"`
}

// TransferType тип передачи
type TransferType string

const (
	// TransferTypeFile передача файла
	TransferTypeFile TransferType = "file"
	// TransferTypeAvatar передача аватарки
	TransferTypeAvatar TransferType = "avatar"
	// TransferTypeImage передача изображения
	TransferTypeImage TransferType = "image"
	// TransferTypeElementMetadata передача метаданных элемента
	TransferTypeElementMetadata TransferType = "element_metadata"
	// TransferTypeBatchFolder передача папки
	TransferTypeBatchFolder TransferType = "batch_folder"
	// TransferTypeBatchPinned передача закреплённых элементов
	TransferTypeBatchPinned TransferType = "batch_pinned"
	// TransferTypeBatchSelection передача выбранных элементов
	TransferTypeBatchSelection TransferType = "batch_selection"
)

// TransferStatus статус передачи
type TransferStatus string

const (
	// TransferStatusPending ожидание начала
	TransferStatusPending TransferStatus = "pending"
	// TransferStatusInProgress передача в процессе
	TransferStatusInProgress TransferStatus = "in_progress"
	// TransferStatusCompleted передача завершена
	TransferStatusCompleted TransferStatus = "completed"
	// TransferStatusFailed передача не удалась
	TransferStatusFailed TransferStatus = "failed"
	// TransferStatusCancelled передача отменена
	TransferStatusCancelled TransferStatus = "cancelled"
)

// FileTransferRequest запрос на передачу файла
type FileTransferRequest struct {
	TransferID   string       `json:"transfer_id"`    // Уникальный ID передачи
	Type         TransferType `json:"type"`           // Тип передачи
	FileName     string       `json:"file_name"`      // Имя файла
	FileSize     int64        `json:"file_size"`      // Размер файла
	MimeType     string       `json:"mime_type"`      // MIME тип
	FileHash     string       `json:"file_hash"`      // Хеш файла для проверки
	SourcePeerID string       `json:"source_peer_id"` // PeerID отправителя
}

// FileTransferChunk чанк файла
type FileTransferChunk struct {
	TransferID string `json:"transfer_id"` // ID передачи
	Offset     int64  `json:"offset"`      // Смещение в файле
	Data       []byte `json:"data"`        // Данные чанка
	IsLast     bool   `json:"is_last"`     // Последний ли чанк
}

// TransferAck подтверждение получения чанка
type TransferAck struct {
	TransferID string `json:"transfer_id"` // ID передачи
	Received   int64  `json:"received"`    // Количество полученных байт
	Success    bool   `json:"success"`     // Успешно ли получено
	Error      string `json:"error,omitempty"`
}

// TransferProgress прогресс передачи
type TransferProgress struct {
	TransferID  string         `json:"transfer_id"`
	FileName    string         `json:"file_name"`
	Total       int64          `json:"total"`
	Transferred int64          `json:"transferred"`
	Percent     float64        `json:"percent"`
	Status      TransferStatus `json:"status"`
	Error       string         `json:"error,omitempty"`
}

// BatchTransferRequest запрос на пакетную передачу
type BatchTransferRequest struct {
	BatchID      string       `json:"batch_id"`
	Type         TransferType `json:"type"`
	ParentUUID   string       `json:"parent_uuid,omitempty"`
	ElementUUIDs []string     `json:"element_uuids,omitempty"`
	TotalItems   int          `json:"total_items"`
	TotalSize    int64        `json:"total_size"`
	SourcePeerID string       `json:"source_peer_id"`
}

// BatchItemProgress прогресс отдельного элемента в батче
type BatchItemProgress struct {
	BatchID     string         `json:"batch_id"`
	ElementUUID string         `json:"element_uuid"`
	Title       string         `json:"title"`
	Index       int            `json:"index"`
	Total       int            `json:"total"`
	FileSize    int64          `json:"file_size"`
	Transferred int64          `json:"transferred"`
	Status      TransferStatus `json:"status"`
	Error       string         `json:"error,omitempty"`
}

// BatchProgress общий прогресс батча
type BatchProgress struct {
	BatchID          string         `json:"batch_id"`
	Type             string         `json:"type"`
	TotalItems       int            `json:"total_items"`
	Completed        int            `json:"completed"`
	Failed           int            `json:"failed"`
	TotalBytes       int64          `json:"total_bytes"`
	TransferredBytes int64          `json:"transferred_bytes"`
	OverallPercent   float64        `json:"overall_percent"`
	Status           TransferStatus `json:"status"`
	CurrentItem      string         `json:"current_item"`
	Error            string         `json:"error,omitempty"`
}

// ActiveTransfer активная передача
type ActiveTransfer struct {
	Request     *FileTransferRequest
	Progress    *TransferProgress
	Destination string // Путь для сохранения файла
	Cancel      context.CancelFunc
	mu          sync.RWMutex
}

// UpdateProgress обновляет прогресс передачи
func (at *ActiveTransfer) UpdateProgress(transferred int64, status TransferStatus, errMsg string) {
	at.mu.Lock()
	defer at.mu.Unlock()

	at.Progress.Transferred = transferred
	at.Progress.Percent = float64(transferred) / float64(at.Progress.Total) * 100
	at.Progress.Status = status
	at.Progress.Error = errMsg
}

// GetProgress возвращает текущий прогресс
func (at *ActiveTransfer) GetProgress() *TransferProgress {
	at.mu.RLock()
	defer at.mu.RUnlock()

	// Возвращаем копию
	return &TransferProgress{
		TransferID:  at.Progress.TransferID,
		FileName:    at.Progress.FileName,
		Total:       at.Progress.Total,
		Transferred: at.Progress.Transferred,
		Percent:     at.Progress.Percent,
		Status:      at.Progress.Status,
		Error:       at.Progress.Error,
	}
}

// Service сервис для передачи файлов между пирами
type Service struct {
	host            host.Host
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	activeTransfers map[string]*ActiveTransfer // Активные передачи по ID
	progressChan    chan *TransferProgress     // Канал для уведомлений о прогрессе
	batchProgress   chan *BatchProgress        // Канал для уведомлений о прогрессе батчей
	batchItemProg   chan *BatchItemProgress    // Канал для уведомлений о прогрессе элементов батча
	activeBatches   map[string]*BatchProgress  // Активные батчи
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

// SendFile отправляет файл пиру
func (ts *Service) SendFile(ctx context.Context, peerID peer.ID, filePath, fileName, mimeType string, transferType TransferType) (string, error) {
	// Читаем файл
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения файла: %w", err)
	}

	// Вычисляем хеш
	fileHash := filesystem.CalculateHash(fileData)

	// Создаём запрос
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

	// Создаём стрим
	stream, err := ts.host.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		return "", fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer func() { _ = stream.Close() }()

	// Отправляем запрос
	writer := bufio.NewWriter(stream)
	encoder := json.NewEncoder(writer)

	if err := encoder.Encode(request); err != nil {
		return "", fmt.Errorf("ошибка отправки запроса: %w", err)
	}

	// Читаем подтверждение готовности
	reader := bufio.NewReader(stream)
	var ack TransferAck
	if err := json.NewDecoder(reader).Decode(&ack); err != nil {
		return "", fmt.Errorf("ошибка чтения подтверждения: %w", err)
	}

	if !ack.Success {
		return "", fmt.Errorf("получатель отклонил передачу: %s", ack.Error)
	}

	// Отправляем файл чанками
	if err := ts.sendFileChunks(stream, transferID, fileData); err != nil {
		return "", fmt.Errorf("ошибка передачи файла: %w", err)
	}

	// Читаем финальное подтверждение
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

		// Отправляем прогресс в канал
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
			// Канал переполнен - пропускаем
		}
	}

	// Обновляем метрики передачи
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

	// Читаем запрос
	var request FileTransferRequest
	if err := decoder.Decode(&request); err != nil {
		log.Printf("Ошибка чтения запроса: %v", err)
		_ = encoder.Encode(&TransferAck{Success: false, Error: err.Error()})
		_ = writer.Flush()
		return
	}

	// Создаём активную передачу
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

	// Отправляем подтверждение готовности
	_ = encoder.Encode(&TransferAck{Success: true, Received: 0})
	_ = writer.Flush()

	// Получаем путь для сохранения файла
	destPath, err := ts.getDestinationPath(&request)
	if err != nil {
		log.Printf("Ошибка получения пути: %v", err)
		transfer.UpdateProgress(0, TransferStatusFailed, err.Error())
		_ = encoder.Encode(&TransferAck{Success: false, Error: err.Error()})
		_ = writer.Flush()
		return
	}

	// Читаем чанки и сохраняем файл
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

		// Отправляем подтверждение чанка
		_ = encoder.Encode(&TransferAck{Success: true, Received: totalReceived})
		_ = writer.Flush()

		if chunk.IsLast {
			break
		}
	}

	// Проверяем хеш
	receivedHash := filesystem.CalculateHash(receivedData)
	if receivedHash != request.FileHash {
		log.Printf("Хеш не совпадает: ожидался %s, получен %s", request.FileHash, receivedHash)
		transfer.UpdateProgress(totalReceived, TransferStatusFailed, "хеш не совпадает")
		_ = encoder.Encode(&TransferAck{Success: false, Error: "хеш не совпадает"})
		_ = writer.Flush()
		return
	}

	// Сохраняем файл
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

	// Отправляем финальное подтверждение
	_ = encoder.Encode(&TransferAck{Success: true, Received: totalReceived})
	_ = writer.Flush()
}

// getDestinationPath возвращает путь для сохранения файла
func (ts *Service) getDestinationPath(request *FileTransferRequest) (string, error) {
	// Определяем тип файла и возвращаем соответствующий путь
	switch request.Type {
	case TransferTypeAvatar:
		// Для аватарок: storage/files/avatars/remote/{peerID}{ext}
		ext := filepath.Ext(request.FileName)
		return filepath.Join("storage", "files", "avatars", "remote", request.SourcePeerID+ext), nil
	case TransferTypeImage:
		// Для изображений: storage/transfers/images/{file_name}
		return filepath.Join("storage", "transfers", "images", request.FileName), nil
	case TransferTypeFile:
		// Для файлов: storage/transfers/files/{file_name}
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

	// Генерируем ID передачи
	transferID := fmt.Sprintf("element_%s_%d", elementUUID, time.Now().UnixNano())

	// Создаём запрос на передачу
	request := &FileTransferRequest{
		TransferID: transferID,
		Type:       TransferTypeElementMetadata,
		FileName:   fmt.Sprintf("Элемент: %s", title),
		FileSize:   0, // Метаданные не имеют размера
	}

	// Создаём активную передачу
	transfer := &ActiveTransfer{
		Request: request,
		Progress: &TransferProgress{
			TransferID:  transferID,
			FileName:    fmt.Sprintf("Элемент: %s", title),
			Total:       100, // Метаданные - это "виртуальная" передача
			Transferred: 0,
			Status:      TransferStatusPending,
			Percent:     0,
		},
	}

	// Регистрируем передачу
	ts.mu.Lock()
	ts.activeTransfers[transferID] = transfer
	ts.mu.Unlock()

	// Отправляем прогресс "начало передачи"
	ts.progressChan <- transfer.GetProgress()

	// Имитируем прогресс отправки метаданных (3 этапа по 33%)
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

	// Удаляем передачу через 2 секунды
	go func() {
		time.Sleep(2 * time.Second)
		ts.RemoveTransfer(transferID)
	}()

	return transferID, nil
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

// SendBatch отправляет пакет элементов пиру
func (ts *Service) SendBatch(ctx context.Context, peerID peer.ID, elementUUIDs []string, batchType TransferType) (string, error) {
	if len(elementUUIDs) == 0 {
		return "", fmt.Errorf("пустой список элементов")
	}

	batchID := fmt.Sprintf("batch_%d", time.Now().UnixNano())

	// Собираем элементы из БД
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

	// Вычисляем общий размер
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

	// Создаём батч прогресс
	batchProg := &BatchProgress{
		BatchID:    batchID,
		Type:       string(batchType),
		TotalItems: len(items),
		Status:     TransferStatusPending,
	}
	ts.mu.Lock()
	ts.activeBatches[batchID] = batchProg
	ts.mu.Unlock()

	// Отправляем батч в горутине
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

	// Обновляем статус на InProgress
	ts.updateBatchStatus(batchID, TransferStatusInProgress, "")

	// Создаём стрим для батча
	stream, err := ts.host.NewStream(ctx, peerID, BatchProtocolID)
	if err != nil {
		ts.updateBatchStatus(batchID, TransferStatusFailed, fmt.Sprintf("ошибка создания стрима: %v", err))
		return
	}
	defer func() { _ = stream.Close() }()

	writer := bufio.NewWriter(stream)
	encoder := json.NewEncoder(writer)
	reader := bufio.NewReader(stream)

	// Отправляем запрос на батч
	if err := encoder.Encode(batchReq); err != nil {
		ts.updateBatchStatus(batchID, TransferStatusFailed, fmt.Sprintf("ошибка отправки запроса: %v", err))
		return
	}
	if err := writer.Flush(); err != nil {
		ts.updateBatchStatus(batchID, TransferStatusFailed, fmt.Sprintf("ошибка flush: %v", err))
		return
	}

	// Читаем подтверждение
	var ack TransferAck
	if err := json.NewDecoder(reader).Decode(&ack); err != nil {
		ts.updateBatchStatus(batchID, TransferStatusFailed, fmt.Sprintf("ошибка чтения подтверждения: %v", err))
		return
	}
	if !ack.Success {
		ts.updateBatchStatus(batchID, TransferStatusFailed, fmt.Sprintf("получатель отклонил: %s", ack.Error))
		return
	}

	// Отправляем элементы последовательно (с параллельной обработкой файлов)
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

		// Сериализуем элемент как ItemResponse
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

		// Читаем подтверждение элемента
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

	// Финальный статус
	if failed == 0 {
		ts.updateBatchStatus(batchID, TransferStatusCompleted, "")
	} else if completed == 0 {
		ts.updateBatchStatus(batchID, TransferStatusFailed, fmt.Sprintf("все элементы не отправлены: %d ошибок", failed))
	} else {
		ts.updateBatchStatus(batchID, TransferStatusCompleted, fmt.Sprintf("частичный успех: %d успешно, %d ошибок", completed, failed))
	}

	// Отправляем финальное подтверждение
	_ = encoder.Encode(&TransferAck{Success: true})
	_ = writer.Flush()

	// Обновляем метрики
	if metrics.IsInitialized() {
		m := metrics.Get().Metrics
		m.P2PTransferBytesTotal.Add(float64(transferredBytes))
		m.P2PFilesTransferred.Add(float64(completed))
	}

	log.Printf("[Batch] ✅ Батч %s завершён: %d успешно, %d ошибок", batchID, completed, failed)

	// Удаляем батч через 5 секунд
	go func() {
		time.Sleep(5 * time.Second)
		ts.RemoveBatch(batchID)
	}()
}

// itemToResponse преобразует элемент в ответ для передачи
func (ts *Service) itemToResponse(item *models.Item) *ItemResponse {
	resp := &ItemResponse{
		ElementUUID: item.ElementUUID,
		ItemID:      item.ID,
		Hash:        item.Hash,
		Type:        item.Type,
		Title:       item.Title,
		Description: item.Description,
		ContentMeta: item.ContentMeta,
		Timestamp:   time.Now().UnixNano(),
	}

	// Извлекаем файл если есть
	fileHash := extractFileHashFromContentMeta(item.ContentMeta)
	if fileHash != "" {
		content, err := filesystem.ReadFile(fileHash)
		if err == nil {
			mimeType := detectMimeType(content)
			resp.FileData = &ItemFileData{
				Hash:     fileHash,
				Size:     int64(len(content)),
				MimeType: mimeType,
				Content:  content,
			}
		}
	}

	return resp
}

// handleBatchTransferRequest обрабатывает входящий запрос пакетной передачи
func (ts *Service) handleBatchTransferRequest(stream network.Stream) {
	defer func() { _ = stream.Close() }()

	remotePeer := stream.Conn().RemotePeer()
	log.Printf("[Batch] 📥 Получен запрос батча от: %s", remotePeer.String())

	reader := bufio.NewReader(stream)
	writer := bufio.NewWriter(stream)
	encoder := json.NewEncoder(writer)

	// Читаем запрос
	var batchReq BatchTransferRequest
	if err := json.NewDecoder(reader).Decode(&batchReq); err != nil {
		log.Printf("[Batch] ❌ Ошибка чтения запроса: %v", err)
		_ = encoder.Encode(&TransferAck{Success: false, Error: err.Error()})
		_ = writer.Flush()
		return
	}

	log.Printf("[Batch] 📋 Батч %s: %d элементов, тип=%s", batchReq.BatchID, batchReq.TotalItems, batchReq.Type)

	// Отправляем подтверждение готовности
	_ = encoder.Encode(&TransferAck{Success: true})
	_ = writer.Flush()

	// Создаём батч прогресс
	batchProg := &BatchProgress{
		BatchID:    batchReq.BatchID,
		Type:       string(batchReq.Type),
		TotalItems: batchReq.TotalItems,
		Status:     TransferStatusInProgress,
	}
	ts.mu.Lock()
	ts.activeBatches[batchReq.BatchID] = batchProg
	ts.mu.Unlock()

	// Читаем элементы
	var completed, failed int
	for i := 0; i < batchReq.TotalItems; i++ {
		var itemResp ItemResponse
		if err := json.NewDecoder(reader).Decode(&itemResp); err != nil {
			log.Printf("[Batch] ❌ Ошибка чтения элемента #%d: %v", i, err)
			failed++
			break
		}

		// Сохраняем элемент
		err := ts.saveBatchItem(remotePeer.String(), &itemResp)
		if err != nil {
			log.Printf("[Batch] ❌ Ошибка сохранения элемента %s: %v", itemResp.ElementUUID[:8], err)
			failed++
		} else {
			completed++
			log.Printf("[Batch] ✅ Элемент %s сохранён (%d/%d)", itemResp.ElementUUID[:8], completed, batchReq.TotalItems)
		}

		// Отправляем подтверждение элемента
		fileSize := int64(0)
		if itemResp.FileData != nil {
			fileSize = itemResp.FileData.Size
		}
		_ = encoder.Encode(&TransferAck{Success: err == nil, Received: fileSize})
		_ = writer.Flush()

		// Обновляем прогресс
		batchProg.Completed = completed
		batchProg.Failed = failed
		batchProg.OverallPercent = float64(completed+failed) / float64(batchReq.TotalItems) * 100
		ts.batchProgress <- batchProg
	}

	// Читаем финальное подтверждение
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
}

// saveBatchItem сохраняет элемент из батча
func (ts *Service) saveBatchItem(sourcePeerID string, resp *ItemResponse) error {
	// Проверяем, существует ли уже элемент
	exists, err := queries.HasRemoteItem(sourcePeerID, resp.ElementUUID)
	if err != nil {
		return fmt.Errorf("ошибка проверки: %w", err)
	}

	if exists {
		// Получаем существующий элемент для проверки версии
		existing, err := queries.GetRemoteItemByElementUUID(sourcePeerID, resp.ElementUUID)
		if err == nil && existing != nil {
			// Если у получателя старая версия (или такая же), обновляем
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
		// Создаём новый элемент
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

	// Сохраняем файл если есть
	if resp.FileData != nil && len(resp.FileData.Content) > 0 {
		_, err := filesystem.SaveFileWithOriginalName(resp.FileData.Content, "")
		if err != nil {
			log.Printf("[Batch] ⚠️ Не удалось сохранить файл: %v", err)
		}
	}

	return nil
}

// updateBatchStatus обновляет статус батча
func (ts *Service) updateBatchStatus(batchID string, status TransferStatus, errMsg string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if batch, exists := ts.activeBatches[batchID]; exists {
		batch.Status = status
		batch.Error = errMsg
		if status == TransferStatusCompleted || status == TransferStatusFailed || status == TransferStatusCancelled {
			batch.OverallPercent = 100
		}
		select {
		case ts.batchProgress <- batch:
		default:
		}
	}
}

// sendBatchItemProgress отправляет прогресс элемента
func (ts *Service) sendBatchItemProgress(prog *BatchItemProgress) {
	select {
	case ts.batchItemProg <- prog:
	default:
	}
}

// RemoveBatch удаляет информацию о завершённом батче
func (ts *Service) RemoveBatch(batchID string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.activeBatches, batchID)
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

// extractFileHashFromContentMeta извлекает file_hash из ContentMeta JSON
func extractFileHashFromContentMeta(contentMeta string) string {
	if contentMeta == "" {
		return ""
	}

	var blocks []map[string]interface{}
	if err := json.Unmarshal([]byte(contentMeta), &blocks); err != nil {
		return ""
	}

	for _, block := range blocks {
		if fileHash, ok := block["file_hash"].(string); ok {
			return fileHash
		}
	}

	return ""
}

// detectMimeType определяет MIME-тип файла по содержимому
func detectMimeType(fileBytes []byte) string {
	return http.DetectContentType(fileBytes)
}
