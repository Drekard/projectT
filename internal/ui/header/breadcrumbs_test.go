package header

import (
	"testing"

	"projectT/internal/storage/database/models"
)

func TestBreadcrumbManager_UpdateRemoteBreadcrumbs_Basic(t *testing.T) {
	bm := &BreadcrumbManager{
		container: nil, // We don't need actual container for this test
		items:     make([]*BreadcrumbItem, 0),
	}

	bm.UpdateRemoteBreadcrumbs("John Doe", "peer-123", nil)

	if !bm.isRemoteMode {
		t.Fatal("expected isRemoteMode=true")
	}
	if bm.remotePeerName != "John Doe" {
		t.Errorf("expected remotePeerName=%q, got %q", "John Doe", bm.remotePeerName)
	}
	if bm.remotePeerID != "peer-123" {
		t.Errorf("expected remotePeerID=%q, got %q", "peer-123", bm.remotePeerID)
	}
}

func TestBreadcrumbManager_UpdateRemoteBreadcrumbs_WithPath(t *testing.T) {
	bm := &BreadcrumbManager{
		container: nil,
		items:     make([]*BreadcrumbItem, 0),
	}

	path := []*models.Item{
		{Title: "Documents", ElementUUID: "uuid-docs"},
		{Title: "Reports", ElementUUID: "uuid-reports"},
	}

	bm.UpdateRemoteBreadcrumbs("Jane", "peer-456", path)

	if len(bm.items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(bm.items))
	}

	// First item should be peer name
	if bm.items[0].item.Title != "Jane" {
		t.Errorf("expected first item title=%q, got %q", "Jane", bm.items[0].item.Title)
	}

	// Second item should be "Documents"
	if bm.items[1].item.Title != "Documents" {
		t.Errorf("expected second item title=%q, got %q", "Documents", bm.items[1].item.Title)
	}

	// Third item should be "Reports"
	if bm.items[2].item.Title != "Reports" {
		t.Errorf("expected third item title=%q, got %q", "Reports", bm.items[2].item.Title)
	}
}

func TestBreadcrumbManager_ResetToLocalMode(t *testing.T) {
	bm := &BreadcrumbManager{
		isRemoteMode:   true,
		remotePeerID:   "peer-123",
		remotePeerName: "John",
	}

	bm.ResetToLocalMode()

	if bm.isRemoteMode {
		t.Fatal("expected isRemoteMode=false after reset")
	}
	if bm.remotePeerID != "" {
		t.Errorf("expected remotePeerID empty, got %q", bm.remotePeerID)
	}
	if bm.remotePeerName != "" {
		t.Errorf("expected remotePeerName empty, got %q", bm.remotePeerName)
	}
}

func TestBreadcrumbManager_Clear(t *testing.T) {
	bm := &BreadcrumbManager{
		container: nil,
		items: []*BreadcrumbItem{
			{item: &models.Item{Title: "test"}},
		},
	}

	bm.Clear()

	if len(bm.items) != 0 {
		t.Fatalf("expected 0 items after clear, got %d", len(bm.items))
	}
}

func TestBreadcrumbManager_UpdateBreadcrumbs_Local(t *testing.T) {
	bm := &BreadcrumbManager{
		container: nil,
		items:     make([]*BreadcrumbItem, 0),
	}

	path := []*models.Item{
		{Title: "Folder1", ID: 1},
		{Title: "Folder2", ID: 2},
	}

	bm.UpdateBreadcrumbs(path)

	// Should have "Saved" + 2 folders = 3 items
	if len(bm.items) != 3 {
		t.Fatalf("expected 3 items (Saved + 2 folders), got %d", len(bm.items))
	}

	// First item should be "Saved"
	if bm.items[0].item.Title != "Saved" {
		t.Errorf("expected first item title=%q, got %q", "Saved", bm.items[0].item.Title)
	}

	// Second item should be "Folder1"
	if bm.items[1].item.Title != "Folder1" {
		t.Errorf("expected second item title=%q, got %q", "Folder1", bm.items[1].item.Title)
	}
}
