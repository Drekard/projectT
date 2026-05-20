package itemsync

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/storage/filesystem"
)

// RequestItems запрашивает элементы у пира
func (iss *Service) RequestItems(ctx context.Context, peerID peer.ID, itemIDs []int) ([]*models.Item, error) {
	stream, err := iss.host.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer func() { _ = stream.Close() }()

	req := &ItemRequest{ItemIDs: itemIDs}
	reqData, _ := json.Marshal(req)

	writer := bufio.NewWriter(stream)
	if _, err := writer.Write(reqData); err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("ошибка flush: %w", err)
	}

	if err := stream.CloseWrite(); err != nil {
		return nil, fmt.Errorf("ошибка CloseWrite: %w", err)
	}

	if err := stream.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		log.Printf("Предупреждение: не удалось установить таймаут: %v", err)
	}

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

		// Пропускаем пустые ответы (элемент не найден)
		if resp.ItemID == 0 {
			continue
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

// RequestItemByHash запрашивает элемент по хешу
func (iss *Service) RequestItemByHash(ctx context.Context, peerID peer.ID, hash string) (*models.Item, error) {
	stream, err := iss.host.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer func() { _ = stream.Close() }()

	req := &ItemRequest{Hash: hash}
	reqData, _ := json.Marshal(req)

	writer := bufio.NewWriter(stream)
	if _, err := writer.Write(reqData); err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("ошибка flush: %w", err)
	}

	if err := stream.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		log.Printf("Предупреждение: не удалось установить таймаут: %v", err)
	}

	reader := bufio.NewReader(stream)
	var resp ItemResponse

	if err := json.NewDecoder(reader).Decode(&resp); err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	if resp.ItemID == 0 {
		return nil, fmt.Errorf("элемент с хешем %s не найден у пира", hash[:8])
	}

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

	if err := stream.CloseWrite(); err != nil {
		log.Printf("[ItemSync] ⚠️ Ошибка CloseWrite: %v", err)
		return nil, fmt.Errorf("ошибка CloseWrite: %w", err)
	}
	log.Printf("[ItemSync] ✅ Write закрыт, ждём ответ...")

	if err := stream.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		log.Printf("[ItemSync] ⚠️ Предупреждение: не удалось установить таймаут: %v", err)
	}

	reader := bufio.NewReader(stream)
	var resp ItemResponse

	log.Printf("[ItemSync] 📥 Чтение ответа...")
	if err := json.NewDecoder(reader).Decode(&resp); err != nil {
		log.Printf("[ItemSync] ❌ Ошибка чтения ответа: %v", err)
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	// Проверяем, что элемент найден (ItemID > 0)
	if resp.ItemID == 0 {
		log.Printf("[ItemSync] ❌ Элемент %s не найден у пира", elementUUID[:8])
		return nil, fmt.Errorf("элемент %s не найден у пира", elementUUID)
	}

	log.Printf("[ItemSync] ✅ Ответ получен: ItemID=%d, UUID=%s, Title=%q", resp.ItemID, resp.ElementUUID[:8], resp.Title)

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

	req := &ItemRequest{All: true}
	reqData, _ := json.Marshal(req)

	writer := bufio.NewWriter(stream)
	if _, err := writer.Write(reqData); err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("ошибка flush: %w", err)
	}

	if err := stream.CloseWrite(); err != nil {
		return nil, fmt.Errorf("ошибка CloseWrite: %w", err)
	}

	if err := stream.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		log.Printf("Предупреждение: не удалось установить таймаут: %v", err)
	}

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

		// Пропускаем пустые ответы (элемент не найден)
		if resp.ItemID == 0 {
			continue
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

// RequestBatchByUUIDs запрашивает несколько элементов по UUID
func (iss *Service) RequestBatchByUUIDs(ctx context.Context, peerID peer.ID, elementUUIDs []string) ([]*models.Item, error) {
	if len(elementUUIDs) == 0 {
		return []*models.Item{}, nil
	}

	stream, err := iss.host.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer func() { _ = stream.Close() }()

	req := &ItemRequest{Hash: elementUUIDs[0]}
	reqData, _ := json.Marshal(req)

	writer := bufio.NewWriter(stream)
	if _, err := writer.Write(reqData); err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("ошибка flush: %w", err)
	}
	if err := stream.CloseWrite(); err != nil {
		return nil, fmt.Errorf("ошибка CloseWrite: %w", err)
	}

	timeout := 30 * time.Second
	if len(elementUUIDs) > 10 {
		timeout = time.Duration(len(elementUUIDs)*3) * time.Second
	}
	if err := stream.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		log.Printf("Предупреждение: не удалось установить таймаут: %v", err)
	}

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

		// Пропускаем пустые ответы (элемент не найден)
		if resp.ItemID == 0 {
			continue
		}

		remoteItem, err := iss.saveRemoteItem(peerID.String(), &resp)
		if err != nil {
			log.Printf("Ошибка сохранения элемента %s: %v", resp.ElementUUID[:8], err)
			continue
		}

		remoteItems = append(remoteItems, remoteItem)
	}

	return remoteItems, nil
}

// BatchRequestCallbacks коллбэки для асинхронного батч-запроса
type BatchRequestCallbacks struct {
	OnItem     func(item *models.Item, index int, total int)      // вызывается для каждого полученного элемента
	OnDone     func(items []*models.Item, err error)              // вызывается когда все элементы получены
	OnProgress func(completed int, total int, currentItem string) // вызывается для обновления прогресса
}

// RequestBatchByUUIDsAsync запрашивает батч элементов асинхронно с коллбэками
func (iss *Service) RequestBatchByUUIDsAsync(ctx context.Context, peerID peer.ID, elementUUIDs []string, callbacks BatchRequestCallbacks) {
	if len(elementUUIDs) == 0 {
		if callbacks.OnDone != nil {
			callbacks.OnDone([]*models.Item{}, nil)
		}
		return
	}

	go func() {
		stream, err := iss.host.NewStream(ctx, peerID, ProtocolID)
		if err != nil {
			if callbacks.OnDone != nil {
				callbacks.OnDone(nil, fmt.Errorf("ошибка создания стрима: %w", err))
			}
			return
		}
		defer func() { _ = stream.Close() }()

		req := &ItemRequest{Hash: elementUUIDs[0]}
		reqData, _ := json.Marshal(req)

		writer := bufio.NewWriter(stream)
		if _, err := writer.Write(reqData); err != nil {
			if callbacks.OnDone != nil {
				callbacks.OnDone(nil, fmt.Errorf("ошибка отправки запроса: %w", err))
			}
			return
		}
		if err := writer.Flush(); err != nil {
			if callbacks.OnDone != nil {
				callbacks.OnDone(nil, fmt.Errorf("ошибка flush: %w", err))
			}
			return
		}
		if err := stream.CloseWrite(); err != nil {
			if callbacks.OnDone != nil {
				callbacks.OnDone(nil, fmt.Errorf("ошибка CloseWrite: %w", err))
			}
			return
		}

		timeout := 30 * time.Second
		if len(elementUUIDs) > 10 {
			timeout = time.Duration(len(elementUUIDs)*3) * time.Second
		}
		if err := stream.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			log.Printf("Предупреждение: не удалось установить таймаут: %v", err)
		}

		reader := bufio.NewReader(stream)
		decoder := json.NewDecoder(reader)

		var remoteItems []*models.Item
		totalItems := len(elementUUIDs)
		completedCount := 0

		for {
			select {
			case <-ctx.Done():
				if callbacks.OnDone != nil {
					callbacks.OnDone(remoteItems, ctx.Err())
				}
				return
			default:
			}

			var resp ItemResponse
			if err := decoder.Decode(&resp); err != nil {
				if err.Error() == "EOF" {
					break
				}
				log.Printf("Ошибка чтения ответа: %v", err)
				break
			}

			// Пропускаем пустые ответы (элемент не найден)
			if resp.ItemID == 0 {
				completedCount++
				if callbacks.OnProgress != nil {
					callbacks.OnProgress(completedCount, totalItems, resp.ElementUUID[:min(8, len(resp.ElementUUID))])
				}
				continue
			}

			remoteItem, err := iss.saveRemoteItem(peerID.String(), &resp)
			if err != nil {
				log.Printf("Ошибка сохранения элемента %s: %v", resp.ElementUUID[:8], err)
				completedCount++
				if callbacks.OnProgress != nil {
					callbacks.OnProgress(completedCount, totalItems, resp.Title)
				}
				continue
			}

			remoteItems = append(remoteItems, remoteItem)
			completedCount++

			// Вызываем коллбэк для каждого элемента — UI может обновляться прогрессивно
			if callbacks.OnItem != nil {
				callbacks.OnItem(remoteItem, len(remoteItems)-1, totalItems)
			}

			if callbacks.OnProgress != nil {
				callbacks.OnProgress(completedCount, totalItems, resp.Title)
			}
		}

		if callbacks.OnDone != nil {
			callbacks.OnDone(remoteItems, nil)
		}
	}()
}

// RequestFolder запрашивает все элементы папки по parent_uuid
func (iss *Service) RequestFolder(ctx context.Context, peerID peer.ID, parentUUID string) ([]*models.Item, error) {
	if parentUUID == "" {
		log.Printf("[ItemSync] 📁 Запрос корневых элементов пира")
	} else {
		log.Printf("[ItemSync] 📁 Запрос папки: parentUUID=%s", parentUUID[:8])
	}

	items, err := iss.RequestAllItems(ctx, peerID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса всех элементов: %w", err)
	}

	var folderItems []*models.Item
	for _, item := range items {
		if parentUUID == "" {
			if item.ParentUUID == nil || *item.ParentUUID == "" {
				folderItems = append(folderItems, item)
			}
		} else {
			if item.ParentUUID != nil && *item.ParentUUID == parentUUID {
				folderItems = append(folderItems, item)
			}
		}
	}

	if parentUUID == "" {
		log.Printf("[ItemSync] 📁 Корневые элементы: найдено %d элементов", len(folderItems))
	} else {
		log.Printf("[ItemSync] 📁 Папка %s: найдено %d элементов", parentUUID[:8], len(folderItems))
	}
	return folderItems, nil
}

// saveRemoteItem сохраняет полученный элемент от другого пира в базу данных
func (iss *Service) saveRemoteItem(sourcePeerID string, resp *ItemResponse) (*models.Item, error) {
	log.Printf("[ItemSync] 💾 saveRemoteItem: UUID=%s, Title=%q, Type=%s, HasFile=%v",
		resp.ElementUUID[:8], resp.Title, resp.Type, resp.FileData != nil)

	item := &models.Item{
		ElementUUID:  resp.ElementUUID,
		OwnerType:    models.OwnerTypeRemote,
		SourcePeerID: &sourcePeerID,
		Hash:         resp.Hash,
		Type:         resp.Type,
		Title:        resp.Title,
		Description:  resp.Description,
		ContentMeta:  resp.ContentMeta,
		ParentUUID:   resp.ParentUUID,
		Signature:    resp.Signature,
		Version:      1,
	}

	log.Printf("[ItemSync] 🔍 Проверка существования элемента...")
	exists, err := queries.HasRemoteItem(sourcePeerID, resp.ElementUUID)
	if err != nil {
		log.Printf("[ItemSync] ❌ Ошибка проверки существования: %v", err)
		return nil, fmt.Errorf("ошибка проверки существования: %w", err)
	}
	log.Printf("[ItemSync] 📋 Элемент существует: %v", exists)

	if exists {
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

	if resp.FileData != nil && len(resp.FileData.Content) > 0 {
		log.Printf("[ItemSync] 📎 Сохранение файла: %d байт, MIME=%s", len(resp.FileData.Content), resp.FileData.MimeType)
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

// RequestRandomItemsAsync запрашивает случайные элементы у пира асинхронно
func (iss *Service) RequestRandomItemsAsync(ctx context.Context, peerID peer.ID, count int, callbacks BatchRequestCallbacks) {
	if count <= 0 {
		count = 10
	}
	if count > 50 {
		count = 50
	}

	go func() {
		stream, err := iss.host.NewStream(ctx, peerID, ProtocolID)
		if err != nil {
			if callbacks.OnDone != nil {
				callbacks.OnDone(nil, fmt.Errorf("ошибка создания стрима: %w", err))
			}
			return
		}
		defer func() { _ = stream.Close() }()

		req := &ItemRequest{Random: true, Count: count}
		reqData, _ := json.Marshal(req)

		writer := bufio.NewWriter(stream)
		if _, err := writer.Write(reqData); err != nil {
			if callbacks.OnDone != nil {
				callbacks.OnDone(nil, fmt.Errorf("ошибка отправки запроса: %w", err))
			}
			return
		}
		if err := writer.Flush(); err != nil {
			if callbacks.OnDone != nil {
				callbacks.OnDone(nil, fmt.Errorf("ошибка flush: %w", err))
			}
			return
		}
		if err := stream.CloseWrite(); err != nil {
			if callbacks.OnDone != nil {
				callbacks.OnDone(nil, fmt.Errorf("ошибка CloseWrite: %w", err))
			}
			return
		}

		if err := stream.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			log.Printf("Предупреждение: не удалось установить таймаут: %v", err)
		}

		reader := bufio.NewReader(stream)
		decoder := json.NewDecoder(reader)

		var remoteItems []*models.Item
		totalExpected := count
		receivedCount := 0

		for {
			select {
			case <-ctx.Done():
				if callbacks.OnDone != nil {
					callbacks.OnDone(remoteItems, ctx.Err())
				}
				return
			default:
			}

			var resp ItemResponse
			if err := decoder.Decode(&resp); err != nil {
				if err.Error() == "EOF" {
					break
				}
				log.Printf("Ошибка чтения ответа: %v", err)
				break
			}

			if resp.ItemID == 0 {
				break
			}

			remoteItem, err := iss.saveRemoteItem(peerID.String(), &resp)
			if err != nil {
				log.Printf("Ошибка сохранения элемента %s: %v", resp.ElementUUID[:8], err)
				receivedCount++
				continue
			}

			remoteItems = append(remoteItems, remoteItem)
			receivedCount++

			if callbacks.OnItem != nil {
				callbacks.OnItem(remoteItem, len(remoteItems)-1, totalExpected)
			}

			if callbacks.OnProgress != nil {
				callbacks.OnProgress(receivedCount, totalExpected, resp.Title)
			}
		}

		if callbacks.OnDone != nil {
			callbacks.OnDone(remoteItems, nil)
		}
	}()
}
