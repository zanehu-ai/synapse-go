package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalStorage_Upload(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir, "http://localhost:8080/files")

	url, err := s.Upload(context.Background(), "test/photo.jpg", strings.NewReader("image-data"), "image/jpeg")
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}

	if url != "http://localhost:8080/files/test/photo.jpg" {
		t.Errorf("url = %q", url)
	}

	data, err := os.ReadFile(filepath.Join(dir, "test/photo.jpg"))
	if err != nil {
		t.Fatalf("file not found: %v", err)
	}
	if string(data) != "image-data" {
		t.Errorf("file content = %q", string(data))
	}
}

func TestLocalStorage_Delete(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir, "http://localhost:8080/files")

	if _, err := s.Upload(context.Background(), "to-delete.txt", strings.NewReader("data"), "text/plain"); err != nil {
		t.Fatalf("Upload() error: %v", err)
	}

	err := s.Delete(context.Background(), "to-delete.txt")
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "to-delete.txt")); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}
}

func TestLocalStorage_DeleteNonExistent(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir, "http://localhost:8080/files")

	err := s.Delete(context.Background(), "nonexistent.txt")
	if err != nil {
		t.Errorf("Delete() non-existent should not error: %v", err)
	}
}

func TestLocalStorage_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir, "http://localhost:8080/files")

	keys := []string{
		"../../../etc/passwd",
		"../../escape",
		"../outside",
		"subdir/../../outside",
	}
	for _, key := range keys {
		t.Run("Upload_"+key, func(t *testing.T) {
			_, err := s.Upload(context.Background(), key, strings.NewReader("data"), "text/plain")
			if err == nil {
				t.Error("expected error for path traversal key")
			}
		})
		t.Run("Delete_"+key, func(t *testing.T) {
			err := s.Delete(context.Background(), key)
			if err == nil {
				t.Error("expected error for path traversal key")
			}
		})
		t.Run("PresignedURL_"+key, func(t *testing.T) {
			_, err := s.PresignedURL(context.Background(), key, 0)
			if err == nil {
				t.Error("expected error for path traversal key")
			}
		})
	}
}

func TestLocalStorage_UploadCreatesSubdirs(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir, "http://localhost:8080/files")

	_, err := s.Upload(context.Background(), "a/b/c/deep.txt", strings.NewReader("deep"), "text/plain")
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "a/b/c/deep.txt")); err != nil {
		t.Errorf("deep file not created: %v", err)
	}
}
