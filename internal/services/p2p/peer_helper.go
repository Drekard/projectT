// Package helper предоставляет сервис для режима помощника - хранение и передача адресов пиров
package p2p

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// PeerCommand команда для режима помощника
type PeerCommand string

const (
	// CmdRegister регистрация своего адреса
	CmdRegister PeerCommand = "register"
	// CmdAsk запрос адреса пира
	CmdAsk PeerCommand = "ask"
	// CmdList запрос списка пиров
	CmdList PeerCommand = "list"
	// CmdAck подтверждение
	CmdAck PeerCommand = "ack"
	// CmdError ошибка
	CmdError PeerCommand = "error"
)

// PeerRequest запрос к режиму помощника
type PeerRequest struct {
	Command PeerCommand `json:"command"`
	PeerID  string      `json:"peer_id,omitempty"`
}

// PeerResponse ответ от режима помощника
type PeerResponse struct {
	Command PeerCommand      `json:"command"`
	Success bool             `json:"success"`
	Data    *PeerAddressData `json:"data,omitempty"`
	List    []PeerEntry      `json:"list,omitempty"`
	Error   string           `json:"error,omitempty"`
}

// PeerAddressData данные об адресе пира
type PeerAddressData struct {
	PeerID  string `json:"peer_id"`
	Address string `json:"address"`
}

// PeerEntry запись о пире в хранилище
type PeerEntry struct {
	PeerID    string    `json:"peer_id"`
	Address   string    `json:"address"`
	Timestamp time.Time `json:"timestamp"`
}

// HelperConfig конфигурация режима помощника
type HelperConfig struct {
	// EntryTTL время жизни записи в хранилище
	EntryTTL time.Duration
	// MaxEntries максимальное количество записей
	MaxEntries int
	// CleanupInterval интервал очистки старых записей
	CleanupInterval time.Duration
}

// DefaultHelperConfig возвращает конфигурацию по умолчанию
func DefaultHelperConfig() *HelperConfig {
	return &HelperConfig{
		EntryTTL:        1 * time.Hour,
		MaxEntries:      1000,
		CleanupInterval: 10 * time.Minute,
	}
}

// Helper сервис режима помощника - хранит и передаёт адреса пиров
type Helper struct {
	host        host.Host
	config      *HelperConfig
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	peerStore   map[string]*PeerEntry // peerID -> PeerEntry
	requestChan chan *peerRequestCtx
}

// peerRequestCtx контекст запроса
type peerRequestCtx struct {
	stream   network.Stream
	response *PeerResponse
}

// NewHelper создаёт новый сервис режима помощника
func NewHelper(h host.Host, config *HelperConfig) *Helper {
	if config == nil {
		config = DefaultHelperConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Helper{
		host:        h,
		config:      config,
		ctx:         ctx,
		cancel:      cancel,
		peerStore:   make(map[string]*PeerEntry),
		requestChan: make(chan *peerRequestCtx, 100),
	}
}

// Start запускает сервис режима помощника
func (h *Helper) Start() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Устанавливаем обработчик входящих запросов
	h.host.SetStreamHandler(HelperProtocolID, h.handleHelperStream)

	// Запускаем очистку старых записей
	go h.startCleanup()

	// Запускаем обработчик запросов
	go h.processRequests()

	log.Printf("🤖 Режим ПОМОЩНИКА запущен")
	log.Printf("   Протокол: %s", HelperProtocolID)
	log.Printf("   Хранилище: %d записей (макс: %d)", len(h.peerStore), h.config.MaxEntries)

	return nil
}

// Stop останавливает сервис режима помощника
func (h *Helper) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.cancel()

	log.Printf("🤖 Режим ПОМОЩНИКА остановлен")
	return nil
}

// Register регистрирует адрес пира в хранилище
func (h *Helper) Register(peerID, address string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Проверяем лимит
	if len(h.peerStore) >= h.config.MaxEntries {
		// Удаляем самую старую запись
		h.removeOldestEntry()
	}

	h.peerStore[peerID] = &PeerEntry{
		PeerID:    peerID,
		Address:   address,
		Timestamp: time.Now(),
	}

	log.Printf("📝 [Helper] Зарегистрирован: %s -> %s", peerID, address)
	return nil
}

// Ask запрашивает адрес пира из хранилища
func (h *Helper) Ask(peerID string) (*PeerAddressData, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	entry, found := h.peerStore[peerID]
	if !found {
		return nil, false
	}

	return &PeerAddressData{
		PeerID:  entry.PeerID,
		Address: entry.Address,
	}, true
}

// List возвращает список всех зарегистрированных пиров
func (h *Helper) List() []PeerEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	list := make([]PeerEntry, 0, len(h.peerStore))
	for _, entry := range h.peerStore {
		list = append(list, *entry)
	}
	return list
}

// GetPeerCount возвращает количество зарегистрированных пиров
func (h *Helper) GetPeerCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.peerStore)
}

// GetPeerIDs возвращает список PeerID
func (h *Helper) GetPeerIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	ids := make([]string, 0, len(h.peerStore))
	for peerID := range h.peerStore {
		ids = append(ids, peerID)
	}
	return ids
}

// Remove удаляет пира из хранилища
func (h *Helper) Remove(peerID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.peerStore, peerID)
	log.Printf("🗑️  [Helper] Удалён: %s", peerID)
}

// Clear очищает всё хранилище
func (h *Helper) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.peerStore = make(map[string]*PeerEntry)
	log.Printf("🧹 [Helper] Хранилище очищено")
}

// handleHelperStream обрабатывает входящий запрос от другого пира
func (h *Helper) handleHelperStream(stream network.Stream) {
	defer stream.Close()

	remotePeer := stream.Conn().RemotePeer()
	log.Printf("📥 [Helper] Запрос от: %s", remotePeer.String())

	// Читаем запрос
	reader := bufio.NewReader(stream)
	var req PeerRequest
	if err := json.NewDecoder(reader).Decode(&req); err != nil {
		h.sendError(stream, fmt.Sprintf("ошибка декодирования запроса: %v", err))
		return
	}

	// Обрабатываем команду
	switch req.Command {
	case CmdRegister:
		h.handleRegister(stream, remotePeer.String())
	case CmdAsk:
		h.handleAsk(stream, req.PeerID)
	case CmdList:
		h.handleList(stream)
	default:
		h.sendError(stream, fmt.Sprintf("неизвестная команда: %s", req.Command))
	}
}

// handleRegister обрабатывает команду регистрации
func (h *Helper) handleRegister(stream network.Stream, peerID string) {
	// Получаем адрес пира из stream
	address := stream.Conn().RemoteMultiaddr().String() + "/p2p/" + peerID

	if err := h.Register(peerID, address); err != nil {
		h.sendError(stream, err.Error())
		return
	}

	h.sendResponse(stream, &PeerResponse{
		Command: CmdAck,
		Success: true,
		Data: &PeerAddressData{
			PeerID:  peerID,
			Address: address,
		},
	})
}

// handleAsk обрабатывает команду запроса адреса
func (h *Helper) handleAsk(stream network.Stream, peerID string) {
	data, found := h.Ask(peerID)
	if !found {
		h.sendResponse(stream, &PeerResponse{
			Command: CmdAsk,
			Success: false,
			Error:   "Пир не найден",
		})
		return
	}

	h.sendResponse(stream, &PeerResponse{
		Command: CmdAsk,
		Success: true,
		Data:    data,
	})
}

// handleList обрабатывает команду получения списка
func (h *Helper) handleList(stream network.Stream) {
	list := h.List()

	h.sendResponse(stream, &PeerResponse{
		Command: CmdList,
		Success: true,
		List:    list,
	})
}

// sendResponse отправляет ответ
func (h *Helper) sendResponse(stream network.Stream, resp *PeerResponse) {
	if err := json.NewEncoder(stream).Encode(resp); err != nil {
		log.Printf("❌ [Helper] Ошибка отправки ответа: %v", err)
	}
}

// sendError отправляет ошибку
func (h *Helper) sendError(stream network.Stream, errMsg string) {
	h.sendResponse(stream, &PeerResponse{
		Command: CmdError,
		Success: false,
		Error:   errMsg,
	})
}

// processRequests обрабатывает исходящие запросы
func (h *Helper) processRequests() {
	for {
		select {
		case <-h.ctx.Done():
			return
		case req := <-h.requestChan:
			h.handleOutgoingRequest(req)
		}
	}
}

// handleOutgoingRequest обрабатывает исходящий запрос к другому помощнику
func (h *Helper) handleOutgoingRequest(req *peerRequestCtx) {
	defer req.stream.Close()

	// Отправляем запрос
	if err := json.NewEncoder(req.stream).Encode(req.response); err != nil {
		log.Printf("❌ [Helper] Ошибка отправки запроса: %v", err)
		return
	}

	// Читаем ответ
	var resp PeerResponse
	if err := json.NewDecoder(req.stream).Decode(&resp); err != nil {
		log.Printf("❌ [Helper] Ошибка чтения ответа: %v", err)
		return
	}

	log.Printf("📤 [Helper] Получен ответ: %+v", resp)
}

// startCleanup запускает периодическую очистку старых записей
func (h *Helper) startCleanup() {
	ticker := time.NewTicker(h.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.cleanupOldEntries()
		}
	}
}

// cleanupOldEntries удаляет старые записи
func (h *Helper) cleanupOldEntries() {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	deleted := 0

	for peerID, entry := range h.peerStore {
		if now.Sub(entry.Timestamp) > h.config.EntryTTL {
			delete(h.peerStore, peerID)
			deleted++
		}
	}

	if deleted > 0 {
		log.Printf("🧹 [Helper] Удалено %d устаревших записей", deleted)
	}
}

// removeOldestEntry удаляет самую старую запись
func (h *Helper) removeOldestEntry() {
	var oldestID string
	var oldestTime time.Time

	for peerID, entry := range h.peerStore {
		if oldestID == "" || entry.Timestamp.Before(oldestTime) {
			oldestID = peerID
			oldestTime = entry.Timestamp
		}
	}

	if oldestID != "" {
		delete(h.peerStore, oldestID)
		log.Printf("🗑️  [Helper] Удалена старейшая запись: %s", oldestID)
	}
}

// RegisterOnRemote регистрирует свой адрес на удалённом помощнике
func (h *Helper) RegisterOnRemote(ctx context.Context, helperPeerID peer.ID) error {
	stream, err := h.host.NewStream(ctx, helperPeerID, HelperProtocolID)
	if err != nil {
		return fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer stream.Close()

	req := &PeerRequest{
		Command: CmdRegister,
		PeerID:  h.host.ID().String(),
	}

	if err := json.NewEncoder(stream).Encode(req); err != nil {
		return fmt.Errorf("ошибка отправки запроса: %w", err)
	}

	var resp PeerResponse
	if err := json.NewDecoder(stream).Decode(&resp); err != nil {
		return fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("ошибка регистрации: %s", resp.Error)
	}

	log.Printf("✅ [Helper] Зарегистрирован на удалённом помощнике: %s", helperPeerID)
	return nil
}

// AskFromRemote запрашивает адрес пира с удалённого помощника
func (h *Helper) AskFromRemote(ctx context.Context, helperPeerID peer.ID, targetPeerID string) (*PeerAddressData, error) {
	stream, err := h.host.NewStream(ctx, helperPeerID, HelperProtocolID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer stream.Close()

	req := &PeerRequest{
		Command: CmdAsk,
		PeerID:  targetPeerID,
	}

	if err := json.NewEncoder(stream).Encode(req); err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}

	var resp PeerResponse
	if err := json.NewDecoder(stream).Decode(&resp); err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("ошибка запроса: %s", resp.Error)
	}

	return resp.Data, nil
}

// ListFromRemote получает список пиров с удалённого помощника
func (h *Helper) ListFromRemote(ctx context.Context, helperPeerID peer.ID) ([]PeerEntry, error) {
	stream, err := h.host.NewStream(ctx, helperPeerID, HelperProtocolID)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания стрима: %w", err)
	}
	defer stream.Close()

	req := &PeerRequest{
		Command: CmdList,
	}

	if err := json.NewEncoder(stream).Encode(req); err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}

	var resp PeerResponse
	if err := json.NewDecoder(stream).Decode(&resp); err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("ошибка запроса: %s", resp.Error)
	}

	return resp.List, nil
}

// ConnectToPeerFromData подключается к пиру по данным из ответа
func (h *Helper) ConnectToPeerFromData(ctx context.Context, data *PeerAddressData) error {
	if data == nil {
		return fmt.Errorf("данные об адресе пустые")
	}

	addr, err := multiaddr.NewMultiaddr(data.Address)
	if err != nil {
		return fmt.Errorf("ошибка парсинга адреса: %w", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return fmt.Errorf("ошибка извлечения PeerID: %w", err)
	}

	// Добавляем в peerstore
	h.host.Peerstore().AddAddr(info.ID, addr, 5*time.Minute)

	// Подключаемся
	connectCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := h.host.Connect(connectCtx, *info); err != nil {
		return fmt.Errorf("ошибка подключения: %w", err)
	}

	log.Printf("✅ [Helper] Подключено к пиру: %s", data.PeerID)
	return nil
}

// ExportPeerData экспортирует данные пира для передачи
func (h *Helper) ExportPeerData(peerID string) *PeerAddressData {
	h.mu.RLock()
	defer h.mu.RUnlock()

	entry, found := h.peerStore[peerID]
	if !found {
		return nil
	}

	return &PeerAddressData{
		PeerID:  entry.PeerID,
		Address: entry.Address,
	}
}

// ImportPeerData импортирует данные пира в хранилище
func (h *Helper) ImportPeerData(data *PeerAddressData) error {
	if data == nil || data.PeerID == "" || data.Address == "" {
		return fmt.Errorf("некорректные данные")
	}

	return h.Register(data.PeerID, data.Address)
}
