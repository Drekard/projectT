package itemsync

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/libp2p/go-libp2p/core/network"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/storage/filesystem"
)

// handleItemRequest обрабатывает входящий запрос элементов
func (iss *Service) handleItemRequest(stream network.Stream) {
	defer func() { _ = stream.Close() }()

	reader := bufio.NewReader(stream)
	reqData, err := io.ReadAll(reader)
	if err != nil {
		return
	}

	var req ItemRequest
	if err := json.Unmarshal(reqData, &req); err != nil {
		return
	}

	var responses []*ItemResponse

	if req.Hash != "" {
		resp, err := iss.getItemByElementUUID(req.Hash)
		if err != nil {
		} else if resp != nil {
			responses = append(responses, resp)
		} else {
			resp, err = iss.getItemByHash(req.Hash)
			if err != nil {
			} else if resp != nil {
				responses = append(responses, resp)
			}
		}
	} else if len(req.ItemIDs) > 0 {
		for _, itemID := range req.ItemIDs {
			resp, err := iss.getItemByID(itemID)
			if err != nil {
				continue
			}
			if resp != nil {
				responses = append(responses, resp)
			}
		}
	} else if req.All {
		items, err := queries.GetAllItems()
		if err != nil {
			return
		}

		for _, item := range items {
			resp, err := iss.itemToResponse(item)
			if err != nil {
				continue
			}
			if resp != nil {
				responses = append(responses, resp)
			}
		}
	}

	writer := bufio.NewWriter(stream)
	encoder := json.NewEncoder(writer)

	for _, resp := range responses {
		if err := encoder.Encode(resp); err != nil {
			break
		}
	}

	_ = writer.Flush()
	_ = stream.CloseWrite()
}

// getItemByID возвращает элемент по ID для отправки
func (iss *Service) getItemByID(itemID int) (*ItemResponse, error) {
	item, err := queries.GetItemByID(itemID)
	if err != nil {
		return nil, err
	}
	return iss.itemToResponse(item)
}

// getItemByHash возвращает элемент по хешу для отправки
func (iss *Service) getItemByHash(hash string) (*ItemResponse, error) {
	item, err := queries.GetItemByHash(hash)
	if err != nil {
		return nil, err
	}
	return iss.itemToResponse(item)
}

// getItemByElementUUID возвращает элемент по element_uuid для отправки
func (iss *Service) getItemByElementUUID(elementUUID string) (*ItemResponse, error) {
	item, err := queries.GetItemByElementUUID(elementUUID)
	if err != nil {
		return nil, err
	}
	return iss.itemToResponse(item)
}

// itemToResponse преобразует элемент в ответ
func (iss *Service) itemToResponse(item *models.Item) (*ItemResponse, error) {
	if item == nil {
		return nil, nil
	}

	signature, err := iss.signItem(item)
	if err != nil {
		signature = nil
	}

	resp := &ItemResponse{
		ElementUUID: item.ElementUUID,
		ItemID:      item.ID,
		Hash:        item.Hash,
		Type:        item.Type,
		Title:       item.Title,
		Description: item.Description,
		ContentMeta: item.ContentMeta,
		ParentUUID:  item.ParentUUID,
		Signature:   signature,
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
			return resp, nil
		}
	}

	return resp, nil
}

// detectMimeType определяет MIME-тип файла по содержимому
func detectMimeType(fileBytes []byte) string {
	return http.DetectContentType(fileBytes)
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
