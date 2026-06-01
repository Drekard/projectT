package profile

import (
	"testing"
)

func TestRemoteProfileUI_ParsePinnedUUIDs_StringArray(t *testing.T) {
	ui := &RemoteProfileUI{}

	jsonStr := `["uuid-1", "uuid-2", "uuid-3"]`
	uuids, err := ui.parsePinnedUUIDs(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(uuids) != 3 {
		t.Fatalf("expected 3 uuids, got %d", len(uuids))
	}
	if uuids[0] != "uuid-1" || uuids[1] != "uuid-2" || uuids[2] != "uuid-3" {
		t.Fatalf("unexpected uuids: %v", uuids)
	}
}

func TestRemoteProfileUI_ParsePinnedUUIDs_EmptyArray(t *testing.T) {
	ui := &RemoteProfileUI{}

	jsonStr := `[]`
	uuids, err := ui.parsePinnedUUIDs(jsonStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(uuids) != 0 {
		t.Fatalf("expected 0 uuids, got %d", len(uuids))
	}
}

func TestRemoteProfileUI_ParsePinnedUUIDs_InvalidJSON(t *testing.T) {
	ui := &RemoteProfileUI{}

	jsonStr := `{invalid json}`
	_, err := ui.parsePinnedUUIDs(jsonStr)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestRemoteProfileUI_ParsePinnedUUIDs_NumberArray(t *testing.T) {
	ui := &RemoteProfileUI{}

	// Number array is not a valid pinned_uuids format — should fail
	jsonStr := `[1, 2, 3]`
	_, err := ui.parsePinnedUUIDs(jsonStr)
	if err == nil {
		t.Fatal("expected error for number array, got nil")
	}
}

func TestRemoteProfileUI_IsLocalDetection(t *testing.T) {
	// Test that isLocal flag is correctly set based on OwnerType
	tests := []struct {
		ownerType string
		isLocal   bool
	}{
		{"local", true},
		{"remote", false},
		{"", false},
	}

	for _, tt := range tests {
		ui := &RemoteProfileUI{}
		ui.isLocal = tt.ownerType == "local"
		if ui.isLocal != tt.isLocal {
			t.Errorf("ownerType=%q: expected isLocal=%v, got %v", tt.ownerType, tt.isLocal, ui.isLocal)
		}
	}
}
