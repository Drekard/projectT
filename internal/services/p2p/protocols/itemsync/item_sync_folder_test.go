package itemsync

import (
	"testing"

	"projectT/internal/storage/database/models"
)

func TestRequestFolder_FilterRootElements(t *testing.T) {
	// Simulate the filtering logic from RequestFolder for root elements
	parentUUID := ""

	emptyStr := ""
	someParent := "some-parent"

	items := []*models.Item{
		{ElementUUID: "item-1", ParentUUID: nil},
		{ElementUUID: "item-2", ParentUUID: &emptyStr},
		{ElementUUID: "item-3", ParentUUID: &someParent},
		{ElementUUID: "item-4", ParentUUID: nil},
	}

	var filtered []*models.Item
	for _, item := range items {
		if parentUUID == "" {
			// Root elements: parent_uuid = nil or empty string
			if item.ParentUUID == nil || *item.ParentUUID == "" {
				filtered = append(filtered, item)
			}
		} else {
			if item.ParentUUID != nil && *item.ParentUUID == parentUUID {
				filtered = append(filtered, item)
			}
		}
	}

	if len(filtered) != 3 {
		t.Fatalf("expected 3 root elements, got %d", len(filtered))
	}

	// Verify item-3 (with parent) is NOT included
	for _, item := range filtered {
		if item.ElementUUID == "item-3" {
			t.Fatal("item-3 should not be in root elements")
		}
	}
}

func TestRequestFolder_FilterSpecificFolder(t *testing.T) {
	parentUUID := "folder-uuid-123"

	parent := "folder-uuid-123"
	otherParent := "other-folder"

	items := []*models.Item{
		{ElementUUID: "item-1", ParentUUID: nil},
		{ElementUUID: "item-2", ParentUUID: &parent},
		{ElementUUID: "item-3", ParentUUID: &parent},
		{ElementUUID: "item-4", ParentUUID: &otherParent},
	}

	var filtered []*models.Item
	for _, item := range items {
		if parentUUID == "" {
			if item.ParentUUID == nil || *item.ParentUUID == "" {
				filtered = append(filtered, item)
			}
		} else {
			if item.ParentUUID != nil && *item.ParentUUID == parentUUID {
				filtered = append(filtered, item)
			}
		}
	}

	if len(filtered) != 2 {
		t.Fatalf("expected 2 elements in folder, got %d", len(filtered))
	}

	// Verify only items with matching parent_uuid are included
	expectedUUIDs := map[string]bool{"item-2": true, "item-3": true}
	for _, item := range filtered {
		if !expectedUUIDs[item.ElementUUID] {
			t.Errorf("unexpected element in folder: %s", item.ElementUUID)
		}
	}
}

func TestRequestFolder_EmptyItemsList(t *testing.T) {
	parentUUID := ""
	items := []*models.Item{}

	var filtered []*models.Item
	for _, item := range items {
		if parentUUID == "" {
			if item.ParentUUID == nil || *item.ParentUUID == "" {
				filtered = append(filtered, item)
			}
		}
	}

	if len(filtered) != 0 {
		t.Fatalf("expected 0 elements, got %d", len(filtered))
	}
}

func TestRequestFolder_AllNilParentUUIDs(t *testing.T) {
	parentUUID := ""

	items := []*models.Item{
		{ElementUUID: "item-1", ParentUUID: nil},
		{ElementUUID: "item-2", ParentUUID: nil},
		{ElementUUID: "item-3", ParentUUID: nil},
	}

	var filtered []*models.Item
	for _, item := range items {
		if parentUUID == "" {
			if item.ParentUUID == nil || *item.ParentUUID == "" {
				filtered = append(filtered, item)
			}
		}
	}

	if len(filtered) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(filtered))
	}
}

func TestRequestFolder_MixedNilAndEmptyString(t *testing.T) {
	parentUUID := ""

	emptyStr := ""
	whitespace := " "

	items := []*models.Item{
		{ElementUUID: "item-1", ParentUUID: nil},
		{ElementUUID: "item-2", ParentUUID: &emptyStr},
		{ElementUUID: "item-3", ParentUUID: &whitespace}, // whitespace is NOT empty
	}

	var filtered []*models.Item
	for _, item := range items {
		if parentUUID == "" {
			if item.ParentUUID == nil || *item.ParentUUID == "" {
				filtered = append(filtered, item)
			}
		}
	}

	// item-1 (nil) and item-2 (empty string) should be included
	// item-3 (whitespace) should NOT be included
	if len(filtered) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(filtered))
	}

	for _, item := range filtered {
		if item.ElementUUID == "item-3" {
			t.Fatal("item-3 with whitespace parent_uuid should not be included")
		}
	}
}
