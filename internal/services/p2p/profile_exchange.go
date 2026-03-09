// Package p2p содержит сервисы для P2P связи на базе libp2p.
package p2p

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// ProfileProtocolID идентификатор протокола обмена профилями
const ProfileProtocolID = "/" + ProtocolPrefix + "/profile/1.0.0"

// ProfileRequest запрос профиля
type ProfileRequest struct{}

// ProfileResponse ответ с профилем
type ProfileResponse struct {
	Username   string `json:"username"`
	AvatarPath string `json:"avatar_path"`
	Status     string `json:"status"`
	PublicKey  []byte `json:"public_key,omitempty"`
}

// ProfileExchangeService сервис для обмена профилями между пирами
type ProfileExchangeService struct {
	host          host.Host
	localUsername string
	localAvatar   string
	localStatus   string
	localPubKey   []byte
}

// NewProfileExchangeService создаёт сервис обмена профилями
func NewProfileExchangeService(host host.Host, username, avatarPath, status string, pubKey []byte) *ProfileExchangeService {
	return &ProfileExchangeService{
		host:          host,
		localUsername: username,
		localAvatar:   avatarPath,
		localStatus:   status,
		localPubKey:   pubKey,
	}
}

// Start запускает сервис обмена профилями
func (pes *ProfileExchangeService) Start() error {
	pes.host.SetStreamHandler(ProfileProtocolID, pes.handleProfileRequest)
	log.Println("ProfileExchangeService запущен")
	return nil
}

// Stop останавливает сервис
func (pes *ProfileExchangeService) Stop() error {
	log.Println("ProfileExchangeService остановлен")
	return nil
}

// UpdateLocalProfile обновляет локальный профиль
func (pes *ProfileExchangeService) UpdateLocalProfile(username, avatarPath, status string, pubKey []byte) {
	pes.localUsername = username
	pes.localAvatar = avatarPath
	pes.localStatus = status
	pes.localPubKey = pubKey
}

// handleProfileRequest обрабатывает входящий запрос профиля
func (pes *ProfileExchangeService) handleProfileRequest(stream network.Stream) {
	defer stream.Close()

	remotePeer := stream.Conn().RemotePeer()
	log.Printf("Получен запрос профиля от: %s", remotePeer.String())

	// Читаем запрос (может быть пустым)
	reader := bufio.NewReader(stream)
	reqBuf := make([]byte, 1)
	_, err := reader.Read(reqBuf)
	if err != nil && err.Error() != "EOF" {
		log.Printf("Ошибка чтения запроса профиля: %v", err)
	}

	// Формируем ответ
	response := &ProfileResponse{
		Username:  pes.localUsername,
		Status:    pes.localStatus,
		PublicKey: pes.localPubKey,
	}

	// Добавляем аватар если есть
	if pes.localAvatar != "" {
		response.AvatarPath = pes.localAvatar
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
func (pes *ProfileExchangeService) RequestPeerProfile(ctx context.Context, peerID peer.ID) (*ProfileResponse, error) {
	// Создаём стрим
	stream, err := pes.host.NewStream(ctx, peerID, ProfileProtocolID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer stream.Close()

	// Отправляем запрос (пустой)
	writer := bufio.NewWriter(stream)
	if _, err := writer.Write([]byte{0x01}); err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("ошибка flush: %w", err)
	}

	// Устанавливаем таймаут
	if err := stream.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		log.Printf("Предупреждение: не удалось установить таймаут: %v", err)
	}

	// Читаем ответ
	reader := bufio.NewReader(stream)
	decoder := json.NewDecoder(reader)
	response := &ProfileResponse{}

	if err := decoder.Decode(response); err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	// Сохраняем профиль в БД
	if err := pes.savePeerProfile(peerID, response); err != nil {
		log.Printf("Предупреждение: не удалось сохранить профиль: %v", err)
	}

	log.Printf("Получен профиль от %s: username=%s, avatar=%s", peerID, response.Username, response.AvatarPath)
	return response, nil
}

// savePeerProfile сохраняет профиль пира в базу данных
func (pes *ProfileExchangeService) savePeerProfile(peerID peer.ID, profile *ProfileResponse) error {
	// Проверяем, есть ли контакт
	_, err := queries.GetContactByPeerID(peerID.String())
	if err != nil {
		// Контакт не найден - создаём новый
		contact := &models.Contact{
			PeerID:     peerID.String(),
			Username:   profile.Username,
			AvatarPath: profile.AvatarPath,
			Status:     profile.Status,
			PublicKey:  profile.PublicKey,
		}
		if createErr := queries.CreateContact(contact); createErr != nil {
			return fmt.Errorf("ошибка создания контакта: %w", createErr)
		}
		return nil
	}

	// Контакт существует - обновляем профиль
	if err := queries.UpdateContactByPeerID(peerID.String(), profile.Username, profile.AvatarPath); err != nil {
		return fmt.Errorf("ошибка обновления контакта: %w", err)
	}

	return nil
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
