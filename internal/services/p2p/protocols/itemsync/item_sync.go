// Package itemsync содержит сервисы для синхронизации элементов между пирами.
package itemsync

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/storage/filesystem"
)

// ProtocolID идентификатор протокола синхронизации элементов
const ProtocolID = "/projectt/itemsync/1.0.0"

// min возвращает минимальное из двух чисел
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ItemRequest запрос элементов
type ItemRequest struct {
	ItemIDs []int  `json:"item_ids,omitempty"` // Запрос конкретных элементов
	All     bool   `json:"all,omitempty"`      // Запрос всех элементов
	Hash    string `json:"hash,omitempty"`     // Запрос элемента по хешу
}

// ItemResponse ответ с элементом
type ItemResponse struct {
	ElementUUID string          `json:"element_uuid"` // Уникальный ID элемента для P2P
	ItemID      int             `json:"item_id"`      // Локальный ID (для совместимости)
	Hash        string          `json:"hash"`         // Хеш содержимого
	Type        models.ItemType `json:"type"`
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	ContentMeta string          `json:"content_meta,omitempty"`
	Signature   []byte          `json:"signature,omitempty"`
	Timestamp   int64           `json:"timestamp"`
	FileData    *ItemFileData   `json:"file_data,omitempty"`
}

// ItemFileData данные о файле элемента
type ItemFileData struct {
	Hash     string `json:"hash"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
	Content  []byte `json:"content,omitempty"` // Содержимое файла (опционально)
}

// Service сервис для синхронизации элементов между пирами
type Service struct {
	host         host.Host
	localPrivKey crypto.PrivKey
	localPubKey  crypto.PubKey
}

// NewService создаёт сервис синхронизации элементов
func NewService(host host.Host, privKey crypto.PrivKey, pubKey crypto.PubKey) *Service {
	return &Service{
		host:         host,
		localPrivKey: privKey,
		localPubKey:  pubKey,
	}
}

// Start запускает сервис синхронизации элементов
func (iss *Service) Start() error {
	iss.host.SetStreamHandler(ProtocolID, iss.handleItemRequest)
	return nil
}

// Stop останавливает сервис
func (iss *Service) Stop() error {
	return nil
}

// handleItemRequest обрабатывает входящий запрос элементов
func (iss *Service) handleItemRequest(stream network.Stream) {
	defer func() { _ = stream.Close() }()

	remotePeer := stream.Conn().RemotePeer()
	log.Printf("[ItemSync] 🔔 ===============================================")
	log.Printf("[ItemSync] 🔔 ПОЛУЧЕН ЗАПРОС ЭЛЕМЕНТОВ ОТ: %s", remotePeer.String())
	log.Printf("[ItemSync] 🔔 ===============================================")

	// Читаем запрос
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

	// Обрабатываем запрос
	var responses []*ItemResponse

	if req.Hash != "" {
		// Запрос по хешу или element_uuid
		log.Printf("[ItemSync] 🔍 Запрос по UUID/Hash: %s", req.Hash)
		log.Printf("[ItemSync] 🔍 Поиск элемента в БД по element_uuid=%s...", req.Hash)

		// Сначала пробуем найти по element_uuid
		resp, err := iss.getItemByElementUUID(req.Hash)
		if err != nil {
			log.Printf("[ItemSync] ⚠️ Элемент с UUID %s не найден в БД: %v", req.Hash, err)
			log.Printf("[ItemSync] ⚠️ ВОЗМОЖНАЯ ПРИЧИНА: Элемент не существует в локальной БД")
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
			// Если не найдено по UUID, пробуем по хешу
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
		// Запрос конкретных элементов
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
		// Запрос всех элементов
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

	// Отправляем ответы
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

	// Закрываем Write чтобы получатель понял, что данные отправлены полностью
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

	// Подписываем элемент
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
		Signature:   signature,
		Timestamp:   time.Now().UnixNano(),
	}

	// Извлекаем file_hash из ContentMeta и читаем файл из общего хранилища
	log.Printf("[ItemSync] 🔍 Извлечение file_hash из ContentMeta для элемента ID=%d...", item.ID)
	log.Printf("[ItemSync] 📋 ContentMeta: %s", item.ContentMeta)

	fileHash := extractFileHashFromContentMeta(item.ContentMeta)
	if fileHash != "" {
		log.Printf("[ItemSync] ✅ file_hash извлечён из ContentMeta: %s", fileHash)
		content, err := filesystem.ReadFile(fileHash)
		if err == nil {
			log.Printf("[ItemSync] ✅ Файл прочитан по хешу из общего хранилища: %d байт", len(content))
			// Определяем MIME-тип
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

	// Парсим JSON
	var blocks []map[string]interface{}
	if err := json.Unmarshal([]byte(contentMeta), &blocks); err != nil {
		log.Printf("[ItemSync] ⚠️ Ошибка парсинга ContentMeta: %v", err)
		return ""
	}

	// Ищем block с file_hash
	for _, block := range blocks {
		if fileHash, ok := block["file_hash"].(string); ok {
			return fileHash
		}
	}

	return ""
}

// RequestItems запрашивает элементы у пира
func (iss *Service) RequestItems(ctx context.Context, peerID peer.ID, itemIDs []int) ([]*models.Item, error) {
	stream, err := iss.host.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer func() { _ = stream.Close() }()

	// Отправляем запрос
	req := &ItemRequest{ItemIDs: itemIDs}
	reqData, _ := json.Marshal(req)

	writer := bufio.NewWriter(stream)
	if _, err := writer.Write(reqData); err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("ошибка flush: %w", err)
	}

	// Закрываем Write чтобы получатель понял, что данные отправлены полностью
	if err := stream.CloseWrite(); err != nil {
		return nil, fmt.Errorf("ошибка CloseWrite: %w", err)
	}

	// Устанавливаем таймаут
	if err := stream.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		log.Printf("Предупреждение: не удалось установить таймаут: %v", err)
	}

	// Читаем ответы
	reader := bufio.NewReader(stream)
	decoder := json.NewDecoder(reader)

	var remoteItems []*models.Item
	for {
		var resp ItemResponse
		if err := decoder.Decode(&resp); err != nil {
			if err.Error() == "EOF" {
				break
			}
			log.Printf("Ошибка чтения ответа: %v", err)
			break
		}

		// Сохраняем элемент
		remoteItem, err := iss.saveRemoteItem(peerID.String(), &resp)
		if err != nil {
			log.Printf("Ошибка сохранения элемента: %v", err)
			continue
		}

		remoteItems = append(remoteItems, remoteItem)
	}

	return remoteItems, nil
}

// RequestItemByHash запрашивает элемент по хешу
func (iss *Service) RequestItemByHash(ctx context.Context, peerID peer.ID, hash string) (*models.Item, error) {
	stream, err := iss.host.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer func() { _ = stream.Close() }()

	// Отправляем запрос
	req := &ItemRequest{Hash: hash}
	reqData, _ := json.Marshal(req)

	writer := bufio.NewWriter(stream)
	if _, err := writer.Write(reqData); err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("ошибка flush: %w", err)
	}

	// Устанавливаем таймаут
	if err := stream.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		log.Printf("Предупреждение: не удалось установить таймаут: %v", err)
	}

	// Читаем ответ
	reader := bufio.NewReader(stream)
	var resp ItemResponse

	if err := json.NewDecoder(reader).Decode(&resp); err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	// Сохраняем элемент
	return iss.saveRemoteItem(peerID.String(), &resp)
}

// RequestItemByElementUUID запрашивает элемент по element_uuid
func (iss *Service) RequestItemByElementUUID(ctx context.Context, peerID peer.ID, elementUUID string) (*models.Item, error) {
	log.Printf("[ItemSync] 🔌 Запрос элемента: UUID=%s, PeerID=%s", elementUUID[:8], peerID.String()[:8])

	stream, err := iss.host.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		log.Printf("[ItemSync] ❌ Ошибка создания стрима: %v", err)
		return nil, fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer func() { _ = stream.Close() }()
	log.Printf("[ItemSync] ✅ Стрим создан")

	// Отправляем запрос с element_uuid в поле Hash (для совместимости)
	// ItemSync поддерживает запрос по уникальному идентификатору
	req := &ItemRequest{Hash: elementUUID}
	reqData, _ := json.Marshal(req)
	log.Printf("[ItemSync] 📤 Отправка запроса: %d байт", len(reqData))

	writer := bufio.NewWriter(stream)
	if _, err := writer.Write(reqData); err != nil {
		log.Printf("[ItemSync] ❌ Ошибка отправки запроса: %v", err)
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}

	if err := writer.Flush(); err != nil {
		log.Printf("[ItemSync] ❌ Ошибка flush: %v", err)
		return nil, fmt.Errorf("ошибка flush: %w", err)
	}
	log.Printf("[ItemSync] ✅ Запрос отправлен, закрываем Write...")

	// Закрываем Write чтобы получатель понял, что данные отправлены полностью
	if err := stream.CloseWrite(); err != nil {
		log.Printf("[ItemSync] ⚠️ Ошибка CloseWrite: %v", err)
		return nil, fmt.Errorf("ошибка CloseWrite: %w", err)
	}
	log.Printf("[ItemSync] ✅ Write закрыт, ждём ответ...")

	// Устанавливаем таймаут
	if err := stream.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		log.Printf("[ItemSync] ⚠️ Предупреждение: не удалось установить таймаут: %v", err)
	}

	// Читаем ответ
	reader := bufio.NewReader(stream)
	var resp ItemResponse

	log.Printf("[ItemSync] 📥 Чтение ответа...")
	if err := json.NewDecoder(reader).Decode(&resp); err != nil {
		log.Printf("[ItemSync] ❌ Ошибка чтения ответа: %v", err)
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}
	log.Printf("[ItemSync] ✅ Ответ получен: ItemID=%d, UUID=%s, Title=%q", resp.ItemID, resp.ElementUUID[:8], resp.Title)

	// Сохраняем элемент
	log.Printf("[ItemSync] 💾 Сохранение элемента в БД...")
	return iss.saveRemoteItem(peerID.String(), &resp)
}

// RequestAllItems запрашивает все элементы у пира
func (iss *Service) RequestAllItems(ctx context.Context, peerID peer.ID) ([]*models.Item, error) {
	stream, err := iss.host.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer func() { _ = stream.Close() }()

	// Отправляем запрос
	req := &ItemRequest{All: true}
	reqData, _ := json.Marshal(req)

	writer := bufio.NewWriter(stream)
	if _, err := writer.Write(reqData); err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("ошибка flush: %w", err)
	}

	// Закрываем Write чтобы получатель понял, что данные отправлены полностью
	if err := stream.CloseWrite(); err != nil {
		return nil, fmt.Errorf("ошибка CloseWrite: %w", err)
	}

	// Устанавливаем таймаут
	if err := stream.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		log.Printf("Предупреждение: не удалось установить таймаут: %v", err)
	}

	// Читаем ответы
	reader := bufio.NewReader(stream)
	decoder := json.NewDecoder(reader)

	var remoteItems []*models.Item
	for {
		var resp ItemResponse
		if err := decoder.Decode(&resp); err != nil {
			if err.Error() == "EOF" {
				break
			}
			log.Printf("Ошибка чтения ответа: %v", err)
			break
		}

		remoteItem, err := iss.saveRemoteItem(peerID.String(), &resp)
		if err != nil {
			log.Printf("Ошибка сохранения элемента: %v", err)
			continue
		}

		remoteItems = append(remoteItems, remoteItem)
	}

	return remoteItems, nil
}

// saveRemoteItem сохраняет полученный элемент от другого пира в базу данных
func (iss *Service) saveRemoteItem(sourcePeerID string, resp *ItemResponse) (*models.Item, error) {
	log.Printf("[ItemSync] 💾 saveRemoteItem: UUID=%s, Title=%q, Type=%s, HasFile=%v",
		resp.ElementUUID[:8], resp.Title, resp.Type, resp.FileData != nil)

	// Создаём item с owner_type = 'remote'
	item := &models.Item{
		ElementUUID:  resp.ElementUUID,
		OwnerType:    models.OwnerTypeRemote,
		SourcePeerID: &sourcePeerID,
		Hash:         resp.Hash,
		Type:         resp.Type,
		Title:        resp.Title,
		Description:  resp.Description,
		ContentMeta:  resp.ContentMeta,
		Signature:    resp.Signature,
		Version:      1,
	}

	// Проверяем, существует ли уже элемент с таким element_uuid
	log.Printf("[ItemSync] 🔍 Проверка существования элемента...")
	exists, err := queries.HasRemoteItem(sourcePeerID, resp.ElementUUID)
	if err != nil {
		log.Printf("[ItemSync] ❌ Ошибка проверки существования: %v", err)
		return nil, fmt.Errorf("ошибка проверки существования: %w", err)
	}
	log.Printf("[ItemSync] 📋 Элемент существует: %v", exists)

	if exists {
		// Обновляем существующий
		log.Printf("[ItemSync] 🔄 Обновление существующего элемента...")
		existing, err := queries.GetRemoteItemByElementUUID(sourcePeerID, resp.ElementUUID)
		if err == nil && existing != nil {
			item.ID = existing.ID
			item.CachedAt = existing.CachedAt
			item.CreatedAt = existing.CreatedAt
			item.UpdatedAt = existing.UpdatedAt
			if err := queries.UpdateRemoteItem(item); err != nil {
				log.Printf("[ItemSync] ❌ Ошибка обновления элемента: %v", err)
				return nil, fmt.Errorf("ошибка обновления элемента: %w", err)
			}
			log.Printf("[ItemSync] ✅ Элемент обновлён: ID=%d", item.ID)
		}
	} else {
		// Создаём новый
		log.Printf("[ItemSync] ➕ Создание нового элемента...")
		log.Printf("[ItemSync] 📋 Данные: ElementUUID=%s, OwnerType=%s, Type=%s, Title=%q, Hash=%s, Signature=%d bytes",
			item.ElementUUID, item.OwnerType, item.Type, item.Title, item.Hash, len(item.Signature))
		if err := queries.CreateRemoteItem(item); err != nil {
			log.Printf("[ItemSync] ❌ Ошибка создания элемента: %v", err)
			log.Printf("[ItemSync] 📋 SQL Error Type: %T", err)
			return nil, fmt.Errorf("ошибка создания элемента: %w", err)
		}
		log.Printf("[ItemSync] ✅ Элемент создан: ID=%d", item.ID)
	}

	// Сохраняем файл если есть
	if resp.FileData != nil && len(resp.FileData.Content) > 0 {
		log.Printf("[ItemSync] 📎 Сохранение файла: %d байт, MIME=%s", len(resp.FileData.Content), resp.FileData.MimeType)
		// Сохраняем файл в общее хранилище по хешу (не в storage/remote/...)
		fileData, err := filesystem.SaveFileWithOriginalName(resp.FileData.Content, "")
		if err != nil {
			log.Printf("[ItemSync] ⚠️ Предупреждение: не удалось сохранить файл: %v", err)
		} else {
			log.Printf("[ItemSync] ✅ Файл сохранён в общее хранилище: %s", fileData.Path)
		}
	}

	log.Printf("[ItemSync] ✅ Сохранён элемент %d от пира %s (hash: %s)", item.ID, sourcePeerID, resp.Hash[:16])
	return item, nil
}

// signItem подписывает элемент
func (iss *Service) signItem(item *models.Item) ([]byte, error) {
	if iss.localPrivKey == nil {
		return nil, fmt.Errorf("приватный ключ не установлен")
	}

	// Данные для подписи
	data := fmt.Sprintf("%s|%s|%s|%s",
		item.Type,
		item.Title,
		item.Description,
		item.Hash,
	)

	signature, err := iss.localPrivKey.Sign([]byte(data))
	if err != nil {
		return nil, fmt.Errorf("ошибка подписи: %w", err)
	}

	return signature, nil
}

// VerifyItemSignature проверяет подпись элемента
func (iss *Service) VerifyItemSignature(item *models.Item, publicKey, signature []byte) (bool, error) {
	if len(signature) == 0 {
		return false, fmt.Errorf("подпись отсутствует")
	}

	if len(publicKey) == 0 {
		return false, fmt.Errorf("публичный ключ отсутствует")
	}

	// Восстанавливаем публичный ключ
	pubKey, err := crypto.UnmarshalPublicKey(publicKey)
	if err != nil {
		return false, fmt.Errorf("ошибка восстановления публичного ключа: %w", err)
	}

	// Данные для проверки
	data := fmt.Sprintf("%s|%s|%s|%s",
		"element", // type
		item.Title,
		item.Description,
		item.Hash,
	)

	valid, err := pubKey.Verify([]byte(data), signature)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки подписи: %w", err)
	}

	return valid, nil
}

// GetRemoteItemsByPeer возвращает все элементы от указанного пира
func (iss *Service) GetRemoteItemsByPeer(peerID string) ([]*models.Item, error) {
	return queries.GetRemoteItemsByPeer(peerID)
}

// DeleteRemoteItems удаляет все элементы от пира (при удалении контакта)
func (iss *Service) DeleteRemoteItems(peerID string) error {
	// Удаляем элементы из БД
	if err := queries.DeleteRemoteItemsByPeer(peerID); err != nil {
		return fmt.Errorf("ошибка удаления элементов: %w", err)
	}

	// Удаляем файлы
	if err := filesystem.DeleteRemoteItemFiles(peerID); err != nil {
		log.Printf("Предупреждение: не удалось удалить файлы: %v", err)
	}

	return nil
}
