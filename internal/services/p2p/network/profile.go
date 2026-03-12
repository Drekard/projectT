// Package network предоставляет функции для управления P2P профилем
package network

import (
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	p2pcrypto "projectT/internal/services/crypto"
	p2p "projectT/internal/services/p2p"
	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// ProfileManager управляет загрузкой и сохранением P2P профиля
type ProfileManager struct {
	masterPassword string
}

// NewProfileManager создаёт менеджер профилей
func NewProfileManager() *ProfileManager {
	return &ProfileManager{}
}

// SetMasterPassword устанавливает мастер-пароль для шифрования
func (pm *ProfileManager) SetMasterPassword(password string) {
	pm.masterPassword = password
}

// LoadOrCreateProfile загружает существующий профиль или создаёт новый
func (pm *ProfileManager) LoadOrCreateProfile() (*models.P2PProfile, error) {
	// Проверяем существование профиля
	exists, err := queries.P2PProfileExists()
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки профиля: %w", err)
	}

	if exists {
		// Загружаем существующий
		profile, err := queries.GetP2PProfile()
		if err != nil {
			return nil, fmt.Errorf("ошибка загрузки профиля: %w", err)
		}
		log.Printf("Загружен существующий P2P профиль: %s", profile.PeerID)
		return profile, nil
	}

	// Создаём новый профиль
	log.Println("Создание нового P2P профиля...")

	// Генерируем ключи
	privKey, pubKey, err := GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации ключей: %w", err)
	}

	// Получаем PeerID
	peerID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения PeerID: %w", err)
	}

	// Сериализуем публичный ключ
	pubKeyBytes, err := crypto.MarshalPublicKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации публичного ключа: %w", err)
	}

	// Шифруем приватный ключ с паролем
	var privKeyBytes []byte
	var isEncrypted bool
	if pm.masterPassword != "" {
		privKeyRaw, err := crypto.MarshalPrivateKey(privKey)
		if err != nil {
			return nil, fmt.Errorf("ошибка сериализации приватного ключа: %w", err)
		}
		privKeyBytes, err = p2pcrypto.EncryptPrivateKey(privKeyRaw, pm.masterPassword)
		if err != nil {
			return nil, fmt.Errorf("ошибка шифрования приватного ключа: %w", err)
		}
		isEncrypted = true
		log.Println("Приватный ключ зашифрован")
	} else {
		// Без шифрования (не рекомендуется)
		privKeyBytes, err = crypto.MarshalPrivateKey(privKey)
		if err != nil {
			return nil, fmt.Errorf("ошибка сериализации приватного ключа: %w", err)
		}
		isEncrypted = false
		log.Println("Предупреждение: приватный ключ сохранён без шифрования")
	}

	// Получаем имя пользователя из профиля
	var username string
	localProfile, err := queries.GetLocalProfile()
	if err != nil {
		username = fmt.Sprintf("User_%s", peerID.String()[:8])
	} else {
		username = localProfile.Username
	}

	// Создаём профиль
	profile := &models.P2PProfile{
		ID:             1,
		PeerID:         peerID.String(),
		PrivateKey:     privKeyBytes,
		PublicKey:      pubKeyBytes,
		IsKeyEncrypted: isEncrypted,
		Username:       username,
		Status:         "online",
	}

	if err := queries.CreateP2PProfile(profile); err != nil {
		return nil, fmt.Errorf("ошибка сохранения профиля: %w", err)
	}

	log.Printf("Создан новый P2P профиль: %s", profile.PeerID)
	return profile, nil
}

// UpdateProfileAddrs обновляет адреса прослушивания в профиле
func UpdateProfileAddrs(peerID peer.ID, addrs []multiaddr.Multiaddr) error {
	var prefixedAddrs []string
	for _, addr := range addrs {
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr.String(), peerID.String())
		// Добавляем префикс к адресу
		prefixedAddr := p2p.ProtocolPrefix + "://" + fullAddr
		prefixedAddrs = append(prefixedAddrs, prefixedAddr)
	}

	addrsStr := joinStrings(prefixedAddrs, "|")
	return queries.UpdateP2PProfileField("listen_addrs", addrsStr)
}

// joinStrings соединяет строки с разделителем (встроенная функция для избежания импорта strings)
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}
