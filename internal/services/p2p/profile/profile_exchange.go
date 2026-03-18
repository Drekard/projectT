// Package profile содержит сервисы для обмена профилями между пирами.
package profile

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
}

// ProfileResponse ответ с профилем (полная версия)
type ProfileResponse struct {
	PeerID         string   `json:"peer_id"`
	Username       string   `json:"username"`
	Title          string   `json:"title"`
	AvatarPath     string   `json:"avatar_path"`
	BackgroundPath string   `json:"background_path"`
	ContentChar    string   `json:"content_characteristic"`
	PinnedUUIDs    []string `json:"pinned_uuids"` // UUID избранных элементов
	PublicKey      []byte   `json:"public_key"`
	Signature      []byte   `json:"signature,omitempty"` // Подпись профиля
	Timestamp      int64    `json:"timestamp"`
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

// handleProfileRequest обрабатывает входящий запрос профиля
func (pes *ExchangeService) handleProfileRequest(stream network.Stream) {
	defer stream.Close()

	remotePeer := stream.Conn().RemotePeer()
	log.Printf("Получен запрос профиля от: %s", remotePeer.String())

	// Читаем запрос
	reader := bufio.NewReader(stream)
	reqData, err := io.ReadAll(reader)
	if err != nil {
		log.Printf("Ошибка чтения запроса профиля: %v", err)
		return
	}

	var req ProfileRequest
	if len(reqData) > 0 {
		if err := json.Unmarshal(reqData, &req); err != nil {
			log.Printf("Ошибка десериализации запроса: %v", err)
			// Продолжаем с запросом по умолчанию
		}
	}

	// Получаем локальный профиль
	localProfile, err := queries.GetLocalProfile()
	if err != nil {
		log.Printf("Ошибка получения локального профиля: %v", err)
		return
	}

	// Получаем ключи
	localKeys, err := queries.GetProfileKeys(localProfile.ID)
	if err != nil {
		log.Printf("Ошибка получения ключей: %v", err)
		return
	}

	// Загружаем избранные элементы для получения их UUID
	pinnedItems, err := queries.GetPinnedItems()
	if err != nil {
		log.Printf("Ошибка загрузки pinned items: %v", err)
		pinnedItems = []*models.Item{}
	}

	// Извлекаем UUID избранных элементов
	var pinnedUUIDs []string
	for _, item := range pinnedItems {
		if item.ElementUUID != "" {
			pinnedUUIDs = append(pinnedUUIDs, item.ElementUUID)
		}
	}

	// Формируем ответ
	response := &ProfileResponse{
		PeerID:         localProfile.PeerID,
		Username:       localProfile.Username,
		Title:          localProfile.Title,
		AvatarPath:     localProfile.AvatarPath,
		BackgroundPath: localProfile.BackgroundPath,
		ContentChar:    localProfile.ContentChar,
		PinnedUUIDs:    pinnedUUIDs,
		PublicKey:      localKeys.PublicKey,
		Timestamp:      time.Now().UnixNano(),
	}

	// Подписываем профиль если запрошено
	if req.RequestFull {
		signature, err := pes.signProfile(localProfile)
		if err != nil {
			log.Printf("Ошибка подписи профиля: %v", err)
		} else {
			response.Signature = signature
		}
	}

	// Сериализуем ответ
	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("Ошибка сериализации профиля: %v", err)
		return
	}

	// Отправляем ответ
	writer := bufio.NewWriter(stream)
	if _, err := writer.Write(data); err != nil {
		log.Printf("Ошибка отправки профиля: %v", err)
		return
	}

	if err := writer.Flush(); err != nil {
		log.Printf("Ошибка flush профиля: %v", err)
		return
	}

	log.Printf("Отправлен профиль пиру %s", remotePeer)
}

// RequestPeerProfile запрашивает профиль у удалённого пира
func (pes *ExchangeService) RequestPeerProfile(ctx context.Context, peerID peer.ID) (*ProfileWithSignature, error) {
	// Создаём стрим
	stream, err := pes.host.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer stream.Close()

	// Отправляем запрос
	req := &ProfileRequest{RequestFull: true}
	reqData, _ := json.Marshal(req)

	writer := bufio.NewWriter(stream)
	if _, err := writer.Write(reqData); err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("ошибка flush: %w", err)
	}

	// Устанавливаем таймаут
	if err := stream.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		log.Printf("Предупреждение: не удалось установить таймаут: %v", err)
	}

	// Читаем ответ
	reader := bufio.NewReader(stream)
	response := &ProfileResponse{}

	if err := json.NewDecoder(reader).Decode(response); err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
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
		PinnedUUIDs:    "[]", // Будет обновлено в savePeerProfile
	}

	// Сериализуем pinned_uuids в JSON
	if len(response.PinnedUUIDs) > 0 {
		uuidsJSON, err := json.Marshal(response.PinnedUUIDs)
		if err == nil {
			profile.PinnedUUIDs = string(uuidsJSON)
		}
	}

	now := time.Now()
	profile.CachedAt = &now

	// Проверяем подпись если есть
	if len(response.Signature) > 0 {
		// TODO: Проверка подписи
		log.Printf("Получена подпись профиля, проверка...")
	}

	// Сохраняем профиль в БД
	if err := pes.savePeerProfile(profile, response.PublicKey, response.Signature); err != nil {
		log.Printf("Предупреждение: не удалось сохранить профиль: %v", err)
	}

	log.Printf("Получен профиль от %s: username=%s", peerID, response.Username)
	return &ProfileWithSignature{
		Profile:   profile,
		PublicKey: response.PublicKey,
		Signature: response.Signature,
	}, nil
}

// savePeerProfile сохраняет профиль пира в базу данных
func (pes *ExchangeService) savePeerProfile(profile *models.Profile, publicKey, signature []byte) error {
	// Проверяем, есть ли уже профиль
	existing, err := queries.GetProfileByPeerID(profile.PeerID)
	if err == nil && existing != nil {
		// Профиль существует - обновляем
		if err := queries.UpdateRemoteProfile(profile); err != nil {
			return fmt.Errorf("ошибка обновления профиля: %w", err)
		}
	} else {
		// Профиль не найден - создаём
		if err := queries.CreateRemoteProfile(profile); err != nil {
			return fmt.Errorf("ошибка создания профиля: %w", err)
		}
	}

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
