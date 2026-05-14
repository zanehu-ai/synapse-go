package files

import (
	"errors"
	"mime"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const DefaultMaxUploadSizeBytes int64 = 10 << 20

var (
	ErrInvalidUploadPolicy = errors.New("files: invalid upload policy")
	ErrInvalidUpload       = errors.New("files: invalid upload")
	ErrUnsupportedType     = errors.New("files: unsupported content type")
	ErrInvalidObjectKey    = errors.New("files: invalid object key")
)

type Visibility string

const (
	VisibilityPrivate Visibility = "private"
	VisibilityPublic  Visibility = "public"
)

type Object struct {
	ID          string
	TenantID    string
	ObjectKey   string
	ContentType string
	SizeBytes   int64
	Visibility  Visibility
	CreatedAt   time.Time
}

type UploadPolicy struct {
	MaxSizeBytes       int64
	AllowedTypes       []string
	AllowedCategories  []string
	DefaultVisibility  Visibility
	ObjectKeyNamespace string
}

type UploadRequest struct {
	TenantID    string
	Category    string
	Filename    string
	ContentType string
	SizeBytes   int64
	ObjectID    string
	Visibility  Visibility
	Now         time.Time
}

var categoryPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

func (p UploadPolicy) Validate(req UploadRequest) error {
	if strings.TrimSpace(req.TenantID) == "" || req.SizeBytes < 0 {
		return ErrInvalidUpload
	}
	if _, err := NormalizeCategory(req.Category); err != nil {
		return err
	}
	maxSize := p.MaxSizeBytes
	if maxSize == 0 {
		maxSize = DefaultMaxUploadSizeBytes
	}
	if maxSize < 0 || len(p.AllowedTypes) == 0 {
		return ErrInvalidUploadPolicy
	}
	if req.SizeBytes > maxSize {
		return ErrInvalidUpload
	}
	contentType := NormalizeContentType(req.ContentType)
	if !containsNormalized(p.AllowedTypes, contentType) {
		return ErrUnsupportedType
	}
	if len(p.AllowedCategories) > 0 && !containsNormalized(p.AllowedCategories, NormalizeCategoryOrEmpty(req.Category)) {
		return ErrInvalidUpload
	}
	visibility := req.Visibility
	if visibility == "" {
		visibility = p.DefaultVisibility
	}
	if visibility == "" {
		visibility = VisibilityPrivate
	}
	return ValidateVisibility(visibility)
}

func (p UploadPolicy) Plan(req UploadRequest) (Object, error) {
	if err := p.Validate(req); err != nil {
		return Object{}, err
	}
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	objectID := strings.TrimSpace(req.ObjectID)
	if objectID == "" {
		objectID = uuid.NewString()
	}
	key, err := BuildObjectKey(ObjectKeyInput{
		Namespace:   p.ObjectKeyNamespace,
		TenantID:    req.TenantID,
		Category:    req.Category,
		ContentType: req.ContentType,
		ObjectID:    objectID,
		Now:         now,
	})
	if err != nil {
		return Object{}, err
	}
	visibility := req.Visibility
	if visibility == "" {
		visibility = p.DefaultVisibility
	}
	if visibility == "" {
		visibility = VisibilityPrivate
	}
	return Object{
		ID:          objectID,
		TenantID:    strings.TrimSpace(req.TenantID),
		ObjectKey:   key,
		ContentType: NormalizeContentType(req.ContentType),
		SizeBytes:   req.SizeBytes,
		Visibility:  visibility,
		CreatedAt:   now,
	}, nil
}

type ObjectKeyInput struct {
	Namespace   string
	TenantID    string
	Category    string
	ContentType string
	ObjectID    string
	Now         time.Time
}

func BuildObjectKey(in ObjectKeyInput) (string, error) {
	tenantID := strings.TrimSpace(in.TenantID)
	objectID := strings.TrimSpace(in.ObjectID)
	category, err := NormalizeCategory(in.Category)
	if err != nil {
		return "", err
	}
	if tenantID == "" || objectID == "" || strings.Contains(objectID, "/") || strings.Contains(objectID, `\`) {
		return "", ErrInvalidObjectKey
	}
	namespace, err := normalizeNamespace(in.Namespace)
	if err != nil {
		return "", err
	}
	ext, ok := ExtensionForContentType(in.ContentType)
	if !ok {
		return "", ErrUnsupportedType
	}
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return strings.Join([]string{
		namespace,
		"tenants",
		tenantID,
		category,
		now.UTC().Format("2006/01/02"),
		objectID + ext,
	}, "/"), nil
}

func NormalizeCategory(category string) (string, error) {
	category = strings.ToLower(strings.TrimSpace(category))
	if !categoryPattern.MatchString(category) {
		return "", ErrInvalidUpload
	}
	return category, nil
}

func NormalizeCategoryOrEmpty(category string) string {
	normalized, err := NormalizeCategory(category)
	if err != nil {
		return ""
	}
	return normalized
}

func NormalizeContentType(contentType string) string {
	if parsed, _, err := mime.ParseMediaType(strings.TrimSpace(contentType)); err == nil {
		return strings.ToLower(parsed)
	}
	return strings.ToLower(strings.TrimSpace(contentType))
}

func ExtensionForContentType(contentType string) (string, bool) {
	switch NormalizeContentType(contentType) {
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/webp":
		return ".webp", true
	case "application/pdf":
		return ".pdf", true
	case "text/csv":
		return ".csv", true
	case "application/json":
		return ".json", true
	default:
		return "", false
	}
}

func DetectContentType(sample []byte) string {
	return NormalizeContentType(http.DetectContentType(sample))
}

func ValidateVisibility(visibility Visibility) error {
	switch visibility {
	case VisibilityPrivate, VisibilityPublic:
		return nil
	default:
		return ErrInvalidUpload
	}
}

func normalizeNamespace(namespace string) (string, error) {
	namespace = strings.Trim(strings.ToLower(strings.TrimSpace(namespace)), "/")
	if namespace == "" {
		return "files", nil
	}
	clean := filepath.ToSlash(filepath.Clean(namespace))
	if strings.HasPrefix(clean, "../") || clean == ".." || strings.Contains(clean, `\`) {
		return "", ErrInvalidObjectKey
	}
	parts := strings.Split(clean, "/")
	for _, part := range parts {
		if !categoryPattern.MatchString(part) {
			return "", ErrInvalidObjectKey
		}
	}
	return clean, nil
}

func containsNormalized(values []string, want string) bool {
	for _, value := range values {
		if NormalizeContentType(value) == want || strings.ToLower(strings.TrimSpace(value)) == want {
			return true
		}
	}
	return false
}
