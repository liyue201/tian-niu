package leveldb_storage

import (
	"context"
	"testing"
)

func TestLevelDBStorage_StoreAndLoad(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewLevelDBStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()

	key := "test_key"
	value := "test_value"

	err = storage.Store(ctx, key, value)
	if err != nil {
		t.Errorf("Store failed: %v", err)
	}

	loadedValue, err := storage.Load(ctx, key)
	if err != nil {
		t.Errorf("Load failed: %v", err)
	}

	if loadedValue != value {
		t.Errorf("expected %q, got %q", value, loadedValue)
	}
}

func TestLevelDBStorage_LoadNonExistentKey(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewLevelDBStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()

	loadedValue, err := storage.Load(ctx, "non_existent_key")
	if err != nil {
		t.Errorf("Load should not return error for non-existent key: %v", err)
	}

	if loadedValue != "" {
		t.Errorf("expected empty string for non-existent key, got %q", loadedValue)
	}
}

func TestLevelDBStorage_OverwriteValue(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewLevelDBStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()

	key := "test_key"
	firstValue := "first_value"
	secondValue := "second_value"

	err = storage.Store(ctx, key, firstValue)
	if err != nil {
		t.Errorf("First Store failed: %v", err)
	}

	err = storage.Store(ctx, key, secondValue)
	if err != nil {
		t.Errorf("Second Store failed: %v", err)
	}

	loadedValue, err := storage.Load(ctx, key)
	if err != nil {
		t.Errorf("Load failed: %v", err)
	}

	if loadedValue != secondValue {
		t.Errorf("expected %q after overwrite, got %q", secondValue, loadedValue)
	}
}
