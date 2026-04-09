// Package avatar предоставляет сервис для загрузки аватарок по запросу
package avatar

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"projectT/internal/storage/database/queries"
	"projectT/internal/storage/filesystem"
)

// ProtocolID идентификатор протокола загрузки аватарок
const ProtocolID = "/projectt/avatar/1.0.0"

// AvatarRequest запрос аватарки по хешу
type AvatarRequest struct {
	AvatarHash string `json:"avatar_hash"` // Хеш аватарки (имя файла)
}

// AvatarResponse ответ с аватаркой
type AvatarResponse struct {
	AvatarData []byte `json:"avatar_data,omitempty"` // Данные аватарки
	Found      bool   `json:"found"`                 // Найдена ли аватарка
}

// Service сервис для загрузки аватарок
type Service struct {
	host host.Host
}

// NewService создаёт сервис загрузки аватарок
func NewService(h host.Host) *Service {
	return &Service{
		host: h,
	}
}

// Start запускает сервис
func (s *Service) Start() error {
	s.host.SetStreamHandler(ProtocolID, s.handleAvatarRequest)
	log.Println("[Avatar] Сервис загрузки аватарок запущен")
	return nil
}

// Stop останавливает сервис
func (s *Service) Stop() error {
	log.Println("[Avatar] Сервис загрузки аватарок остановлен")
	return nil
}

// handleAvatarRequest обрабатывает входящий запрос аватарки
func (s *Service) handleAvatarRequest(stream network.Stream) {
	defer func() { _ = stream.Close() }()

	remotePeer := stream.Conn().RemotePeer()
	log.Printf("[Avatar] 📥 Запрос аватарки от %s", remotePeer.String()[:8])

	// Читаем запрос
	reader := bufio.NewReader(stream)
	reqData, err := io.ReadAll(reader)
	if err != nil {
		log.Printf("[Avatar] ❌ Ошибка чтения запроса: %v", err)
		return
	}

	var req AvatarRequest
	if err := json.Unmarshal(reqData, &req); err != nil {
		log.Printf("[Avatar] ❌ Ошибка десериализации запроса: %v", err)
		return
	}

	log.Printf("[Avatar] 📋 Запрос аватарки: hash=%q", req.AvatarHash)

	// Ищем аватарку
	response := &AvatarResponse{
		Found: false,
	}

	if req.AvatarHash != "" {
		// Читаем файл из хранилища
		avatarData, err := filesystem.ReadFile(req.AvatarHash)
		if err == nil && len(avatarData) > 0 {
			response.Found = true
			response.AvatarData = avatarData
			log.Printf("[Avatar] ✅ Аватарка найдена: %d байт", len(avatarData))
		} else {
			log.Printf("[Avatar] ⚠️ Аватарка не найдена: %v", err)
		}
	}

	// Отправляем ответ
	writer := bufio.NewWriter(stream)
	respData, _ := json.Marshal(response)
	if _, err := writer.Write(respData); err != nil {
		log.Printf("[Avatar] ❌ Ошибка отправки ответа: %v", err)
		return
	}

	if err := writer.Flush(); err != nil {
		log.Printf("[Avatar] ❌ Ошибка flush: %v", err)
		return
	}

	log.Printf("[Avatar] ✅ Ответ отправлен: found=%v", response.Found)
}

// RequestAvatar запрашивает аватарку у пира
func (s *Service) RequestAvatar(ctx context.Context, peerID peer.ID, avatarHash string) ([]byte, error) {
	log.Printf("[Avatar] 🔌 Запрос аватарки: hash=%q, peer=%s", avatarHash, peerID.String()[:8])

	stream, err := s.host.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer func() { _ = stream.Close() }()

	// Отправляем запрос
	req := &AvatarRequest{
		AvatarHash: avatarHash,
	}
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
		log.Printf("[Avatar] ⚠️ Предупреждение: не удалось установить таймаут: %v", err)
	}

	// Читаем ответ
	reader := bufio.NewReader(stream)
	var resp AvatarResponse

	if err := json.NewDecoder(reader).Decode(&resp); err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	if !resp.Found || len(resp.AvatarData) == 0 {
		log.Printf("[Avatar] ⚠️ Аватарка не найдена у пира")
		return nil, nil
	}

	log.Printf("[Avatar] ✅ Аватарка получена: %d байт", len(resp.AvatarData))
	return resp.AvatarData, nil
}

// GetAvatarHashFromProfile извлекает хеш аватарки из профиля
func GetAvatarHashFromProfile(avatarPath string) string {
	if avatarPath == "" {
		return ""
	}

	// Путь может быть полным или именем файла
	// Если это путь, извлекаем имя файла
	for i := len(avatarPath) - 1; i >= 0; i-- {
		if avatarPath[i] == '\\' || avatarPath[i] == '/' {
			return avatarPath[i+1:]
		}
	}

	return avatarPath
}

// SaveAvatarData сохраняет аватарку в хранилище
func SaveAvatarData(peerID string, avatarData []byte) (string, error) {
	// Сохраняем аватарку в файловую систему
	filePath, err := filesystem.SaveAvatar(peerID, avatarData)
	if err != nil {
		return "", fmt.Errorf("ошибка сохранения аватарки: %w", err)
	}

	log.Printf("[Avatar] ✅ Аватарка сохранена: %s", filePath)
	return filePath, nil
}

// UpdateProfileAvatarPath обновляет путь к аватарке в профиле
func UpdateProfileAvatarPath(peerID string, avatarPath string) error {
	profile, err := queries.GetProfileByPeerID(peerID)
	if err != nil {
		return fmt.Errorf("ошибка получения профиля: %w", err)
	}

	if profile == nil {
		return fmt.Errorf("профиль не найден")
	}

	// Обновляем путь к аватарке
	if profile.OwnerType == "remote" {
		remoteProfile, err := queries.GetRemoteProfile(peerID)
		if err != nil {
			return fmt.Errorf("ошибка получения remote профиля: %w", err)
		}

		if remoteProfile != nil {
			remoteProfile.AvatarPath = avatarPath
			if err := queries.UpdateRemoteProfile(remoteProfile); err != nil {
				return fmt.Errorf("ошибка обновления профиля: %w", err)
			}
			log.Printf("[Avatar] ✅ Путь к аватарке обновлён в БД: %s", avatarPath)
		}
	} else {
		if err := queries.UpdateLocalProfileField("avatar_path", avatarPath); err != nil {
			return fmt.Errorf("ошибка обновления локального профиля: %w", err)
		}
		log.Printf("[Avatar] ✅ Путь к аватарке обновлён в локальном профиле: %s", avatarPath)
	}

	return nil
}

// detectMimeType определяет MIME-тип файла по содержимому
func detectMimeType(fileBytes []byte) string {
	return http.DetectContentType(fileBytes)
}

// GetMimeType определяет MIME-тип аватарки
func GetMimeType(avatarData []byte) string {
	return detectMimeType(avatarData)
}
