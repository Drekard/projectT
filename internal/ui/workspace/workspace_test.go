package workspace

import (
	"testing"

	"projectT/internal/storage/database/models"
)

func TestWorkspace_NotifyRemoteModeChanged_Remote(t *testing.T) {
	ws := &Workspace{
		remoteProfilePeerID: "peer-123",
		remoteProfileName:   "John",
		remoteFolderPath:    nil,
	}

	var receivedIsRemote bool
	var receivedPeerID, receivedPeerName string
	var receivedPath []*models.Item

	ws.onRemoteModeChanged = func(isRemote bool, peerID, peerName string, path []*models.Item) {
		receivedIsRemote = isRemote
		receivedPeerID = peerID
		receivedPeerName = peerName
		receivedPath = path
	}

	ws.notifyRemoteModeChanged(true)

	if !receivedIsRemote {
		t.Fatal("expected isRemote=true")
	}
	if receivedPeerID != "peer-123" {
		t.Errorf("expected peerID=%q, got %q", "peer-123", receivedPeerID)
	}
	if receivedPeerName != "John" {
		t.Errorf("expected peerName=%q, got %q", "John", receivedPeerName)
	}
	if receivedPath != nil {
		t.Errorf("expected path=nil, got %v", receivedPath)
	}
}

func TestWorkspace_NotifyRemoteModeChanged_RemoteWithPath(t *testing.T) {
	path := []*models.Item{
		{Title: "Documents", ElementUUID: "uuid-docs"},
	}

	ws := &Workspace{
		remoteProfilePeerID: "peer-456",
		remoteProfileName:   "Jane",
		remoteFolderPath:    path,
	}

	var receivedPath []*models.Item

	ws.onRemoteModeChanged = func(isRemote bool, peerID, peerName string, path []*models.Item) {
		receivedPath = path
	}

	ws.notifyRemoteModeChanged(true)

	if len(receivedPath) != 1 {
		t.Fatalf("expected path length=1, got %d", len(receivedPath))
	}
	if receivedPath[0].Title != "Documents" {
		t.Errorf("expected path[0].Title=%q, got %q", "Documents", receivedPath[0].Title)
	}
}

func TestWorkspace_NotifyRemoteModeChanged_Local(t *testing.T) {
	ws := &Workspace{
		remoteProfilePeerID: "",
		remoteProfileName:   "",
		remoteFolderPath:    nil,
	}

	var receivedIsRemote bool

	ws.onRemoteModeChanged = func(isRemote bool, peerID, peerName string, path []*models.Item) {
		receivedIsRemote = isRemote
	}

	ws.notifyRemoteModeChanged(false)

	if receivedIsRemote {
		t.Fatal("expected isRemote=false")
	}
}

func TestWorkspace_NotifyRemoteModeChanged_NoCallback(t *testing.T) {
	ws := &Workspace{
		onRemoteModeChanged: nil,
	}

	// Should not panic
	ws.notifyRemoteModeChanged(true)
}

func TestWorkspace_ResetToLocalSaved_ClearsRemoteState(t *testing.T) {
	ws := &Workspace{
		remoteProfilePeerID: "peer-123",
		remoteFolderUUID:    "folder-uuid",
		remoteFolderTitle:   "My Folder",
		remoteProfileName:   "John",
		remoteFolderPath:    []*models.Item{{Title: "Test"}},
		navigationManager:   NewNavigationManager(),
	}

	var receivedIsRemote bool
	ws.onRemoteModeChanged = func(isRemote bool, peerID, peerName string, path []*models.Item) {
		receivedIsRemote = isRemote
	}

	ws.ResetToLocalSaved()

	if ws.remoteProfilePeerID != "" {
		t.Errorf("expected remoteProfilePeerID empty, got %q", ws.remoteProfilePeerID)
	}
	if ws.remoteFolderUUID != "" {
		t.Errorf("expected remoteFolderUUID empty, got %q", ws.remoteFolderUUID)
	}
	if ws.remoteFolderTitle != "" {
		t.Errorf("expected remoteFolderTitle empty, got %q", ws.remoteFolderTitle)
	}
	if ws.remoteProfileName != "" {
		t.Errorf("expected remoteProfileName empty, got %q", ws.remoteProfileName)
	}
	if ws.remoteFolderPath != nil {
		t.Errorf("expected remoteFolderPath nil, got %v", ws.remoteFolderPath)
	}
	if receivedIsRemote {
		t.Fatal("expected isRemote=false after reset")
	}
}
