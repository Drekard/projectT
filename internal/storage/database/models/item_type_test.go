package models

import (
	"testing"
)

func TestItemType_Folder(t *testing.T) {
	if ItemTypeFolder != "folder" {
		t.Errorf("expected ItemTypeFolder=%q, got %q", "folder", ItemTypeFolder)
	}
}

func TestItemType_Element(t *testing.T) {
	if ItemTypeElement != "element" {
		t.Errorf("expected ItemTypeElement=%q, got %q", "element", ItemTypeElement)
	}
}

func TestItem_IsRemote(t *testing.T) {
	peerStr := "peer-123"
	tests := []struct {
		name       string
		ownerType  string
		sourcePeer *string
		expected   bool
	}{
		{"remote with peer", "remote", &peerStr, true},
		{"remote without peer", "remote", nil, true},
		{"local with peer", "local", &peerStr, false},
		{"local without peer", "local", nil, false},
		{"empty owner", "", &peerStr, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &Item{
				OwnerType:    OwnerType(tt.ownerType),
				SourcePeerID: tt.sourcePeer,
			}
			if item.IsRemote() != tt.expected {
				t.Errorf("expected IsRemote=%v, got %v", tt.expected, item.IsRemote())
			}
		})
	}
}

func TestItem_IsLocal(t *testing.T) {
	tests := []struct {
		name      string
		ownerType string
		expected  bool
	}{
		{"local", "local", true},
		{"remote", "remote", false},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &Item{OwnerType: OwnerType(tt.ownerType)}
			if item.IsLocal() != tt.expected {
				t.Errorf("expected IsLocal=%v, got %v", tt.expected, item.IsLocal())
			}
		})
	}
}
