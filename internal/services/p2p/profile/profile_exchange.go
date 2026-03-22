// Package profile содержит сервисы для обмена профилями между пирами.
package profile

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"projectT/internal/services/p2p/transfer"
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
	AvatarPath     string   `json:"avatar_path"`
	AvatarData     []byte   `json:"avatar_data,omitempty"` // Данные аватара в base64
	BackgroundPath string   `json:"background_path"`
	ContentChar    string   `json:"content_characteristic"`
	PinnedUUIDs    []string `json:"pinned_uuids"` // UUID избранных элементов
	PublicKey      []byte   `json:"public_key"`
	Signature      []byte   `json:"signature,omitempty"` // Подпись профиля
	Timestamp      int64    `json:"timestamp"`
	IsInitiator    bool     `json:"is_initiator"` // Роль отправителя профиля
}

// ProfileWithSignature профиль вместе с подписью для проверки
type ProfileWithSignature struct {
	Profile   *models.Profile
	PublicKey []byte
	Signature []byte
}

// ExchangeService сервис для обмена профилями между пирами
type ExchangeService struct {
	host         host.Host
	localPrivKey crypto.PrivKey
	localPubKey  crypto.PubKey
	transferSvc  *transfer.Service // сервис передачи файлов для аватарок
	connSvc      interface {
		MarkProfilePending(peer.ID)
		MarkProfileComplete(peer.ID)
	} // сервис подключений для отслеживания статуса профиля
}

// NewExchangeService создаёт сервис обмена профилями
func NewExchangeService(host host.Host, privKey crypto.PrivKey, pubKey crypto.PubKey) *ExchangeService {
	return &ExchangeService{
		host:         host,
		localPrivKey: privKey,
		localPubKey:  pubKey,
	}
}

// SetTransferService устанавливает сервис передачи файлов
func (pes *ExchangeService) SetTransferService(transferSvc *transfer.Service) {
	pes.transferSvc = transferSvc
}

// SetConnectionService устанавливает сервис подключений
func (pes *ExchangeService) SetConnectionService(connSvc interface {
	MarkProfilePending(peer.ID)
	MarkProfileComplete(peer.ID)
}) {
	pes.connSvc = connSvc
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
	log.Printf("[Profile] ✅ Профиль инициатора получен от %s за %v (username: %s, аватар: %d байт)", remotePeer.String()[:8], time.Since(startTime), response.Username, avatarSize)

	// Сохраняем профиль инициатора
	profile := &models.Profile{
		OwnerType:      models.OwnerTypeRemote,
		PeerID:         response.PeerID,
		Username:       response.Username,
		Title:          response.Title,
		AvatarPath:     response.AvatarPath,
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

	// Загружаем аватар если он есть и данные получены
	if len(response.AvatarData) > 0 {
		go func() {
			// Проверяем, существует ли уже файл аватара
			existingAvatar, err := filesystem.GetAvatar(remotePeer.String())
			if err == nil && existingAvatar != "" {
				log.Printf("[Profile] Аватар уже загружен для %s: %s", remotePeer.String()[:8], existingAvatar)
				return
			}

			// Сохраняем аватар
			filePath, err := filesystem.SaveAvatar(remotePeer.String(), response.AvatarData)
			if err != nil {
				log.Printf("[Profile] Не удалось сохранить аватар от %s: %v", remotePeer.String()[:8], err)
			} else {
				log.Printf("[Profile] ✅ Аватар сохранён: %s", filePath)

				// Обновляем путь к аватару в профиле (асинхронно)
				remoteProfile, err := queries.GetRemoteProfile(remotePeer.String())
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
	log.Printf("[Profile] ✅ Профиль сервера получен от %s за %v (username: %s, аватар: %d байт)", peerID.String()[:8], time.Since(startTime), response.Username, avatarSize)

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
		AvatarPath:     response.AvatarPath,
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
		}
	}

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
		AvatarData:     avatarData,
		BackgroundPath: localProfile.BackgroundPath,
		ContentChar:    localProfile.ContentChar,
		PinnedUUIDs:    pinnedUUIDs,
		PublicKey:      localKeys.PublicKey,
		Timestamp:      time.Now().UnixNano(),
		IsInitiator:    isInitiator,
	}

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
	log.Printf("[Profile] savePeerProfile: peer_id=%s, username=%s, owner_type=%s", profile.PeerID[:8], profile.Username, profile.OwnerType)

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
		log.Printf("[Profile] Профиль уже существует: owner_type=%s, username=%s", existing.OwnerType, existing.Username)
		// Профиль существует - обновляем
		if err := queries.UpdateRemoteProfile(profile); err != nil {
			return fmt.Errorf("ошибка обновления профиля: %w", err)
		}
		log.Printf("[Profile] ✅ Профиль обновлён: %s (username: %s)", profile.PeerID[:8], profile.Username)
		return nil
	}

	// Профиль не найден - создаём
	log.Printf("[Profile] Создаём новый профиль...")
	err = queries.CreateRemoteProfile(profile)
	if err != nil {
		// Если UNIQUE constraint - профиль уже создан другим потоком
		if contains(err.Error(), "UNIQUE constraint") {
			log.Printf("[Profile] UNIQUE constraint! Получаем существующий профиль...")
			// Получаем существующий профиль и обновляем его
			existing, err = queries.GetProfileByPeerID(profile.PeerID)
			if err == nil && existing != nil {
				if err := queries.UpdateRemoteProfile(profile); err != nil {
					return fmt.Errorf("ошибка обновления профиля после UNIQUE constraint: %w", err)
				}
				log.Printf("[Profile] ✅ Профиль обновлён (после UNIQUE): %s (username: %s)", profile.PeerID[:8], profile.Username)
				return nil
			}
			log.Printf("[Profile] ⚠️ Профиль уже существует но не получен: %s", profile.PeerID[:8])
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
