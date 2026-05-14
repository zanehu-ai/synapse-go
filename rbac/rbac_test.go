package rbac

import (
	"errors"
	"testing"
)

func TestPermissionMatchesExactAndWildcard(t *testing.T) {
	cases := []struct {
		granted   string
		requested string
		want      bool
	}{
		{"cargo.parcel.read", "cargo.parcel.read", true},
		{"cargo.parcel.*", "cargo.parcel.write", true},
		{"cargo.*", "cargo.parcel.read", true},
		{"*", "tenant.create", true},
		{"cargo.wallet.read", "cargo.parcel.read", false},
		{"cargo.*", "game.room.read", false},
	}
	for _, tc := range cases {
		if got := PermissionMatches(tc.granted, tc.requested); got != tc.want {
			t.Fatalf("PermissionMatches(%q, %q) = %v, want %v", tc.granted, tc.requested, got, tc.want)
		}
	}
}

func TestCheckPermissionAllowsSliceSubject(t *testing.T) {
	err := CheckPermission([]string{"tenant.*"}, "tenant", "create")
	if err != nil {
		t.Fatalf("CheckPermission returned error: %v", err)
	}
}

func TestCheckPermissionDenied(t *testing.T) {
	err := CheckPermission([]string{"cargo.read"}, "tenant", "create")
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}
