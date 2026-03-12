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

// ProfileProtocolID идентификатор протокола обмена профилями (версия 2.0 для новой схемы)
const ProfileProtocolID = "/projectt/profile/2.0.0"

// ProfileRequest запрос профиля
type ProfileRequest struct {
	RequestFull bool `json:"request_full"` // Запрос полного профиля с подписью
}

// ProfileResponse ответ с профилем (полная версия)
type ProfileResponse struct {
	PeerID         string `json:"peer_id"`
	Username       string `json:"username"`
	Status         string `json:"status"`
	AvatarPath     string `json:"avatar_path"`
	BackgroundPath string `json:"background_path"`
	ContentChar    string `json:"content_characteristic"`
	DemoElements   string `json:"demo_elements"`
	PublicKey      []byte `json:"public_key"`
	Signature      []byte `json:"signature,omitempty"` // Подпись профиля
	Timestamp      int64  `json:"timestamp"`
}

// ProfileWithSignature профиль вместе с подписью для проверки
type ProfileWithSignature struct {
	Profile   *models.Profile
	PublicKey []byte
	Signature []byte
}

// ProfileExchangeService сервис для обмена профилями между пирами
type ProfileExchangeService struct {
	host         host.Host
	localPrivKey crypto.PrivKey
	localPubKey  crypto.PubKey
}

// NewProfileExchangeService создаёт сервис обмена профилями
func NewProfileExchangeService(host host.Host, privKey crypto.PrivKey, pubKey crypto.PubKey) *ProfileExchangeService {
	return &ProfileExchangeService{
		host:         host,
		localPrivKey: privKey,
		localPubKey:  pubKey,
	}
}

// Start запускает сервис обмена профилями
func (pes *ProfileExchangeService) Start() error {
	pes.host.SetStreamHandler(ProfileProtocolID, pes.handleProfileRequest)
	log.Println("ProfileExchangeService v2.0 запущен")
	return nil
}

// Stop останавливает сервис
func (pes *ProfileExchangeService) Stop() error {
	log.Println("ProfileExchangeService остановлен")
	return nil
}

// handleProfileRequest обрабатывает входящий запрос профиля
func (pes *ProfileExchangeService) handleProfileRequest(stream network.Stream) {
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

	// Формируем ответ
	response := &ProfileResponse{
		PeerID:         localProfile.PeerID,
		Username:       localProfile.Username,
		Status:         localProfile.Status,
		AvatarPath:     localProfile.AvatarPath,
		BackgroundPath: localProfile.BackgroundPath,
		ContentChar:    localProfile.ContentChar,
		DemoElements:   localProfile.DemoElements,
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
func (pes *ProfileExchangeService) RequestPeerProfile(ctx context.Context, peerID peer.ID) (*ProfileWithSignature, error) {
	// Создаём стрим
	stream, err := pes.host.NewStream(ctx, peerID, ProfileProtocolID)
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
		Status:         response.Status,
		AvatarPath:     response.AvatarPath,
		BackgroundPath: response.BackgroundPath,
		ContentChar:    response.ContentChar,
		DemoElements:   response.DemoElements,
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
func (pes *ProfileExchangeService) savePeerProfile(profile *models.Profile, publicKey, signature []byte) error {
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
		// Обновляем имя и аватар контакта
		if err := queries.UpdateContactByPeerID(profile.PeerID, profile.Username, profile.AvatarPath); err != nil {
			log.Printf("Предупреждение: не удалось обновить контакт: %v", err)
		}
	}

	return nil
}

// signProfile подписывает профиль локального пользователя
func (pes *ProfileExchangeService) signProfile(profile *models.Profile) ([]byte, error) {
	if pes.localPrivKey == nil {
		return nil, fmt.Errorf("приватный ключ не установлен")
	}

	// Данные для подписи
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		profile.PeerID,
		profile.Username,
		profile.Status,
		profile.AvatarPath,
		profile.ContentChar,
		profile.DemoElements,
	)

	// Подписываем
	signature, err := pes.localPrivKey.Sign([]byte(data))
	if err != nil {
		return nil, fmt.Errorf("ошибка подписи: %w", err)
	}

	return signature, nil
}

// VerifyProfileSignature проверяет подпись профиля
func (pes *ProfileExchangeService) VerifyProfileSignature(profile *models.Profile, publicKey, signature []byte) (bool, error) {
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
		profile.Status,
		profile.AvatarPath,
		profile.ContentChar,
		profile.DemoElements,
	)

	// Проверяем подпись
	valid, err := pubKey.Verify([]byte(data), signature)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки подписи: %w", err)
	}

	return valid, nil
}

// GetFullProfile возвращает полный локальный профиль с подписью
func (pes *ProfileExchangeService) GetFullProfile() (*ProfileWithSignature, error) {
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
func (pes *ProfileExchangeService) RequestProfilesForAllContacts(ctx context.Context) {
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
func (pes *ProfileExchangeService) SaveAvatarFromData(peerID string, avatarData []byte) (string, error) {
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
