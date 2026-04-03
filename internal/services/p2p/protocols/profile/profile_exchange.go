// Package profile содержит сервисы для обмена профилями между пирами.
package profile

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"projectT/internal/services/p2p/protocols/avatar"
	"projectT/internal/services/p2p/protocols/transfer"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/storage/filesystem"
)

// ProtocolID идентификатор протокола обмена профилями (версия 2.0 для новой схемы)
const ProtocolID = "/projectt/profile/2.0.0"

// ProfileRequest запрос профиля
type ProfileRequest struct {
	RequestFull bool `json:"request_full"` // Запрос полного профиля с подписью
	IsInitiator bool `json:"is_initiator"` // true = инициатор подключения (Роль 1), false = сервер (Роль 2)
}

// ProfileResponse ответ с профилем (полная версия)
type ProfileResponse struct {
	PeerID         string   `json:"peer_id"`
	Username       string   `json:"username"`
	Title          string   `json:"title"`
	AvatarPath     string   `json:"avatar_path"`           // Путь к аватарке (для совместимости)
	AvatarHash     string   `json:"avatar_hash"`           // ✅ Хеш аватарки (имя файла) для отдельной загрузки
	AvatarData     []byte   `json:"avatar_data,omitempty"` // Данные аватара в base64 (устаревает)
	BackgroundPath string   `json:"background_path"`
	ContentChar    string   `json:"content_characteristic"`
	PinnedUUIDs    []string `json:"pinned_uuids"`          // UUID избранных элементов
	PinnedData     []byte   `json:"pinned_data,omitempty"` // Сжатые данные закреплённых элементов
	PublicKey      []byte   `json:"public_key"`
	Signature      []byte   `json:"signature,omitempty"` // Подпись профиля
	Timestamp      int64    `json:"timestamp"`
	IsInitiator    bool     `json:"is_initiator"`             // Роль отправителя профиля
	EncryptionKey  []byte   `json:"encryption_key,omitempty"` // Ключ для симметричного шифрования сообщений
}

// MinimalProfileResponse минимальный профиль для быстрого обмена (без avatar/pinned_uuids)
// Используется при первичном обнаружении пира для экономии трафика
type MinimalProfileResponse struct {
	PeerID        string `json:"peer_id"`
	Username      string `json:"username"`
	PublicKey     []byte `json:"public_key"`
	Timestamp     int64  `json:"timestamp"`
	IsInitiator   bool   `json:"is_initiator"`
	EncryptionKey []byte `json:"encryption_key,omitempty"`
}

// ProfileWithSignature профиль вместе с подписью для проверки
type ProfileWithSignature struct {
	Profile   *models.Profile
	PublicKey []byte
	Signature []byte
}

// ExchangeService сервис для обмена профилями между пирами
type ExchangeService struct {
	mu              sync.RWMutex // мьютекс для потокобезопасности
	host            host.Host
	localPrivKey    crypto.PrivKey
	localPubKey     crypto.PubKey
	localEncryptKey []byte             // локальный ключ для симметричного шифрования
	peerEncryptKeys map[peer.ID][]byte // ключи шифрования пиров
	transferSvc     *transfer.Service  // сервис передачи файлов для аватарок
	avatarSvc       *avatar.Service    // ✅ сервис загрузки аватарок
	connSvc         interface {
		MarkProfilePending(peer.ID)
		MarkProfileComplete(peer.ID)
		CanRequestProfile(peer.ID) bool
	} // сервис подключений для отслеживания статуса профиля
	uiP2P interface {
		OnProfileUpdated(peerID string)
	} // UI callback для уведомления об обновлении профиля
	uiProfilePanel interface {
		RefreshDemoElementsAfterSync(peerID string)
	} // UI callback для обновления витрины элементов после синхронизации
}

// NewExchangeService создаёт сервис обмена профилями
func NewExchangeService(host host.Host, privKey crypto.PrivKey, pubKey crypto.PubKey) *ExchangeService {
	// Генерируем локальный ключ шифрования
	localEncryptKey := generateEncryptionKey()

	return &ExchangeService{
		host:            host,
		localPrivKey:    privKey,
		localPubKey:     pubKey,
		localEncryptKey: localEncryptKey,
		peerEncryptKeys: make(map[peer.ID][]byte),
	}
}

// SetTransferService устанавливает сервис передачи файлов
func (pes *ExchangeService) SetTransferService(transferSvc *transfer.Service) {
	pes.transferSvc = transferSvc
}

// SetAvatarService устанавливает сервис загрузки аватарок
func (pes *ExchangeService) SetAvatarService(avatarSvc *avatar.Service) {
	pes.avatarSvc = avatarSvc
}

// SetConnectionService устанавливает сервис подключений
func (pes *ExchangeService) SetConnectionService(connSvc interface {
	MarkProfilePending(peer.ID)
	MarkProfileComplete(peer.ID)
	CanRequestProfile(peer.ID) bool
}) {
	pes.connSvc = connSvc
}

// SetUIP2P устанавливает UI API для уведомления об обновлении профиля
func (pes *ExchangeService) SetUIP2P(uiP2P interface {
	OnProfileUpdated(peerID string)
}) {
	pes.uiP2P = uiP2P
}

// SetUIProfilePanel устанавливает UI callback для обновления витрины элементов
func (pes *ExchangeService) SetUIProfilePanel(uiPanel interface {
	RefreshDemoElementsAfterSync(peerID string)
}) {
	pes.uiProfilePanel = uiPanel
}

// getTransferService возвращает сервис передачи файлов
func (pes *ExchangeService) getTransferService() *transfer.Service {
	if pes == nil {
		return nil
	}
	return pes.transferSvc
}

// Start запускает сервис обмена профилями
func (pes *ExchangeService) Start() error {
	pes.host.SetStreamHandler(ProtocolID, pes.handleProfileRequest)
	log.Println("ProfileExchangeService v2.0 запущен")
	return nil
}

// Stop останавливает сервис
func (pes *ExchangeService) Stop() error {
	log.Println("ProfileExchangeService остановлен")
	return nil
}

// handleProfileRequest обрабатывает входящий запрос профиля (Роль 2 - СЕРВЕР)
// Протокол:
// 1. Читаем запрос (с флагом IsInitiator)
// 2. Отправляем свой профиль
// 3. Читаем ответ (профиль инициатора)
func (pes *ExchangeService) handleProfileRequest(stream network.Stream) {
	defer stream.Close()

	remotePeer := stream.Conn().RemotePeer()
	log.Printf("[Profile] === Получен запрос профиля от: %s (Роль 2 - СЕРВЕР) ===", remotePeer.String())

	// Читаем запрос с помощью json.Decoder (не читаем весь стрим)
	reader := bufio.NewReader(stream)
	decoder := json.NewDecoder(reader)

	log.Printf("[Profile] Чтение запроса от %s...", remotePeer.String()[:8])
	req := ProfileRequest{}
	if err := decoder.Decode(&req); err != nil {
		log.Printf("[Profile] Ошибка чтения запроса профиля от %s: %v", remotePeer.String()[:8], err)
		return
	}
	log.Printf("[Profile] Запрос получен от %s: RequestFull=%v, IsInitiator=%v", remotePeer.String()[:8], req.RequestFull, req.IsInitiator)

	// Отправляем свой профиль (Роль 2 - СЕРВЕР, isInitiator=false)
	log.Printf("[Profile] Отправка своего профиля (СЕРВЕР) пиру %s...", remotePeer.String()[:8])
	if err := pes.sendLocalProfile(stream, false); err != nil {
		log.Printf("[Profile] Ошибка отправки профиля: %v", err)
		return
	}
	log.Printf("[Profile] ✅ Свой профиль (СЕРВЕР) отправлен %s", remotePeer.String()[:8])

	// Читаем профиль инициатора в ответ
	log.Printf("[Profile] Чтение профиля инициатора от %s...", remotePeer.String()[:8])
	startTime := time.Now()
	// Увеличиваем таймаут до 60 секунд для соединений с большими аватарами
	if err := stream.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		log.Printf("[Profile] Предупреждение: не удалось установить таймаут: %v", err)
	}

	response := &ProfileResponse{}
	if err := json.NewDecoder(reader).Decode(response); err != nil {
		log.Printf("[Profile] Ошибка чтения профиля инициатора от %s за %v: %v", remotePeer.String()[:8], time.Since(startTime), err)
		return
	}
	avatarSize := len(response.AvatarData)
	log.Printf("[Profile] ✅ Профиль инициатора получен от %s за %v (username: %s, аватар: %d байт, encryption_key[0]=%d)", remotePeer.String()[:8], time.Since(startTime), response.Username, avatarSize, response.EncryptionKey[0])

	// Сохраняем профиль инициатора
	profile := &models.Profile{
		OwnerType:      models.OwnerTypeRemote,
		PeerID:         response.PeerID,
		Username:       response.Username,
		Title:          response.Title,
		AvatarPath:     "", // Путь будет установлен после сохранения аватара локально
		BackgroundPath: response.BackgroundPath,
		ContentChar:    response.ContentChar,
		PinnedUUIDs:    "[]",
	}

	if len(response.PinnedUUIDs) > 0 {
		uuidsJSON, err := json.Marshal(response.PinnedUUIDs)
		if err == nil {
			profile.PinnedUUIDs = string(uuidsJSON)
		}
	}

	now := time.Now()
	profile.CachedAt = &now

	if err := pes.savePeerProfile(profile, response.PublicKey, response.Signature); err != nil {
		log.Printf("[Profile] Предупреждение: не удалось сохранить профиль: %v", err)
	}

	log.Printf("[Profile] ✅ Профиль инициатора %s сохранён (username: %s)", response.PeerID[:8], response.Username)

	// ✅ Загружаем аватарку через отдельный сервис если есть AvatarHash
	if response.AvatarHash != "" {
		go pes.downloadAvatarSeparately(remotePeer, response.AvatarHash)
	} else if len(response.AvatarData) > 0 {
		// ⚠️ Устаревший способ: загружаем из данных в профиле (для совместимости)
		go pes.saveAvatarFromProfileData(remotePeer, response.AvatarData)
	}

	// ⚠️ Pinned элементы НЕ загружаются автоматически при обмене профилями
	// Загрузка происходит только при создании чата (в chat_controller.go)
	// if len(response.PinnedUUIDs) > 0 {
	// 	go pes.downloadPinnedItems(remotePeer, response.PinnedUUIDs)
	// }

	// Сохраняем ключ шифрования пира
	if len(response.EncryptionKey) > 0 {
		pes.SetPeerEncryptionKey(remotePeer, response.EncryptionKey)
		log.Printf("[Profile] 🔑 Ключ шифрования сохранён для %s (len=%d, key[0]=%d)", remotePeer.String()[:8], len(response.EncryptionKey), response.EncryptionKey[0])
	} else {
		log.Printf("[Profile] ⚠️ Ключ шифрования не получен от %s", remotePeer.String()[:8])
	}
}

// RequestPeerProfile запрашивает профиль у удалённого пира (Роль 1 - ИНИЦИАТОР)
// Протокол:
// 1. Отправляем запрос с IsInitiator=true
// 2. Читаем ответ (профиль сервера)
// 3. Отправляем свой профиль в ответ
func (pes *ExchangeService) RequestPeerProfile(ctx context.Context, peerID peer.ID) (*ProfileWithSignature, error) {
	return pes.requestPeerProfileWithRole(ctx, peerID, true)
}

// requestPeerProfileWithRole запрашивает профиль с указанной ролью
func (pes *ExchangeService) requestPeerProfileWithRole(ctx context.Context, peerID peer.ID, isInitiator bool) (*ProfileWithSignature, error) {
	// Проверяем, можно ли запрашивать профиль
	if pes.connSvc != nil && !pes.connSvc.CanRequestProfile(peerID) {
		log.Printf("[Profile] ⏭️ Пропуск запроса профиля у %s (обмен был недавно)", peerID.String()[:8])
		return nil, nil // Не ошибка, просто пропускаем
	}

	log.Printf("[Profile] Запрос профиля у пира %s (роль: %s)...", peerID.String()[:8], map[bool]string{true: "ИНИЦИАТОР", false: "СЕРВЕР"}[isInitiator])

	// Отмечаем начало обмена профиля
	if pes.connSvc != nil {
		pes.connSvc.MarkProfilePending(peerID)
	}
	defer func() {
		if pes.connSvc != nil {
			pes.connSvc.MarkProfileComplete(peerID)
		}
	}()

	// Создаём стрим
	startTime := time.Now()
	stream, err := pes.host.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		log.Printf("[Profile] Ошибка создания стрима для %s: %v", peerID.String()[:8], err)
		return nil, fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer stream.Close()
	log.Printf("[Profile] Стрим создан для %s за %v", peerID.String()[:8], time.Since(startTime))

	// Отправляем запрос
	req := &ProfileRequest{RequestFull: true, IsInitiator: isInitiator}
	reqData, _ := json.Marshal(req)

	log.Printf("[Profile] Отправка запроса профилю %s (IsInitiator=%v)...", peerID.String()[:8], isInitiator)
	writer := bufio.NewWriter(stream)
	if _, err := writer.Write(reqData); err != nil {
		log.Printf("[Profile] Ошибка отправки запроса для %s: %v", peerID.String()[:8], err)
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}

	if err := writer.Flush(); err != nil {
		log.Printf("[Profile] Ошибка flush для %s: %v", peerID.String()[:8], err)
		return nil, fmt.Errorf("ошибка flush: %w", err)
	}
	log.Printf("[Profile] Запрос отправлен %s, ждём ответ...", peerID.String()[:8])

	// Увеличиваем таймаут до 60 секунд для нестабильных соединений и больших аватаров
	if err := stream.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		log.Printf("[Profile] Предупреждение: не удалось установить таймаут: %v", err)
	}

	// Читаем ответ (профиль сервера)
	log.Printf("[Profile] Чтение ответа (профиль сервера) от %s...", peerID.String()[:8])
	startTime = time.Now()
	reader := bufio.NewReader(stream)
	response := &ProfileResponse{}

	if err := json.NewDecoder(reader).Decode(response); err != nil {
		log.Printf("[Profile] Ошибка чтения ответа от %s за %v: %v", peerID.String()[:8], time.Since(startTime), err)
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}
	avatarSize := len(response.AvatarData)
	log.Printf("[Profile] ✅ Профиль сервера получен от %s за %v (username: %s, аватар: %d байт, encryption_key[0]=%d)", peerID.String()[:8], time.Since(startTime), response.Username, avatarSize, response.EncryptionKey[0])

	// Если мы инициатор - отправляем свой профиль в ответ (с isInitiator=true)
	if isInitiator {
		log.Printf("[Profile] Отправка своего профиля пиру %s...", peerID.String()[:8])
		if err := pes.sendLocalProfile(stream, true); err != nil {
			log.Printf("[Profile] Ошибка отправки своего профиля: %v", err)
			return nil, fmt.Errorf("ошибка отправки профиля: %w", err)
		}
		log.Printf("[Profile] ✅ Свой профиль отправлен %s", peerID.String()[:8])
	}

	// Преобразуем в модель
	profile := &models.Profile{
		OwnerType:      models.OwnerTypeRemote,
		PeerID:         response.PeerID,
		Username:       response.Username,
		Title:          response.Title,
		AvatarPath:     "", // Путь будет установлен после сохранения аватара локально
		BackgroundPath: response.BackgroundPath,
		ContentChar:    response.ContentChar,
		PinnedUUIDs:    "[]",
	}

	if len(response.PinnedUUIDs) > 0 {
		uuidsJSON, err := json.Marshal(response.PinnedUUIDs)
		if err == nil {
			profile.PinnedUUIDs = string(uuidsJSON)
			log.Printf("[Profile] 📌 Получены PinnedUUIDs от пира %s: %d элементов, UUIDs=%v",
				peerID.String()[:8], len(response.PinnedUUIDs), response.PinnedUUIDs)
		} else {
			log.Printf("[Profile] ❌ Ошибка маршалинга PinnedUUIDs: %v", err)
		}
	} else {
		log.Printf("[Profile] ℹ️ Пир %s не имеет закреплённых элементов", peerID.String()[:8])
	}

	now := time.Now()
	profile.CachedAt = &now

	// ✅ СОХРАНЯЕМ профиль сервера в БД и Peerstore
	if err := pes.savePeerProfile(profile, response.PublicKey, response.Signature); err != nil {
		log.Printf("[Profile] Предупреждение: не удалось сохранить профиль сервера: %v", err)
	} else {
		log.Printf("[Profile] ✅ Профиль сервера %s сохранён в БД и Peerstore", response.PeerID[:8])
	}

	// Сохраняем ключ шифрования пира
	if len(response.EncryptionKey) > 0 {
		pes.SetPeerEncryptionKey(peerID, response.EncryptionKey)
		log.Printf("[Profile] 🔑 Ключ шифрования сохранён для %s (len=%d, key[0]=%d)", peerID.String()[:8], len(response.EncryptionKey), response.EncryptionKey[0])
	} else {
		log.Printf("[Profile] ⚠️ Ключ шифрования не получен от %s", peerID.String()[:8])
	}

	result := &ProfileWithSignature{
		Profile:   profile,
		PublicKey: response.PublicKey,
		Signature: response.Signature,
	}

	// Загружаем аватар если он есть и данные получены
	if len(response.AvatarData) > 0 {
		go func() {
			// Проверяем, существует ли уже файл аватара
			existingAvatar, err := filesystem.GetAvatar(peerID.String())
			if err == nil && existingAvatar != "" {
				log.Printf("[Profile] Аватар уже загружен для %s: %s", peerID.String()[:8], existingAvatar)
				// Обновляем путь к аватару в профиле, даже если аватар уже загружен
				remoteProfile, err := queries.GetRemoteProfile(peerID.String())
				if err == nil && remoteProfile != nil && remoteProfile.AvatarPath != existingAvatar {
					remoteProfile.AvatarPath = existingAvatar
					if err := queries.UpdateRemoteProfile(remoteProfile); err != nil {
						log.Printf("[Profile] Не удалось обновить путь к аватару в БД: %v", err)
					} else {
						log.Printf("[Profile] ✅ Путь к аватару обновлён в БД: %s", existingAvatar)
					}
				}
				return
			}

			// Сохраняем аватар
			filePath, err := filesystem.SaveAvatar(peerID.String(), response.AvatarData)
			if err != nil {
				log.Printf("[Profile] Не удалось сохранить аватар от %s: %v", peerID.String()[:8], err)
			} else {
				log.Printf("[Profile] ✅ Аватар сохранён: %s", filePath)

				// Обновляем путь к аватару в профиле (асинхронно)
				// Профиль уже сохранён, обновляем только avatar_path
				remoteProfile, err := queries.GetRemoteProfile(peerID.String())
				if err == nil && remoteProfile != nil {
					remoteProfile.AvatarPath = filePath
					if err := queries.UpdateRemoteProfile(remoteProfile); err != nil {
						log.Printf("[Profile] Не удалось обновить путь к аватару в БД: %v", err)
					} else {
						log.Printf("[Profile] ✅ Путь к аватару обновлён в БД")
					}
				}
			}
		}()
	} else if response.AvatarPath != "" {
		log.Printf("[Profile] ⚠️ Аватар не загружен (путь: %s, данные: пустые)", response.AvatarPath)
	}

	return result, nil
}

// sendLocalProfile отправляет локальный профиль в стрим
func (pes *ExchangeService) sendLocalProfile(stream network.Stream, isInitiator bool) error {
	localProfile, err := queries.GetLocalProfile()
	if err != nil {
		return fmt.Errorf("ошибка получения локального профиля: %w", err)
	}

	localKeys, err := queries.GetProfileKeys(localProfile.ID)
	if err != nil {
		return fmt.Errorf("ошибка получения ключей: %w", err)
	}

	pinnedItems, err := queries.GetPinnedItems()
	if err != nil {
		pinnedItems = []*models.Item{}
	}

	var pinnedUUIDs []string
	for _, item := range pinnedItems {
		if item.ElementUUID != "" {
			pinnedUUIDs = append(pinnedUUIDs, item.ElementUUID)
		} else {
			log.Printf("[Profile] ⚠️ Элемент ID=%d, title=%q не имеет ElementUUID, пропускаем", item.ID, item.Title)
		}
	}
	log.Printf("[Profile] 📌 PinnedUUIDs для отправки: %d элементов, UUIDs=%v", len(pinnedUUIDs), pinnedUUIDs)

	// Читаем данные аватара если он есть
	var avatarData []byte
	if localProfile.AvatarPath != "" {
		avatarBytes, err := os.ReadFile(localProfile.AvatarPath)
		if err == nil {
			avatarData = avatarBytes
			log.Printf("[Profile] Аватар загружен (%d байт) для отправки", len(avatarData))
		} else {
			log.Printf("[Profile] Предупреждение: не удалось прочитать аватар: %v", err)
		}
	}

	response := &ProfileResponse{
		PeerID:         localProfile.PeerID,
		Username:       localProfile.Username,
		Title:          localProfile.Title,
		AvatarPath:     localProfile.AvatarPath,
		AvatarHash:     avatar.GetAvatarHashFromProfile(localProfile.AvatarPath), // ✅ Хеш для отдельной загрузки
		AvatarData:     avatarData,                                               // ⚠️ Устаревает, оставляем для совместимости
		BackgroundPath: localProfile.BackgroundPath,
		ContentChar:    localProfile.ContentChar,
		PinnedUUIDs:    pinnedUUIDs,
		PublicKey:      localKeys.PublicKey,
		Timestamp:      time.Now().UnixNano(),
		IsInitiator:    isInitiator,
		EncryptionKey:  pes.localEncryptKey,
	}

	log.Printf("[Profile] 📸 AvatarHash=%q для отправки", response.AvatarHash)

	if isInitiator {
		signature, err := pes.signProfile(localProfile)
		if err == nil {
			response.Signature = signature
		}
	}

	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("ошибка сериализации: %w", err)
	}

	writer := bufio.NewWriter(stream)
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("ошибка записи: %w", err)
	}
	return writer.Flush()
}

// savePeerProfile сохраняет профиль пира в базу данных
func (pes *ExchangeService) savePeerProfile(profile *models.Profile, publicKey, signature []byte) error {
	log.Printf("[Profile] savePeerProfile: peer_id=%s, username=%s, owner_type=%s, avatar_path=%q",
		profile.PeerID[:8], profile.Username, profile.OwnerType, profile.AvatarPath)

	// Сохраняем публичный ключ в Peerstore для проверки подписи
	peerID, err := peer.Decode(profile.PeerID)
	if err == nil && len(publicKey) > 0 {
		pubKey, err := crypto.UnmarshalPublicKey(publicKey)
		if err == nil {
			if addKeyErr := pes.host.Peerstore().AddPubKey(peerID, pubKey); addKeyErr != nil {
				log.Printf("[Profile] ⚠️ Ошибка добавления публичного ключа в Peerstore: %v", addKeyErr)
			} else {
				log.Printf("[Profile] ✅ Публичный ключ сохранён в Peerstore для %s", profile.PeerID[:8])
			}
		} else {
			log.Printf("[Profile] ⚠️ Ошибка распаковки публичного ключа: %v", err)
		}
	} else if err != nil {
		log.Printf("[Profile] ⚠️ Ошибка декодирования PeerID: %v", err)
	}

	// Проверяем, есть ли уже профиль
	existing, err := queries.GetProfileByPeerID(profile.PeerID)
	if err == nil && existing != nil {
		log.Printf("[Profile] Профиль уже существует: owner_type=%s, username=%s, avatar_path=%q",
			existing.OwnerType, existing.Username, existing.AvatarPath)
		// Профиль существует - обновляем, но сохраняем существующий avatar_path
		// Аватар будет обновлён асинхронно после сохранения файла
		profile.AvatarPath = existing.AvatarPath
		if err := queries.UpdateRemoteProfile(profile); err != nil {
			return fmt.Errorf("ошибка обновления профиля: %w", err)
		}
		log.Printf("[Profile] ✅ Профиль обновлён: %s (username: %s, avatar_path сохранён: %q)",
			profile.PeerID[:8], profile.Username, existing.AvatarPath)
		return nil
	}

	// Профиль не найден - создаём
	log.Printf("[Profile] Создаём новый профиль...")
	err = queries.CreateRemoteProfile(profile)
	if err != nil {
		// Если UNIQUE constraint - профиль уже создан другим потоком
		if contains(err.Error(), "UNIQUE constraint") {
			log.Printf("[Profile] Профиль уже создан в другом потоке (UNIQUE constraint), используем существующий...")
			// Получаем существующий профиль и обновляем его
			existing, err = queries.GetProfileByPeerID(profile.PeerID)
			if err == nil && existing != nil {
				profile.AvatarPath = existing.AvatarPath
				if err := queries.UpdateRemoteProfile(profile); err != nil {
					return fmt.Errorf("ошибка обновления профиля после UNIQUE constraint: %w", err)
				}
				log.Printf("[Profile] ✅ Профиль обновлён (после UNIQUE): %s (username: %s)", profile.PeerID[:8], profile.Username)
				return nil
			}
			log.Printf("[Profile] ℹ️ Профиль уже существует: %s", profile.PeerID[:8])
			return nil
		}
		log.Printf("[Profile] ❌ Ошибка создания профиля: %v", err)
		return fmt.Errorf("ошибка создания профиля: %w", err)
	}

	log.Printf("[Profile] ✅ Профиль создан: %s (username: %s)", profile.PeerID[:8], profile.Username)

	// Сохраняем ключи
	if len(publicKey) > 0 {
		key := &models.ProfileKey{
			ProfileID:      profile.ID,
			PublicKey:      publicKey,
			Signature:      signature,
			IsKeyEncrypted: false,
		}
		// Проверяем, существуют ли уже ключи
		exists, err := queries.ProfileKeysExists(profile.ID)
		if err != nil {
			exists = false
		}
		if exists {
			if err := queries.UpdateProfileKeys(key); err != nil {
				return fmt.Errorf("ошибка обновления ключей: %w", err)
			}
		} else {
			if err := queries.CreateProfileKeys(key); err != nil {
				return fmt.Errorf("ошибка сохранения ключей: %w", err)
			}
		}
	}

	// Обновляем контакт если существует
	contact, err := queries.GetContactByPeerID(profile.PeerID)
	if err == nil && contact != nil {
		// Обновляем multiaddr контакта
		if err := queries.UpdateContactByPeerID(profile.PeerID, contact.Multiaddr); err != nil {
			log.Printf("Предупреждение: не удалось обновить контакт: %v", err)
		}
	}

	return nil
}

// signProfile подписывает профиль локального пользователя
func (pes *ExchangeService) signProfile(profile *models.Profile) ([]byte, error) {
	if pes.localPrivKey == nil {
		return nil, fmt.Errorf("приватный ключ не установлен")
	}

	// Данные для подписи
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		profile.PeerID,
		profile.Username,
		profile.Title,
		profile.AvatarPath,
		profile.ContentChar,
		profile.PinnedUUIDs,
	)

	// Подписываем
	signature, err := pes.localPrivKey.Sign([]byte(data))
	if err != nil {
		return nil, fmt.Errorf("ошибка подписи: %w", err)
	}

	return signature, nil
}

// VerifyProfileSignature проверяет подпись профиля
func (pes *ExchangeService) VerifyProfileSignature(profile *models.Profile, publicKey, signature []byte) (bool, error) {
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
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		profile.PeerID,
		profile.Username,
		profile.Title,
		profile.AvatarPath,
		profile.ContentChar,
		profile.PinnedUUIDs,
	)

	// Проверяем подпись
	valid, err := pubKey.Verify([]byte(data), signature)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки подписи: %w", err)
	}

	return valid, nil
}

// GetFullProfile возвращает полный локальный профиль с подписью
func (pes *ExchangeService) GetFullProfile() (*ProfileWithSignature, error) {
	profile, err := queries.GetLocalProfile()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения профиля: %w", err)
	}

	keys, err := queries.GetProfileKeys(profile.ID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения ключей: %w", err)
	}

	signature, err := pes.signProfile(profile)
	if err != nil {
		return nil, fmt.Errorf("ошибка подписи профиля: %w", err)
	}

	return &ProfileWithSignature{
		Profile:   profile,
		PublicKey: keys.PublicKey,
		Signature: signature,
	}, nil
}

// RequestProfilesForAllContacts запрашивает профили для всех контактов
func (pes *ExchangeService) RequestProfilesForAllContacts(ctx context.Context) {
	contacts, err := queries.GetAllContacts()
	if err != nil {
		log.Printf("Ошибка получения контактов: %v", err)
		return
	}

	for _, contact := range contacts {
		peerID, err := peer.Decode(contact.PeerID)
		if err != nil {
			log.Printf("Ошибка декодирования PeerID %s: %v", contact.PeerID, err)
			continue
		}

		// Проверяем подключение
		if pes.host.Network().Connectedness(peerID) != network.Connected {
			log.Printf("Пир %s не подключён, пропускаем", peerID)
			continue
		}

		// Запрашиваем профиль
		go func(p peer.ID) {
			if _, err := pes.RequestPeerProfile(ctx, p); err != nil {
				log.Printf("Ошибка получения профиля от %s: %v", p, err)
			}
		}(peerID)
	}
}

// SaveAvatarFromData сохраняет аватарку из полученных данных
func (pes *ExchangeService) SaveAvatarFromData(peerID string, avatarData []byte) (string, error) {
	if len(avatarData) == 0 {
		return "", nil
	}

	// Сохраняем аватарку в файловую систему
	filePath, err := filesystem.SaveAvatar(peerID, avatarData)
	if err != nil {
		return "", fmt.Errorf("ошибка сохранения аватарки: %w", err)
	}

	// Обновляем профиль в БД
	profile, err := queries.GetProfileByPeerID(peerID)
	if err == nil && profile != nil {
		if err := queries.UpdateLocalProfileField("avatar_path", filePath); err != nil {
			log.Printf("Предупреждение: не удалось обновить путь к аватарке: %v", err)
		}
	}

	return filePath, nil
}

// SendAvatar отправляет аватарку пиру
func (pes *ExchangeService) SendAvatar(ctx context.Context, peerID peer.ID, avatarPath string) (string, error) {
	transferSvc := pes.getTransferService()

	if transferSvc == nil {
		return "", fmt.Errorf("сервис передачи файлов не инициализирован")
	}

	if avatarPath == "" {
		return "", fmt.Errorf("путь к аватарке пуст")
	}

	// Извлекаем имя файла из пути
	fileName := avatarPath
	if len(fileName) > 50 {
		fileName = fileName[len(fileName)-50:]
	}

	// Отправляем аватарку через transfer service
	return transferSvc.SendAvatar(ctx, peerID, avatarPath, fileName)
}

// contains проверяет, содержит ли строка подстроку
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// generateEncryptionKey генерирует ключ для симметричного шифрования
func generateEncryptionKey() []byte {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		// Fallback к детерминированному ключу в случае ошибки
		return []byte("projectt-chat-encryption-key-32")
	}
	return key
}

// GetPeerEncryptionKey возвращает ключ шифрования для пира
func (pes *ExchangeService) GetPeerEncryptionKey(peerID peer.ID) []byte {
	pes.mu.RLock()
	defer pes.mu.RUnlock()
	return pes.peerEncryptKeys[peerID]
}

// SetPeerEncryptionKey сохраняет ключ шифрования для пира
func (pes *ExchangeService) SetPeerEncryptionKey(peerID peer.ID, key []byte) {
	pes.mu.Lock()
	defer pes.mu.Unlock()
	pes.peerEncryptKeys[peerID] = key
}

// GetLocalEncryptionKey возвращает локальный ключ шифрования
func (pes *ExchangeService) GetLocalEncryptionKey() []byte {
	return pes.localEncryptKey
}

// downloadAvatarSeparately загружает аватарку через отдельный протокол
func (pes *ExchangeService) downloadAvatarSeparately(remotePeer peer.ID, avatarHash string) {
	// Проверяем, существует ли уже файл аватара
	existingAvatar, err := filesystem.GetAvatar(remotePeer.String())
	if err == nil && existingAvatar != "" {
		log.Printf("[Profile] 📸 Аватар уже загружен: %s", existingAvatar)
		return
	}

	// Получаем Avatar сервис
	avatarSvc := pes.getAvatarService()
	if avatarSvc == nil {
		log.Printf("[Profile] ❌ Avatar сервис не инициализирован")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Запрашиваем аватарку у пира
	avatarData, err := avatarSvc.RequestAvatar(ctx, remotePeer, avatarHash)
	if err != nil {
		log.Printf("[Profile] ❌ Ошибка загрузки аватарки: %v", err)
		return
	}

	if len(avatarData) == 0 {
		log.Printf("[Profile] ⚠️ Аватарка не найдена у пира")
		return
	}

	// Сохраняем аватарку
	filePath, err := filesystem.SaveAvatar(remotePeer.String(), avatarData)
	if err != nil {
		log.Printf("[Profile] ❌ Ошибка сохранения аватарки: %v", err)
		return
	}

	log.Printf("[Profile] ✅ Аватарка загружена: %s (%d байт)", filePath, len(avatarData))

	// Обновляем путь в профиле
	remoteProfile, err := queries.GetRemoteProfile(remotePeer.String())
	if err == nil && remoteProfile != nil {
		remoteProfile.AvatarPath = filePath
		if err := queries.UpdateRemoteProfile(remoteProfile); err != nil {
			log.Printf("[Profile] ❌ Ошибка обновления пути к аватарке: %v", err)
		} else {
			log.Printf("[Profile] ✅ Путь к аватарке обновлён в БД")
		}
	}

	// Уведомляем UI
	if pes.uiP2P != nil {
		pes.uiP2P.OnProfileUpdated(remotePeer.String())
	}
}

// saveAvatarFromProfileData сохраняет аватарку из данных профиля (устаревший способ)
func (pes *ExchangeService) saveAvatarFromProfileData(remotePeer peer.ID, avatarData []byte) {
	if len(avatarData) == 0 {
		return
	}

	filePath, err := filesystem.SaveAvatar(remotePeer.String(), avatarData)
	if err != nil {
		log.Printf("[Profile] ❌ Ошибка сохранения аватарки: %v", err)
		return
	}

	log.Printf("[Profile] ✅ Аватарка сохранена (устаревший способ): %s", filePath)

	// Обновляем путь в профиле
	remoteProfile, err := queries.GetRemoteProfile(remotePeer.String())
	if err == nil && remoteProfile != nil {
		remoteProfile.AvatarPath = filePath
		if err := queries.UpdateRemoteProfile(remoteProfile); err != nil {
			log.Printf("[Profile] ❌ Ошибка обновления пути к аватарке: %v", err)
		}
	}
}

// getAvatarService возвращает Avatar сервис
func (pes *ExchangeService) getAvatarService() *avatar.Service {
	if pes.avatarSvc != nil {
		return pes.avatarSvc
	}
	return nil
}
