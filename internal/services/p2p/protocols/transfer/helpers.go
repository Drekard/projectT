package transfer

import (
	"encoding/json"
	"net/http"
	"time"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/filesystem"
)

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
