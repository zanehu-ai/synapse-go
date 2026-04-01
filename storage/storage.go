package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Storage is the interface for object storage operations.
type Storage interface {
	// Upload stores data at the given key and returns the public URL.
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) (url string, err error)
	// Delete removes the object at the given key.
	Delete(ctx context.Context, key string) error
	// PresignedURL generates a time-limited URL for direct access to the object.
	PresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// LocalStorage implements Storage using the local filesystem.
// Suitable for development and testing environments.
type LocalStorage struct {
	basePath string // directory to store files
	baseURL  string // URL prefix for accessing files
}

// NewLocal creates a LocalStorage that saves files to basePath
// and returns URLs prefixed with baseURL. basePath is resolved to an absolute
// path at construction time.
func NewLocal(basePath, baseURL string) *LocalStorage {
	abs, err := filepath.Abs(basePath)
	if err != nil {
		abs = basePath
	}
	return &LocalStorage{basePath: abs, baseURL: baseURL}
}

// safePath validates and resolves key to an absolute path within basePath.
func (s *LocalStorage) safePath(key string) (string, error) {
	fullPath := filepath.Join(s.basePath, key)
	if !strings.HasPrefix(fullPath, s.basePath+string(filepath.Separator)) && fullPath != s.basePath {
		return "", errors.New("storage: key escapes base directory")
	}
	return fullPath, nil
}

// Upload saves the data to a local file and returns the URL.
func (s *LocalStorage) Upload(_ context.Context, key string, reader io.Reader, _ string) (string, error) {
	fullPath, err := s.safePath(key)
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("storage mkdir: %w", err)
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("storage create: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, reader); err != nil {
		return "", fmt.Errorf("storage write: %w", err)
	}

	return s.baseURL + "/" + key, nil
}

// Delete removes the file at the given key.
func (s *LocalStorage) Delete(_ context.Context, key string) error {
	fullPath, err := s.safePath(key)
	if err != nil {
		return err
	}
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage delete: %w", err)
	}
	return nil
}

// PresignedURL returns the direct URL for local storage (no expiry enforcement).
func (s *LocalStorage) PresignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	if _, err := s.safePath(key); err != nil {
		return "", err
	}
	return s.baseURL + "/" + key, nil
}
