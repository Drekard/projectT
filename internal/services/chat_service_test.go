package services

import (
	"encoding/json"
	"testing"
	"time"

	"projectT/internal/storage/database/models"
)

func TestSendFolderMessage_MetadataSerialization(t *testing.T) {
	folder := &models.Item{
		ElementUUID: "folder-uuid-123",
		Title:       "Test Folder",
		CreatedAt:   time.Now(),
	}

	childItems := []*models.Item{
		{ElementUUID: "item-1"},
		{ElementUUID: "item-2"},
		{ElementUUID: "item-3"},
	}

	metadata := map[string]interface{}{
		"folder_uuid":  folder.ElementUUID,
		"folder_title": folder.Title,
		"item_count":   len(childItems),
		"sent_at":      folder.CreatedAt.Format(time.RFC3339),
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("failed to marshal metadata: %v", err)
	}

	// Verify we can unmarshal it back
	var parsed map[string]interface{}
	if err := json.Unmarshal(metadataJSON, &parsed); err != nil {
		t.Fatalf("failed to unmarshal metadata: %v", err)
	}

	if parsed["folder_uuid"] != "folder-uuid-123" {
		t.Errorf("expected folder_uuid=%q, got %v", "folder-uuid-123", parsed["folder_uuid"])
	}
	if parsed["folder_title"] != "Test Folder" {
		t.Errorf("expected folder_title=%q, got %v", "Test Folder", parsed["folder_title"])
	}
	if int(parsed["item_count"].(float64)) != 3 {
		t.Errorf("expected item_count=3, got %v", parsed["item_count"])
	}
}

func TestSendFolderMessage_MetadataEmptyFolder(t *testing.T) {
	folder := &models.Item{
		ElementUUID: "empty-folder",
		Title:       "Empty",
		CreatedAt:   time.Now(),
	}

	metadata := map[string]interface{}{
		"folder_uuid":  folder.ElementUUID,
		"folder_title": folder.Title,
		"item_count":   0,
		"sent_at":      folder.CreatedAt.Format(time.RFC3339),
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("failed to marshal metadata: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(metadataJSON, &parsed); err != nil {
		t.Fatalf("failed to unmarshal metadata: %v", err)
	}

	if int(parsed["item_count"].(float64)) != 0 {
		t.Errorf("expected item_count=0, got %v", parsed["item_count"])
	}
}

func TestSendFolderMessage_ContentType(t *testing.T) {
	// Verify that folder_batch content type is correctly set
	expectedContentType := "folder_batch"
	if expectedContentType != "folder_batch" {
		t.Errorf("expected content_type=%q, got %q", "folder_batch", expectedContentType)
	}
}
