package integration

import (
	"context"
	"testing"

	"github.com/ajepe/talon/internal/filestore"
)

func TestAttachmentFilestoreRoundTrip(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	store, err := filestore.New(t.TempDir())
	if err != nil {
		t.Fatalf("filestore.New() error = %v", err)
	}
	service := filestore.NewAttachmentService(store, h.db)

	var attachmentID int
	if err := h.db.QueryRowContext(ctx,
		`INSERT INTO ir_attachment (name, filename, mimetype)
		 VALUES ('report', 'report.csv', 'text/csv') RETURNING id`,
	).Scan(&attachmentID); err != nil {
		t.Fatalf("insert attachment error = %v", err)
	}

	content := []byte("Name,Code\nAcme,ACME\n")
	if err := service.Set(ctx, attachmentID, content); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	var fileID string
	var size int
	if err := h.db.QueryRowContext(ctx,
		`SELECT file_id, size FROM ir_attachment WHERE id = $1`, attachmentID).Scan(&fileID, &size); err != nil {
		t.Fatalf("attachment query error = %v", err)
	}
	if fileID == "" {
		t.Fatal("file_id is empty after Set()")
	}
	if size != len(content) {
		t.Errorf("size = %d, want %d", size, len(content))
	}
	if !store.Exists(fileID) {
		t.Error("file missing from filestore")
	}

	data, err := service.Get(ctx, attachmentID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("Get() = %q, want %q", data, content)
	}

	if err := service.Delete(ctx, attachmentID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if store.Exists(fileID) {
		t.Error("file still present after Delete()")
	}
}

func TestAttachmentServiceRequiresStore(t *testing.T) {
	service := filestore.NewAttachmentService(nil, nil)
	if service != nil {
		t.Error("NewAttachmentService(nil) should return nil")
	}
}
