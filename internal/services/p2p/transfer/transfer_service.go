// Package transfer содержит сервисы для передачи файлов между пирами.
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

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"projectT/internal/storage/filesystem"
)

// ProtocolID идентификатор протокола передачи файлов
const ProtocolID = "/projectt/transfer/1.0.0"

// ChunkSize размер чанка при передаче файла (64 KB)
const ChunkSize = 64 * 1024

// TransferType тип передачи
type TransferType string

const (
	// TransferTypeFile передача файла
	TransferTypeFile TransferType = "file"
	// TransferTypeAvatar передача аватарки
	TransferTypeAvatar TransferType = "avatar"
	// TransferTypeImage передача изображения
	TransferTypeImage TransferType = "image"
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
	}
}

// Start запускает сервис передачи файлов
func (ts *Service) Start() error {
	ts.host.SetStreamHandler(ProtocolID, ts.handleTransferRequest)
	log.Println("TransferService запущен")
	return nil
}

// Stop останавливает сервис
func (ts *Service) Stop() error {
	ts.cancel()
	log.Println("TransferService остановлен")
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
	defer stream.Close()

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

	return nil
}

// handleTransferRequest обрабатывает входящий запрос передачи
func (ts *Service) handleTransferRequest(stream network.Stream) {
	defer stream.Close()

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
		writer.Flush()
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
	writer.Flush()

	// Получаем путь для сохранения файла
	destPath, err := ts.getDestinationPath(&request)
	if err != nil {
		log.Printf("Ошибка получения пути: %v", err)
		transfer.UpdateProgress(0, TransferStatusFailed, err.Error())
		_ = encoder.Encode(&TransferAck{Success: false, Error: err.Error()})
		writer.Flush()
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
			writer.Flush()
			return
		}

		receivedData = append(receivedData, chunk.Data...)
		totalReceived += int64(len(chunk.Data))

		transfer.UpdateProgress(totalReceived, TransferStatusInProgress, "")

		// Отправляем подтверждение чанка
		_ = encoder.Encode(&TransferAck{Success: true, Received: totalReceived})
		writer.Flush()

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
		writer.Flush()
		return
	}

	// Сохраняем файл
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		log.Printf("Ошибка создания директории: %v", err)
		transfer.UpdateProgress(totalReceived, TransferStatusFailed, err.Error())
		_ = encoder.Encode(&TransferAck{Success: false, Error: err.Error()})
		writer.Flush()
		return
	}

	if err := os.WriteFile(destPath, receivedData, 0644); err != nil {
		log.Printf("Ошибка сохранения файла: %v", err)
		transfer.UpdateProgress(totalReceived, TransferStatusFailed, err.Error())
		_ = encoder.Encode(&TransferAck{Success: false, Error: err.Error()})
		writer.Flush()
		return
	}

	transfer.UpdateProgress(totalReceived, TransferStatusCompleted, "")
	log.Printf("Файл %s успешно получен от %s", request.FileName, remotePeer)

	// Отправляем финальное подтверждение
	_ = encoder.Encode(&TransferAck{Success: true, Received: totalReceived})
	writer.Flush()
}

// getDestinationPath возвращает путь для сохранения файла
func (ts *Service) getDestinationPath(request *FileTransferRequest) (string, error) {
	// Определяем тип файла и возвращаем соответствующий путь
	switch request.Type {
	case TransferTypeAvatar:
		// Для аватарок: storage/avatars/{peerID}{ext}
		ext := filepath.Ext(request.FileName)
		return filepath.Join("storage", "avatars", request.SourcePeerID+ext), nil
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
