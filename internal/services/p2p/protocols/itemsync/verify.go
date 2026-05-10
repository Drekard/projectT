package itemsync

import (
	"fmt"
	"log"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
	"projectT/internal/storage/filesystem"

	"github.com/libp2p/go-libp2p/core/crypto"
)

// signItem подписывает элемент
func (iss *Service) signItem(item *models.Item) ([]byte, error) {
	if iss.localPrivKey == nil {
		return nil, fmt.Errorf("приватный ключ не установлен")
	}

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

	pubKey, err := crypto.UnmarshalPublicKey(publicKey)
	if err != nil {
		return false, fmt.Errorf("ошибка восстановления публичного ключа: %w", err)
	}

	data := fmt.Sprintf("%s|%s|%s|%s",
		"element",
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
	if err := queries.DeleteRemoteItemsByPeer(peerID); err != nil {
		return fmt.Errorf("ошибка удаления элементов: %w", err)
	}

	if err := filesystem.DeleteRemoteItemFiles(peerID); err != nil {
		log.Printf("Предупреждение: не удалось удалить файлы: %v", err)
	}

	return nil
}
