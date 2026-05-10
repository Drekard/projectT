package transfer

import (
	"context"
	"sync"
)

// ProtocolID идентификатор протокола передачи файлов
const ProtocolID = "/projectt/transfer/1.0.0"

// BatchProtocolID идентификатор протокола пакетной передачи
const BatchProtocolID = "/projectt/batch-transfer/1.0.0"

// ChunkSize размер чанка при передаче файла (64 KB)
const ChunkSize = 64 * 1024

// MaxConcurrentBatchStreams максимальное количество параллельных стримов в батче
const MaxConcurrentBatchStreams = 3

// TransferType тип передачи
type TransferType string

const (
	TransferTypeFile            TransferType = "file"
	TransferTypeAvatar          TransferType = "avatar"
	TransferTypeImage           TransferType = "image"
	TransferTypeElementMetadata TransferType = "element_metadata"
	TransferTypeBatchFolder     TransferType = "batch_folder"
	TransferTypeBatchPinned     TransferType = "batch_pinned"
	TransferTypeBatchSelection  TransferType = "batch_selection"
)

// TransferStatus статус передачи
type TransferStatus string

const (
	TransferStatusPending    TransferStatus = "pending"
	TransferStatusInProgress TransferStatus = "in_progress"
	TransferStatusCompleted  TransferStatus = "completed"
	TransferStatusFailed     TransferStatus = "failed"
	TransferStatusCancelled  TransferStatus = "cancelled"
)

// FileTransferRequest запрос на передачу файла
type FileTransferRequest struct {
	TransferID   string       `json:"transfer_id"`
	Type         TransferType `json:"type"`
	FileName     string       `json:"file_name"`
	FileSize     int64        `json:"file_size"`
	MimeType     string       `json:"mime_type"`
	FileHash     string       `json:"file_hash"`
	SourcePeerID string       `json:"source_peer_id"`
}

// FileTransferChunk чанк файла
type FileTransferChunk struct {
	TransferID string `json:"transfer_id"`
	Offset     int64  `json:"offset"`
	Data       []byte `json:"data"`
	IsLast     bool   `json:"is_last"`
}

// TransferAck подтверждение получения чанка
type TransferAck struct {
	TransferID string `json:"transfer_id"`
	Received   int64  `json:"received"`
	Success    bool   `json:"success"`
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
	Destination string
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
