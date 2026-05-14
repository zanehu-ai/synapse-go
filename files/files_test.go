package files

import (
	"errors"
	"testing"
	"time"
)

func TestUploadPolicyPlanBuildsTenantScopedObject(t *testing.T) {
	policy := UploadPolicy{
		MaxSizeBytes:       1024,
		AllowedTypes:       []string{"image/png", "image/jpeg"},
		AllowedCategories:  []string{"avatar"},
		DefaultVisibility:  VisibilityPrivate,
		ObjectKeyNamespace: "platform",
	}
	now := time.Date(2026, 4, 23, 1, 2, 3, 0, time.UTC)
	obj, err := policy.Plan(UploadRequest{
		TenantID:    "tenant-1",
		Category:    "Avatar",
		ContentType: "image/png; charset=binary",
		SizeBytes:   512,
		ObjectID:    "obj-1",
		Now:         now,
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if obj.ObjectKey != "platform/tenants/tenant-1/avatar/2026/04/23/obj-1.png" {
		t.Fatalf("ObjectKey = %q", obj.ObjectKey)
	}
	if obj.Visibility != VisibilityPrivate || obj.ContentType != "image/png" {
		t.Fatalf("object metadata = %+v", obj)
	}
}

func TestUploadPolicyRejectsUnsupportedTypeAndOversize(t *testing.T) {
	policy := UploadPolicy{MaxSizeBytes: 10, AllowedTypes: []string{"image/png"}}
	req := UploadRequest{TenantID: "t-1", Category: "avatar", ContentType: "image/gif", SizeBytes: 1}
	if err := policy.Validate(req); !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("Validate error = %v, want ErrUnsupportedType", err)
	}

	req.ContentType = "image/png"
	req.SizeBytes = 11
	if err := policy.Validate(req); !errors.Is(err, ErrInvalidUpload) {
		t.Fatalf("Validate error = %v, want ErrInvalidUpload", err)
	}
}

func TestUploadPolicyDefaultsVisibilityToPrivate(t *testing.T) {
	policy := UploadPolicy{AllowedTypes: []string{"image/png"}}
	obj, err := policy.Plan(UploadRequest{
		TenantID:    "t-1",
		Category:    "avatar",
		ContentType: "image/png",
		SizeBytes:   1,
		ObjectID:    "obj-1",
		Now:         time.Date(2026, 4, 23, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if obj.Visibility != VisibilityPrivate {
		t.Fatalf("Visibility = %q, want private", obj.Visibility)
	}
}

func TestBuildObjectKeyRejectsTraversal(t *testing.T) {
	_, err := BuildObjectKey(ObjectKeyInput{
		Namespace:   "../bad",
		TenantID:    "t-1",
		Category:    "avatar",
		ContentType: "image/png",
		ObjectID:    "obj-1",
		Now:         time.Date(2026, 4, 23, 0, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, ErrInvalidObjectKey) {
		t.Fatalf("BuildObjectKey error = %v, want ErrInvalidObjectKey", err)
	}
}

func TestNormalizeCategory(t *testing.T) {
	got, err := NormalizeCategory(" Avatar_1 ")
	if err != nil {
		t.Fatalf("NormalizeCategory returned error: %v", err)
	}
	if got != "avatar_1" {
		t.Fatalf("NormalizeCategory = %q", got)
	}
	if _, err := NormalizeCategory("../avatar"); !errors.Is(err, ErrInvalidUpload) {
		t.Fatalf("NormalizeCategory error = %v, want ErrInvalidUpload", err)
	}
}
