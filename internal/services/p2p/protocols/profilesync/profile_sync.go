// Package profilesync предоставляет протокол для синхронизации списков известных профилей между пирами.
//
// Протокол использует 3 раунда для эффективного обмена:
//   - Раунд 1: обмен (root_hash, count) — быстрый выход при идентичных списках
//   - Раунд 2: обмен списками PeerID — когда размеры списков близки
//   - Раунд 3: передача ID из меньшего списка — когда размеры сильно отличаются
package profilesync

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"projectT/internal/storage/database/models"
	"projectT/internal/storage/database/queries"
)

// ProtocolID идентификатор протокола синхронизации профилей
const ProtocolID = "/projectt/profile-sync/1.0.0"

// SyncRound определяет текущий раунд синхронизации
type SyncRound uint8

const (
	SyncRoundHeader   SyncRound = 1
	SyncRoundPeerIDs  SyncRound = 2
	SyncRoundProfiles SyncRound = 3
)

// SyncHeader заголовок для раунда 1 — хеш и количество профилей
type SyncHeader struct {
	RootHash string `json:"root_hash"` // SHA-256 от конкатенации отсортированных PeerID
	Count    int    `json:"count"`     // Количество известных remote профилей
}

// PeerIDList список PeerID для раунда 2
type PeerIDList struct {
	PeerIDs []string `json:"peer_ids"`
}

// ProfileSummary минимальная информация о профиле для раунда 3
type ProfileSummary struct {
	PeerID      string `json:"peer_id"`
	Username    string `json:"username"`
	Multiaddr   string `json:"multiaddr"`
	AddressType string `json:"address_type"` // bootstrap, contact, discovered
}

// ProfileProvider интерфейс для получения и сохранения профилей
type ProfileProvider interface {
	GetRemotePeerIDs() ([]string, error)
	GetProfileSummaries(peerIDs []string) ([]*ProfileSummary, error)
	SaveProfileSummary(summary *ProfileSummary) error
}

// DefaultProvider провайдер по умолчанию (работает с БД)
type DefaultProvider struct{}

// GetRemotePeerIDs возвращает отсортированный список PeerID всех remote профилей
func (p *DefaultProvider) GetRemotePeerIDs() ([]string, error) {
	profiles, err := queries.GetAllRemoteProfiles()
	if err != nil {
		return nil, err
	}

	peerIDs := make([]string, 0, len(profiles))
	for _, prof := range profiles {
		if prof.PeerID != "" {
			peerIDs = append(peerIDs, prof.PeerID)
		}
	}
	sort.Strings(peerIDs)
	return peerIDs, nil
}

// GetProfileSummaries возвращает summaries для указанных PeerID
func (p *DefaultProvider) GetProfileSummaries(peerIDs []string) ([]*ProfileSummary, error) {
	summaries := make([]*ProfileSummary, 0, len(peerIDs))

	for _, peerID := range peerIDs {
		profile, err := queries.GetProfileByPeerID(peerID)
		if err != nil {
			continue
		}

		// Получаем адрес из peer_addresses
		addresses, err := queries.GetActivePeerAddresses()
		if err != nil {
			continue
		}

		multiaddr := ""
		addressType := "discovered"
		for _, addr := range addresses {
			if addr.PeerID == peerID {
				multiaddr = addr.Multiaddr
				addressType = addr.AddressType
				break
			}
		}

		summaries = append(summaries, &ProfileSummary{
			PeerID:      profile.PeerID,
			Username:    profile.Username,
			Multiaddr:   multiaddr,
			AddressType: addressType,
		})
	}

	return summaries, nil
}

// SaveProfileSummary сохраняет профиль из summary
func (p *DefaultProvider) SaveProfileSummary(summary *ProfileSummary) error {
	if summary.PeerID == "" {
		return nil
	}

	// Проверяем, существует ли уже профиль
	existing, err := queries.GetProfileByPeerID(summary.PeerID)
	if err == nil && existing != nil {
		// Профиль существует — обновляем username если изменился
		if existing.Username != summary.Username {
			profile := &models.Profile{
				PeerID:   summary.PeerID,
				Username: summary.Username,
			}
			return queries.UpdateProfileBasic(profile)
		}
		return nil
	}

	// Профиль не существует — создаём через AddPeerAddressWithProfile
	if summary.Multiaddr != "" {
		return queries.AddPeerAddressWithProfile(
			summary.PeerID,
			summary.Multiaddr,
			summary.AddressType,
			"profile_sync",
			summary.Username,
		)
	}

	// Если адреса нет, создаём только профиль
	_, err = queries.GetProfileByPeerID(summary.PeerID)
	if err != nil {
		// Создаём профиль напрямую
		profile := &models.Profile{
			OwnerType: models.OwnerTypeRemote,
			PeerID:    summary.PeerID,
			Username:  summary.Username,
		}
		return queries.CreateRemoteProfile(profile)
	}

	return nil
}

// SyncService сервис для синхронизации профилей
type SyncService struct {
	host     host.Host
	provider ProfileProvider
}

// NewSyncService создаёт сервис синхронизации профилей
func NewSyncService(h host.Host, provider ProfileProvider) *SyncService {
	if provider == nil {
		provider = &DefaultProvider{}
	}

	return &SyncService{
		host:     h,
		provider: provider,
	}
}

// Start запускает сервис, регистрируя обработчик входящих запросов
func (s *SyncService) Start() error {
	s.host.SetStreamHandler(ProtocolID, s.handleSync)
	return nil
}

// Stop останавливает сервис
func (s *SyncService) Stop() error {
	s.host.RemoveStreamHandler(ProtocolID)
	return nil
}

// computeRootHash вычисляет SHA-256 хеш от конкатенации отсортированных PeerID
func (s *SyncService) computeRootHash() (string, int, error) {
	peerIDs, err := s.provider.GetRemotePeerIDs()
	if err != nil {
		return "", 0, err
	}

	// Конкатенация всех PeerID с разделителем
	concat := strings.Join(peerIDs, "|")
	hash := sha256.Sum256([]byte(concat))

	return hex.EncodeToString(hash[:]), len(peerIDs), nil
}

// handleSync обрабатывает входящий запрос синхронизации
func (s *SyncService) handleSync(stream network.Stream) {
	defer func() { _ = stream.Close() }()

	remotePeer := stream.Conn().RemotePeer()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_ = s.runSyncProtocol(ctx, stream, remotePeer)
}

// SyncWithPeer запускает синхронизацию профилей с указанным пиром
func (s *SyncService) SyncWithPeer(ctx context.Context, peerID peer.ID) error {
	stream, err := s.host.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		return fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer func() { _ = stream.Close() }()

	if err := s.runSyncProtocol(ctx, stream, peerID); err != nil {
		return err
	}

	return nil
}

// runSyncProtocol выполняет полный протокол синхронизации
func (s *SyncService) runSyncProtocol(ctx context.Context, stream network.Stream, remotePeer peer.ID) error {
	_ = ctx // используем таймаут стрима

	// === РАУНД 1: Обмен заголовками ===
	ourHash, ourCount, err := s.computeRootHash()
	if err != nil {
		return fmt.Errorf("ошибка вычисления хеша: %w", err)
	}

	// Отправляем свой header
	writer := bufio.NewWriter(stream)
	ourHeader := SyncHeader{RootHash: ourHash, Count: ourCount}
	if err := json.NewEncoder(writer).Encode(ourHeader); err != nil {
		return fmt.Errorf("ошибка отправки header: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("ошибка flush header: %w", err)
	}

	// Читаем header собеседника
	reader := bufio.NewReader(stream)
	var theirHeader SyncHeader
	if err := json.NewDecoder(reader).Decode(&theirHeader); err != nil {
		return fmt.Errorf("ошибка чтения header: %w", err)
	}

	// Если хеши и количество совпали — списки идентичны
	if ourHash == theirHeader.RootHash && ourCount == theirHeader.Count {
		return nil
	}

	// === РАУНД 2: Обмен списками PeerID ===
	// Определяем стратегию: если размеры близки (< 2x) — оба отправляют списки
	// Если сильно отличаются — только меньший отправляет свой список
	ratio := float64(ourCount) / float64(max(theirHeader.Count, 1))
	similarSize := ratio >= 0.5 && ratio <= 2.0

	var ourPeerIDs []string
	var theirPeerIDs []string

	if similarSize {
		// Оба отправляют списки PeerID
		ourPeerIDs, err = s.provider.GetRemotePeerIDs()
		if err != nil {
			return fmt.Errorf("ошибка получения PeerID: %w", err)
		}

		// Отправляем наш список
		if err := json.NewEncoder(writer).Encode(PeerIDList{PeerIDs: ourPeerIDs}); err != nil {
			return fmt.Errorf("ошибка отправки PeerID list: %w", err)
		}
		if err := writer.Flush(); err != nil {
			return fmt.Errorf("ошибка flush PeerID list: %w", err)
		}

		// Читаем их список
		var theirList PeerIDList
		if err := json.NewDecoder(reader).Decode(&theirList); err != nil {
			return fmt.Errorf("ошибка чтения PeerID list: %w", err)
		}
		theirPeerIDs = theirList.PeerIDs
	} else {
		// Только меньший список отправляет свои ID
		if ourCount <= theirHeader.Count {
			// Мы — меньший, отправляем свои ID
			ourPeerIDs, err = s.provider.GetRemotePeerIDs()
			if err != nil {
				return fmt.Errorf("ошибка получения PeerID: %w", err)
			}

			if err := json.NewEncoder(writer).Encode(PeerIDList{PeerIDs: ourPeerIDs}); err != nil {
				return fmt.Errorf("ошибка отправки PeerID list: %w", err)
			}
			if err := writer.Flush(); err != nil {
				return fmt.Errorf("ошибка flush PeerID list: %w", err)
			}

			// Читаем их ответ — какие ID им неизвестны
			var unknownList PeerIDList
			if err := json.NewDecoder(reader).Decode(&unknownList); err != nil {
				return fmt.Errorf("ошибка чтения unknown list: %w", err)
			}

			// Отправляем профили для неизвестных им ID
			return s.sendMissingProfiles(writer, unknownList.PeerIDs)
		} else {
			// Они — меньший, читаем их ID
			var theirList PeerIDList
			if err := json.NewDecoder(reader).Decode(&theirList); err != nil {
				return fmt.Errorf("ошибка чтения PeerID list: %w", err)
			}
			theirPeerIDs = theirList.PeerIDs

			// Вычисляем какие из их ID нам неизвестны
			ourPeerIDs, err = s.provider.GetRemotePeerIDs()
			if err != nil {
				return fmt.Errorf("ошибка получения PeerID: %w", err)
			}

			ourSet := make(map[string]bool)
			for _, id := range ourPeerIDs {
				ourSet[id] = true
			}

			unknownIDs := make([]string, 0)
			for _, id := range theirPeerIDs {
				if !ourSet[id] {
					unknownIDs = append(unknownIDs, id)
				}
			}

			// Отправляем обратно список неизвестных нам ID
			if err := json.NewEncoder(writer).Encode(PeerIDList{PeerIDs: unknownIDs}); err != nil {
				return fmt.Errorf("ошибка отправки unknown list: %w", err)
			}
			if err := writer.Flush(); err != nil {
				return fmt.Errorf("ошибка flush unknown list: %w", err)
			}

			// Читаем профили для неизвестных ID
			return s.receiveMissingProfiles(reader)
		}
	}

	// === РАУНД 3: Обмен недостающими профилями ===
	// Вычисляем какие профили нам нужны (есть у них, нет у нас)
	ourSet := make(map[string]bool)
	for _, id := range ourPeerIDs {
		ourSet[id] = true
	}

	theirSet := make(map[string]bool)
	for _, id := range theirPeerIDs {
		theirSet[id] = true
	}

	missingForThem := make([]string, 0)
	for _, id := range ourPeerIDs {
		if !theirSet[id] {
			missingForThem = append(missingForThem, id)
		}
	}

	// Обмен missing lists и профилями через goroutines для избежания deadlock
	missingForUsCh := make(chan []string, 1)
	profilesErrCh := make(chan error, 2)

	// Горутина для чтения их missing list
	go func() {
		var list PeerIDList
		if err := json.NewDecoder(reader).Decode(&list); err != nil {
			profilesErrCh <- fmt.Errorf("ошибка чтения missing for us: %w", err)
			return
		}
		missingForUsCh <- list.PeerIDs
	}()

	// Отправляем наши missing для них
	if err := json.NewEncoder(writer).Encode(PeerIDList{PeerIDs: missingForThem}); err != nil {
		return fmt.Errorf("ошибка отправки missing for them: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("ошибка flush missing for them: %w", err)
	}

	// Ждём их missing list
	select {
	case <-missingForUsCh:
	case err := <-profilesErrCh:
		return err
	}

	// Горутина для чтения профилей которые нам нужны
	go func() {
		profilesErrCh <- s.receiveMissingProfiles(reader)
	}()

	// Отправляем профили которые им нужны
	if err := s.sendMissingProfiles(writer, missingForThem); err != nil {
		return fmt.Errorf("ошибка отправки missing profiles: %w", err)
	}

	// Ждём завершения чтения профилей
	if err := <-profilesErrCh; err != nil {
		return fmt.Errorf("ошибка получения missing profiles: %w", err)
	}

	return nil
}

// sendMissingProfiles отправляет профили для указанных PeerID
func (s *SyncService) sendMissingProfiles(writer *bufio.Writer, peerIDs []string) error {
	summaries, err := s.provider.GetProfileSummaries(peerIDs)
	if err != nil {
		return fmt.Errorf("ошибка получения summaries: %w", err)
	}

	for _, summary := range summaries {
		if err := json.NewEncoder(writer).Encode(summary); err != nil {
			return fmt.Errorf("ошибка отправки профиля %s: %w", summary.PeerID[:8], err)
		}
	}

	// Отправляем маркер конца
	if err := json.NewEncoder(writer).Encode(nil); err != nil {
		return fmt.Errorf("ошибка отправки EOF маркера: %w", err)
	}

	return writer.Flush()
}

// receiveMissingProfiles читает и сохраняет входящие профили
func (s *SyncService) receiveMissingProfiles(reader *bufio.Reader) error {
	decoder := json.NewDecoder(reader)
	receivedCount := 0

	for {
		var summary *ProfileSummary
		if err := decoder.Decode(&summary); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("ошибка чтения профиля: %w", err)
		}

		// nil — маркер конца
		if summary == nil {
			break
		}

		if summary.PeerID == "" {
			continue
		}

		if err := s.provider.SaveProfileSummary(summary); err != nil {
			continue
		}

		receivedCount++
	}

	return nil
}

// max возвращает большее из двух чисел
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
