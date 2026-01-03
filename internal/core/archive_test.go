package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArchive_StoreAndGet(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "archive_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	archive, err := NewArchive(dbPath)
	if err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}
	defer archive.Close()

	// Store an entity
	key := "tasks/task-123"
	data := []byte(`{"id": "task-123", "status": "completed"}`)
	err = archive.Store(key, data, "test-node")
	if err != nil {
		t.Fatalf("failed to store entity: %v", err)
	}

	// Retrieve it
	entity, err := archive.Get(key)
	if err != nil {
		t.Fatalf("failed to get entity: %v", err)
	}

	if entity.Key != key {
		t.Errorf("expected key %s, got %s", key, entity.Key)
	}
	if string(entity.Data) != string(data) {
		t.Errorf("expected data %s, got %s", data, entity.Data)
	}
	if entity.ArchivedBy != "test-node" {
		t.Errorf("expected archived_by test-node, got %s", entity.ArchivedBy)
	}
}

func TestArchive_List(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "archive_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	archive, err := NewArchive(dbPath)
	if err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}
	defer archive.Close()

	// Store multiple entities
	keys := []string{"a/1", "b/2", "c/3"}
	for _, key := range keys {
		err = archive.Store(key, []byte(`{}`), "test-node")
		if err != nil {
			t.Fatalf("failed to store entity %s: %v", key, err)
		}
	}

	// List them
	listed, err := archive.List()
	if err != nil {
		t.Fatalf("failed to list entities: %v", err)
	}

	if len(listed) != len(keys) {
		t.Errorf("expected %d keys, got %d", len(keys), len(listed))
	}
}

func TestArchive_Delete(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "archive_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	archive, err := NewArchive(dbPath)
	if err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}
	defer archive.Close()

	// Store and delete
	key := "tasks/task-456"
	err = archive.Store(key, []byte(`{}`), "test-node")
	if err != nil {
		t.Fatalf("failed to store entity: %v", err)
	}

	err = archive.Delete(key)
	if err != nil {
		t.Fatalf("failed to delete entity: %v", err)
	}

	// Should not exist anymore
	_, err = archive.Get(key)
	if err == nil {
		t.Error("expected error getting deleted entity")
	}
}

func TestArchive_GetNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "archive_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	archive, err := NewArchive(dbPath)
	if err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}
	defer archive.Close()

	_, err = archive.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent entity")
	}
}

func TestArchive_EmptyPath(t *testing.T) {
	_, err := NewArchive("")
	if err == nil {
		t.Error("expected error for empty path")
	}
}
