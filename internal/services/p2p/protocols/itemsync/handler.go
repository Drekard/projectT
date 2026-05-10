package itemsync

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/libp2p/go-libp2p/core/network"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/storage/filesystem"
)

// min возвращает минимальное из двух чисел
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// handleItemRequest обрабатывает входящий запрос элементов
func (iss *Service) handleItemRequest(stream network.Stream) {
	defer func() { _ = stream.Close() }()

	remotePeer := stream.Conn().RemotePeer()
	log.Printf("[ItemSync] 🔔 ===============================================")
	log.Printf("[ItemSync] 🔔 ПОЛУЧЕН ЗАПРОС ЭЛЕМЕНТОВ ОТ: %s", remotePeer.String())
	log.Printf("[ItemSync] 🔔 ===============================================")

	reader := bufio.NewReader(stream)
	reqData, err := io.ReadAll(reader)
	if err != nil {
		log.Printf("[ItemSync] ❌ Ошибка чтения запроса элементов: %v", err)
		return
	}
	log.Printf("[ItemSync] 📥 Прочитано %d байт запроса", len(reqData))
	log.Printf("[ItemSync] 📄 RAW данные запроса (hex): %x", reqData[:min(len(reqData), 100)])

	var req ItemRequest
	if err := json.Unmarshal(reqData, &req); err != nil {
		log.Printf("[ItemSync] ❌ Ошибка десериализации запроса: %v", err)
		log.Printf("[ItemSync] 📄 Raw данные: %s", string(reqData))
		return
	}

	log.Printf("[ItemSync] 📋 === ДЕТАЛИ ЗАПРОСА ===")
	log.Printf("[ItemSync] 📋 Hash/UUID: %q", req.Hash)
	log.Printf("[ItemSync] 📋 ItemIDs: %v", req.ItemIDs)
	log.Printf("[ItemSync] 📋 All: %v", req.All)
	log.Printf("[ItemSync] 📋 ОЖИДАНИЕ: Пир хочет получить элемент с UUID=%s", req.Hash)

	var responses []*ItemResponse

	if req.Hash != "" {
		log.Printf("[ItemSync] 🔍 Запрос по UUID/Hash: %s", req.Hash)
		log.Printf("[ItemSync] 🔍 Поиск элемента в БД по element_uuid=%s...", req.Hash)

		resp, err := iss.getItemByElementUUID(req.Hash)
		if err != nil {
			log.Printf("[ItemSync] ⚠️ Элемент с UUID %s не найден в БД: %v", req.Hash, err)
		} else if resp != nil {
			log.Printf("[ItemSync] ✅ Элемент НАЙДЕН в БД:")
			log.Printf("[ItemSync]    - ItemID: %d", resp.ItemID)
			log.Printf("[ItemSync]    - ElementUUID: %s", resp.ElementUUID)
			log.Printf("[ItemSync]    - Title: %q", resp.Title)
			log.Printf("[ItemSync]    - Type: %s", resp.Type)
			log.Printf("[ItemSync]    - Hash: %s", resp.Hash)
			log.Printf("[ItemSync]    - Description: %q", resp.Description)
			log.Printf("[ItemSync]    - ContentMeta: %q", resp.ContentMeta)
			log.Printf("[ItemSync]    - FileData: %v", resp.FileData != nil)
			if resp.FileData != nil {
				log.Printf("[ItemSync]    - File Size: %d байт", resp.FileData.Size)
				log.Printf("[ItemSync]    - File MIME: %s", resp.FileData.MimeType)
			}
			responses = append(responses, resp)
		} else {
			log.Printf("[ItemSync] 🔍 Элемент не найден по UUID, пробуем по хешу: %s", req.Hash)
			resp, err = iss.getItemByHash(req.Hash)
			if err != nil {
				log.Printf("[ItemSync] ⚠️ Элемент с хэшем %s не найден: %v", req.Hash, err)
			} else if resp != nil {
				log.Printf("[ItemSync] ✅ Элемент найден по хешу: ID=%d, Title=%q", resp.ItemID, resp.Title)
				responses = append(responses, resp)
			}
		}
	} else if len(req.ItemIDs) > 0 {
		log.Printf("[ItemSync] 🔍 Запрос %d элементов по ID", len(req.ItemIDs))
		for _, itemID := range req.ItemIDs {
			resp, err := iss.getItemByID(itemID)
			if err != nil {
				log.Printf("[ItemSync] ⚠️ Элемент %d не найден: %v", itemID, err)
				continue
			}
			if resp != nil {
				log.Printf("[ItemSync] ✅ Элемент %d найден: Title=%q", itemID, resp.Title)
				responses = append(responses, resp)
			}
		}
	} else if req.All {
		log.Printf("[ItemSync] 🔍 Запрос всех элементов")
		items, err := queries.GetAllItems()
		if err != nil {
			log.Printf("[ItemSync] ❌ Ошибка получения всех элементов: %v", err)
			return
		}
		log.Printf("[ItemSync] 📊 Найдено %d элементов", len(items))

		for _, item := range items {
			resp, err := iss.itemToResponse(item)
			if err != nil {
				log.Printf("[ItemSync] ⚠️ Ошибка конвертации элемента %d: %v", item.ID, err)
				continue
			}
			if resp != nil {
				responses = append(responses, resp)
			}
		}
	}

	log.Printf("[ItemSync] 📤 Отправка %d ответов...", len(responses))
	log.Printf("[ItemSync] 📤 === ОТПРАВКА ОТВЕТА ===")
	log.Printf("[ItemSync] 📤 Найдено элементов для отправки: %d", len(responses))

	writer := bufio.NewWriter(stream)
	encoder := json.NewEncoder(writer)

	for i, resp := range responses {
		log.Printf("[ItemSync] 📤 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		log.Printf("[ItemSync] 📤 ОТПРАВКА элемента #%d/%d:", i+1, len(responses))
		log.Printf("[ItemSync] 📤    - ItemID: %d", resp.ItemID)
		log.Printf("[ItemSync] 📤    - ElementUUID: %s", resp.ElementUUID)
		log.Printf("[ItemSync] 📤    - Title: %q", resp.Title)
		log.Printf("[ItemSync] 📤    - Type: %s", resp.Type)
		log.Printf("[ItemSync] 📤    - Hash: %s", resp.Hash)
		log.Printf("[ItemSync] 📤    - Description: %q", resp.Description)
		log.Printf("[ItemSync] 📤    - ContentMeta: %s", resp.ContentMeta)
		log.Printf("[ItemSync] 📤    - Signature: %d байт", len(resp.Signature))
		if resp.FileData != nil {
			log.Printf("[ItemSync] 📤    - ФАЙЛ ВКЛЮЧЁН:")
			log.Printf("[ItemSync] 📤       * Size: %d байт", resp.FileData.Size)
			log.Printf("[ItemSync] 📤       * MimeType: %s", resp.FileData.MimeType)
			log.Printf("[ItemSync] 📤       * Hash: %s", resp.FileData.Hash)
		} else {
			log.Printf("[ItemSync] 📤    - ФАЙЛ: не прикреплён")
		}

		if err := encoder.Encode(resp); err != nil {
			log.Printf("[ItemSync] ❌ Ошибка отправки элемента #%d: %v", i, err)
			break
		}
		log.Printf("[ItemSync] ✅ Элемент #%d отправлен в стрим", i+1)
	}

	if err := writer.Flush(); err != nil {
		log.Printf("[ItemSync] ❌ Ошибка flush: %v", err)
	}

	if err := stream.CloseWrite(); err != nil {
		log.Printf("[ItemSync] ⚠️ Ошибка CloseWrite: %v", err)
	}

	log.Printf("[ItemSync] ✅ ===============================================")
	log.Printf("[ItemSync] ✅ ОТПРАВЛЕНО %d элементов пиру %s", len(responses), remotePeer)
	log.Printf("[ItemSync] ✅ ===============================================")
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

	log.Printf("[ItemSync] 📋 itemToResponse: ItemID=%d, UUID=%s, Title=%q, OwnerType=%s",
		item.ID, item.ElementUUID[:8], item.Title, item.OwnerType)

	signature, err := iss.signItem(item)
	if err != nil {
		log.Printf("Предупреждение: не удалось подписать элемент: %v", err)
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

	log.Printf("[ItemSync] 🔍 Извлечение file_hash из ContentMeta для элемента ID=%d...", item.ID)
	log.Printf("[ItemSync] 📋 ContentMeta: %s", item.ContentMeta)

	fileHash := extractFileHashFromContentMeta(item.ContentMeta)
	if fileHash != "" {
		log.Printf("[ItemSync] ✅ file_hash извлечён из ContentMeta: %s", fileHash)
		content, err := filesystem.ReadFile(fileHash)
		if err == nil {
			log.Printf("[ItemSync] ✅ Файл прочитан по хешу из общего хранилища: %d байт", len(content))
			mimeType := detectMimeType(content)
			resp.FileData = &ItemFileData{
				Hash:     fileHash,
				Size:     int64(len(content)),
				MimeType: mimeType,
				Content:  content,
			}
			log.Printf("[ItemSync] ✅ Файл добавлен в ответ (MIME=%s)", mimeType)
			return resp, nil
		}
		log.Printf("[ItemSync] ❌ Ошибка чтения файла по хешу: %v", err)
	} else {
		log.Printf("[ItemSync] ℹ️ file_hash не найден в ContentMeta (элемент без файла)")
	}

	log.Printf("[ItemSync] ⚠️ Файл не будет включён в ответ (FileData=nil)")
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
		log.Printf("[ItemSync] ⚠️ Ошибка парсинга ContentMeta: %v", err)
		return ""
	}

	for _, block := range blocks {
		if fileHash, ok := block["file_hash"].(string); ok {
			return fileHash
		}
	}

	return ""
}
