// frozen reference: any change requires ADR/SPEC
// This file is the Go source-of-truth for Synapse RBAC permission matching semantics.
// Java SDK conformance tests in templates/game/.../RbacWildcardConformanceTest.java
// load testdata/permission_vectors.json generated from this file and assert byte-equality.
package rbac

import (
	"errors"
	"strings"
)

var ErrPermissionDenied = errors.New("rbac: permission denied")

// ErrNotImplemented is kept for compatibility with old skeleton callers.
// New code should check ErrPermissionDenied.
var ErrNotImplemented = ErrPermissionDenied

// PermissionMatches returns true when a granted permission code allows a
// requested permission code. It supports exact matches, "*" and suffix
// wildcards such as "cargo.*" or "cargo.parcel.*".
func PermissionMatches(granted, requested string) bool {
	granted = strings.TrimSpace(granted)
	requested = strings.TrimSpace(requested)
	if granted == "" || requested == "" {
		return false
	}
	if granted == "*" || granted == requested {
		return true
	}
	if strings.HasSuffix(granted, ".*") {
		prefix := strings.TrimSuffix(granted, ".*")
		return requested == prefix || strings.HasPrefix(requested, prefix+".")
	}
	return false
}

// IsAllowed checks whether any granted permission allows the requested code.
func IsAllowed(granted []string, requested string) bool {
	for _, code := range granted {
		if PermissionMatches(code, requested) {
			return true
		}
	}
	return false
}

// CheckPermission checks a subject permission collection against resource/action.
// Supported subject forms are []string and interface{ PermissionCodes() []string }.
func CheckPermission(subject any, resource string, action string) error {
	requested := strings.Trim(strings.TrimSpace(resource), ".") + "." + strings.Trim(strings.TrimSpace(action), ".")
	if requested == "." {
		return ErrPermissionDenied
	}
	if IsAllowed(permissionCodes(subject), requested) {
		return nil
	}
	return ErrPermissionDenied
}

type permissionCoder interface {
	PermissionCodes() []string
}

func permissionCodes(subject any) []string {
	switch v := subject.(type) {
	case []string:
		return v
	case permissionCoder:
		return v.PermissionCodes()
	default:
		return nil
	}
}
