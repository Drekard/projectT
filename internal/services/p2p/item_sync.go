// Package p2p содержит сервисы для P2P связи на базе libp2p.
package p2p

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/storage/filesystem"
)

// ItemSyncProtocolID идентификатор протокола синхронизации элементов
const ItemSyncProtocolID = "/projectt/itemsync/1.0.0"

// ItemRequest запрос элементов
type ItemRequest struct {
	ItemIDs []int  `json:"item_ids,omitempty"` // Запрос конкретных элементов
	All     bool   `json:"all,omitempty"`      // Запрос всех элементов
	Hash    string `json:"hash,omitempty"`     // Запрос элемента по хешу
}

// ItemResponse ответ с элементом
type ItemResponse struct {
	ItemID       int             `json:"item_id"`
	OriginalID   int             `json:"original_id"` // ID у владельца
	OriginalHash string          `json:"original_hash"`
	Type         models.ItemType `json:"type"`
	Title        string          `json:"title"`
	Description  string          `json:"description,omitempty"`
	ContentMeta  string          `json:"content_meta,omitempty"`
	Signature    []byte          `json:"signature,omitempty"`
	Timestamp    int64           `json:"timestamp"`
	FileData     *ItemFileData   `json:"file_data,omitempty"`
}

// ItemFileData данные о файле элемента
type ItemFileData struct {
	Hash     string `json:"hash"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
	Content  []byte `json:"content,omitempty"` // Содержимое файла (опционально)
}

// ItemSyncService сервис для синхронизации элементов между пирами
type ItemSyncService struct {
	host         host.Host
	localPrivKey crypto.PrivKey
	localPubKey  crypto.PubKey
}

// NewItemSyncService создаёт сервис синхронизации элементов
func NewItemSyncService(host host.Host, privKey crypto.PrivKey, pubKey crypto.PubKey) *ItemSyncService {
	return &ItemSyncService{
		host:         host,
		localPrivKey: privKey,
		localPubKey:  pubKey,
	}
}

// Start запускает сервис синхронизации элементов
func (iss *ItemSyncService) Start() error {
	iss.host.SetStreamHandler(ItemSyncProtocolID, iss.handleItemRequest)
	log.Println("ItemSyncService запущен")
	return nil
}

// Stop останавливает сервис
func (iss *ItemSyncService) Stop() error {
	log.Println("ItemSyncService остановлен")
	return nil
}

// handleItemRequest обрабатывает входящий запрос элементов
func (iss *ItemSyncService) handleItemRequest(stream network.Stream) {
	defer stream.Close()

	remotePeer := stream.Conn().RemotePeer()
	log.Printf("Получен запрос элементов от: %s", remotePeer.String())

	// Читаем запрос
	reader := bufio.NewReader(stream)
	reqData, err := io.ReadAll(reader)
	if err != nil {
		log.Printf("Ошибка чтения запроса элементов: %v", err)
		return
	}

	var req ItemRequest
	if err := json.Unmarshal(reqData, &req); err != nil {
		log.Printf("Ошибка десериализации запроса: %v", err)
		return
	}

	// Обрабатываем запрос
	var responses []*ItemResponse

	if req.Hash != "" {
		// Запрос по хешу
		resp, err := iss.getItemByHash(req.Hash)
		if err != nil {
			log.Printf("Элемент с хэшем %s не найден: %v", req.Hash, err)
		} else if resp != nil {
			responses = append(responses, resp)
		}
	} else if len(req.ItemIDs) > 0 {
		// Запрос конкретных элементов
		for _, itemID := range req.ItemIDs {
			resp, err := iss.getItemByID(itemID)
			if err != nil {
				log.Printf("Элемент %d не найден: %v", itemID, err)
				continue
			}
			if resp != nil {
				responses = append(responses, resp)
			}
		}
	} else if req.All {
		// Запрос всех элементов
		items, err := queries.GetAllItems()
		if err != nil {
			log.Printf("Ошибка получения всех элементов: %v", err)
			return
		}

		for _, item := range items {
			resp, err := iss.itemToResponse(item)
			if err != nil {
				log.Printf("Ошибка конвертации элемента %d: %v", item.ID, err)
				continue
			}
			if resp != nil {
				responses = append(responses, resp)
			}
		}
	}

	// Отправляем ответы
	writer := bufio.NewWriter(stream)
	encoder := json.NewEncoder(writer)

	for _, resp := range responses {
		if err := encoder.Encode(resp); err != nil {
			log.Printf("Ошибка отправки элемента: %v", err)
			break
		}
	}

	if err := writer.Flush(); err != nil {
		log.Printf("Ошибка flush: %v", err)
	}

	log.Printf("Отправлено %d элементов пиру %s", len(responses), remotePeer)
}

// getItemByID возвращает элемент по ID для отправки
func (iss *ItemSyncService) getItemByID(itemID int) (*ItemResponse, error) {
	item, err := queries.GetItemByID(itemID)
	if err != nil {
		return nil, err
	}

	return iss.itemToResponse(item)
}

// getItemByHash возвращает элемент по хешу для отправки
func (iss *ItemSyncService) getItemByHash(hash string) (*ItemResponse, error) {
	item, err := queries.GetItemByHash(hash)
	if err != nil {
		return nil, err
	}

	return iss.itemToResponse(item)
}

// itemToResponse преобразует элемент в ответ
func (iss *ItemSyncService) itemToResponse(item *models.Item) (*ItemResponse, error) {
	if item == nil {
		return nil, nil
	}

	// Подписываем элемент
	signature, err := iss.signItem(item)
	if err != nil {
		log.Printf("Предупреждение: не удалось подписать элемент: %v", err)
	}

	resp := &ItemResponse{
		ItemID:       item.ID,
		OriginalID:   item.ID,
		OriginalHash: item.ContentHash,
		Type:         item.Type,
		Title:        item.Title,
		Description:  item.Description,
		ContentMeta:  item.ContentMeta,
		Signature:    signature,
		Timestamp:    time.Now().UnixNano(),
	}

	// Получаем файл если есть
	file, err := queries.GetItemFile(item.ID)
	if err == nil && file != nil {
		// Читаем содержимое файла
		content, err := filesystem.ReadFile(file.Hash)
		if err == nil {
			resp.FileData = &ItemFileData{
				Hash:     file.Hash,
				Size:     file.Size,
				MimeType: file.MimeType,
				Content:  content,
			}
		}
	}

	return resp, nil
}

// RequestItems запрашивает элементы у пира
func (iss *ItemSyncService) RequestItems(ctx context.Context, peerID peer.ID, itemIDs []int) ([]*models.RemoteItem, error) {
	stream, err := iss.host.NewStream(ctx, peerID, ItemSyncProtocolID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer stream.Close()

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

	// Устанавливаем таймаут
	if err := stream.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		log.Printf("Предупреждение: не удалось установить таймаут: %v", err)
	}

	// Читаем ответы
	reader := bufio.NewReader(stream)
	decoder := json.NewDecoder(reader)

	var remoteItems []*models.RemoteItem
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
func (iss *ItemSyncService) RequestItemByHash(ctx context.Context, peerID peer.ID, hash string) (*models.RemoteItem, error) {
	stream, err := iss.host.NewStream(ctx, peerID, ItemSyncProtocolID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer stream.Close()

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

// RequestAllItems запрашивает все элементы у пира
func (iss *ItemSyncService) RequestAllItems(ctx context.Context, peerID peer.ID) ([]*models.RemoteItem, error) {
	stream, err := iss.host.NewStream(ctx, peerID, ItemSyncProtocolID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer stream.Close()

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

	// Устанавливаем таймаут
	if err := stream.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		log.Printf("Предупреждение: не удалось установить таймаут: %v", err)
	}

	// Читаем ответы
	reader := bufio.NewReader(stream)
	decoder := json.NewDecoder(reader)

	var remoteItems []*models.RemoteItem
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

// saveRemoteItem сохраняет полученный элемент в базу данных
func (iss *ItemSyncService) saveRemoteItem(sourcePeerID string, resp *ItemResponse) (*models.RemoteItem, error) {
	// Создаём remote item
	remoteItem := &models.RemoteItem{
		SourcePeerID: sourcePeerID,
		OriginalID:   resp.OriginalID,
		OriginalHash: resp.OriginalHash,
		Title:        resp.Title,
		Description:  resp.Description,
		ContentMeta:  resp.ContentMeta,
		Signature:    resp.Signature,
		Version:      1,
	}

	// Проверяем, существует ли уже элемент с таким хешем
	exists, err := queries.RemoteItemExists(sourcePeerID, resp.OriginalHash)
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки существования: %w", err)
	}

	if exists {
		// Обновляем существующий
		existing, err := queries.GetRemoteItemByHash(sourcePeerID, resp.OriginalHash)
		if err == nil && existing != nil {
			remoteItem.ID = existing.ID
			if err := queries.UpdateRemoteItem(remoteItem); err != nil {
				return nil, fmt.Errorf("ошибка обновления элемента: %w", err)
			}
		}
	} else {
		// Создаём новый
		if err := queries.CreateRemoteItem(remoteItem); err != nil {
			return nil, fmt.Errorf("ошибка создания элемента: %w", err)
		}
	}

	// Сохраняем файл если есть
	if resp.FileData != nil && len(resp.FileData.Content) > 0 {
		fileData, err := filesystem.SaveItemFile(remoteItem.ID, resp.FileData.Content, true, sourcePeerID)
		if err != nil {
			log.Printf("Предупреждение: не удалось сохранить файл: %v", err)
		} else {
			// Сохраняем информацию о файле в БД
			itemFile := &models.ItemFile{
				ItemID:       remoteItem.ID,
				Hash:         fileData.Hash,
				FilePath:     fileData.Path,
				Size:         fileData.Size,
				MimeType:     fileData.MimeType,
				IsRemote:     true,
				SourcePeerID: sourcePeerID,
			}
			if err := queries.CreateItemFile(itemFile); err != nil {
				log.Printf("Предупреждение: не удалось сохранить метаданные файла: %v", err)
			}
		}
	}

	log.Printf("Сохранён элемент %d от пира %s (hash: %s)", remoteItem.ID, sourcePeerID, resp.OriginalHash[:16])
	return remoteItem, nil
}

// signItem подписывает элемент
func (iss *ItemSyncService) signItem(item *models.Item) ([]byte, error) {
	if iss.localPrivKey == nil {
		return nil, fmt.Errorf("приватный ключ не установлен")
	}

	// Данные для подписи
	data := fmt.Sprintf("%s|%s|%s|%s",
		item.Type,
		item.Title,
		item.Description,
		item.ContentHash,
	)

	signature, err := iss.localPrivKey.Sign([]byte(data))
	if err != nil {
		return nil, fmt.Errorf("ошибка подписи: %w", err)
	}

	return signature, nil
}

// VerifyItemSignature проверяет подпись элемента
func (iss *ItemSyncService) VerifyItemSignature(item *models.RemoteItem, publicKey, signature []byte) (bool, error) {
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
		item.OriginalHash,
	)

	valid, err := pubKey.Verify([]byte(data), signature)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки подписи: %w", err)
	}

	return valid, nil
}

// GetRemoteItemsByPeer возвращает все элементы от указанного пира
func (iss *ItemSyncService) GetRemoteItemsByPeer(peerID string) ([]*models.RemoteItem, error) {
	return queries.GetRemoteItemsByPeer(peerID)
}

// DeleteRemoteItems удаляет все элементы от пира (при удалении контакта)
func (iss *ItemSyncService) DeleteRemoteItems(peerID string) error {
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
