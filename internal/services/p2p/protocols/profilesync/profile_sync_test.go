package profilesync

import (
	"sort"
	"testing"
)

func TestComputeRootHash(t *testing.T) {
	provider := &mockProvider{
		peerIDs: []string{"peer1", "peer2", "peer3"},
	}

	svc := &SyncService{provider: provider}

	hash1, count1, err := svc.computeRootHash()
	if err != nil {
		t.Fatalf("ошибка вычисления хеша: %v", err)
	}

	if count1 != 3 {
		t.Errorf("ожидали count=3, получили %d", count1)
	}

	if len(hash1) != 64 { // SHA-256 в hex = 64 символа
		t.Errorf("ожидали hash длины 64, получили %d", len(hash1))
	}

	// Тот же список → тот же хеш
	hash2, count2, err := svc.computeRootHash()
	if err != nil {
		t.Fatalf("ошибка вычисления хеша (второй раз): %v", err)
	}

	if hash1 != hash2 {
		t.Error("хэши должны совпадать для одинаковых списков")
	}

	if count1 != count2 {
		t.Error("количество должно совпадать для одинаковых списков")
	}
}

func TestComputeRootHashOrderIndependent(t *testing.T) {
	provider1 := &mockProvider{
		peerIDs: []string{"peer1", "peer2", "peer3"},
	}
	provider2 := &mockProvider{
		peerIDs: []string{"peer3", "peer1", "peer2"},
	}

	svc1 := &SyncService{provider: provider1}
	svc2 := &SyncService{provider: provider2}

	hash1, _, _ := svc1.computeRootHash()
	hash2, _, _ := svc2.computeRootHash()

	if hash1 != hash2 {
		t.Error("хэши должны совпадать независимо от порядка ввода (сортировка внутри)")
	}
}

func TestComputeRootHashDifferent(t *testing.T) {
	provider1 := &mockProvider{
		peerIDs: []string{"peer1", "peer2"},
	}
	provider2 := &mockProvider{
		peerIDs: []string{"peer1", "peer3"},
	}

	svc1 := &SyncService{provider: provider1}
	svc2 := &SyncService{provider: provider2}

	hash1, _, _ := svc1.computeRootHash()
	hash2, _, _ := svc2.computeRootHash()

	if hash1 == hash2 {
		t.Error("хэши должны отличаться для разных списков")
	}
}

func TestComputeRootHashEmpty(t *testing.T) {
	provider := &mockProvider{
		peerIDs: []string{},
	}

	svc := &SyncService{provider: provider}

	hash, count, err := svc.computeRootHash()
	if err != nil {
		t.Fatalf("ошибка вычисления хеша: %v", err)
	}

	if count != 0 {
		t.Errorf("ожидали count=0, получили %d", count)
	}

	// Хеш пустой строки
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if hash != expected {
		t.Errorf("ожидали хеш пустой строки %s, получили %s", expected, hash)
	}
}

func TestSaveProfileSummaryNew(t *testing.T) {
	provider := &mockProvider{
		existingProfiles: make(map[string]bool),
		savedProfiles:    make(map[string]*ProfileSummary),
	}

	summary := &ProfileSummary{
		PeerID:      "new-peer-1",
		Username:    "NewUser",
		Multiaddr:   "/ip4/127.0.0.1/tcp/4001",
		AddressType: "discovered",
	}

	if err := provider.SaveProfileSummary(summary); err != nil {
		t.Fatalf("ошибка сохранения профиля: %v", err)
	}

	if _, ok := provider.savedProfiles["new-peer-1"]; !ok {
		t.Error("профиль должен быть сохранён")
	}
}

func TestSaveProfileSummaryExisting(t *testing.T) {
	provider := &mockProvider{
		existingProfiles: map[string]bool{"existing-peer": true},
		savedProfiles:    make(map[string]*ProfileSummary),
		updatedProfiles:  make(map[string]*ProfileSummary),
	}

	summary := &ProfileSummary{
		PeerID:   "existing-peer",
		Username: "UpdatedName",
	}

	if err := provider.SaveProfileSummary(summary); err != nil {
		t.Fatalf("ошибка сохранения профиля: %v", err)
	}

	if _, ok := provider.updatedProfiles["existing-peer"]; !ok {
		t.Error("существующий профиль должен быть обновлён")
	}
}

func TestGetProfileSummaries(t *testing.T) {
	provider := &mockProvider{
		profiles: map[string]*ProfileSummary{
			"peer1": {PeerID: "peer1", Username: "User1", Multiaddr: "/ip4/1.2.3.4/tcp/4001"},
			"peer2": {PeerID: "peer2", Username: "User2", Multiaddr: "/ip4/5.6.7.8/tcp/4001"},
		},
	}

	summaries, err := provider.GetProfileSummaries([]string{"peer1", "peer2"})
	if err != nil {
		t.Fatalf("ошибка получения summaries: %v", err)
	}

	if len(summaries) != 2 {
		t.Fatalf("ожидали 2 summary, получили %d", len(summaries))
	}
}

// mockProvider — мок для тестирования
type mockProvider struct {
	peerIDs          []string
	profiles         map[string]*ProfileSummary
	existingProfiles map[string]bool
	savedProfiles    map[string]*ProfileSummary
	updatedProfiles  map[string]*ProfileSummary
}

func (p *mockProvider) GetRemotePeerIDs() ([]string, error) {
	ids := make([]string, len(p.peerIDs))
	copy(ids, p.peerIDs)
	sort.Strings(ids)
	return ids, nil
}

func (p *mockProvider) GetProfileSummaries(peerIDs []string) ([]*ProfileSummary, error) {
	summaries := make([]*ProfileSummary, 0, len(peerIDs))
	for _, id := range peerIDs {
		if summary, ok := p.profiles[id]; ok {
			summaries = append(summaries, summary)
		}
	}
	return summaries, nil
}

func (p *mockProvider) SaveProfileSummary(summary *ProfileSummary) error {
	if p.existingProfiles[summary.PeerID] {
		if p.updatedProfiles != nil {
			p.updatedProfiles[summary.PeerID] = summary
		}
		return nil
	}
	if p.savedProfiles != nil {
		p.savedProfiles[summary.PeerID] = summary
	}
	return nil
}
