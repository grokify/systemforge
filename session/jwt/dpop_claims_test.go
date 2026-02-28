package jwt

import (
	"testing"

	"github.com/google/uuid"
)

func TestClaims_WithDPoPBinding(t *testing.T) {
	cfg := DefaultConfig()
	claims := NewAccessClaims(cfg, uuid.New(), "test@example.com", "Test User")

	thumbprint := "example-thumbprint-abc123"
	claims.WithDPoPBinding(thumbprint)

	if claims.Confirmation == nil {
		t.Fatal("Confirmation should not be nil after WithDPoPBinding")
	}
	if claims.Confirmation.JKT != thumbprint {
		t.Errorf("Confirmation.JKT = %s, want %s", claims.Confirmation.JKT, thumbprint)
	}
}

func TestClaims_IsDPoPBound(t *testing.T) {
	cfg := DefaultConfig()

	// Not bound
	claims := NewAccessClaims(cfg, uuid.New(), "test@example.com", "Test User")
	if claims.IsDPoPBound() {
		t.Error("IsDPoPBound() = true for unbound token, want false")
	}

	// Bound
	claims.WithDPoPBinding("thumbprint")
	if !claims.IsDPoPBound() {
		t.Error("IsDPoPBound() = false for bound token, want true")
	}
}

func TestClaims_IsDPoPBound_EmptyThumbprint(t *testing.T) {
	cfg := DefaultConfig()
	claims := NewAccessClaims(cfg, uuid.New(), "test@example.com", "Test User")

	// Set Confirmation with empty JKT
	claims.Confirmation = &CNFClaim{JKT: ""}

	if claims.IsDPoPBound() {
		t.Error("IsDPoPBound() = true for empty thumbprint, want false")
	}
}

func TestClaims_DPoPThumbprint(t *testing.T) {
	cfg := DefaultConfig()
	claims := NewAccessClaims(cfg, uuid.New(), "test@example.com", "Test User")

	// Not bound
	if thumbprint := claims.DPoPThumbprint(); thumbprint != "" {
		t.Errorf("DPoPThumbprint() = %s for unbound token, want empty", thumbprint)
	}

	// Bound
	expectedThumbprint := "my-thumbprint"
	claims.WithDPoPBinding(expectedThumbprint)

	if thumbprint := claims.DPoPThumbprint(); thumbprint != expectedThumbprint {
		t.Errorf("DPoPThumbprint() = %s, want %s", thumbprint, expectedThumbprint)
	}
}

func TestCNFClaim(t *testing.T) {
	cnf := &CNFClaim{
		JKT: "test-thumbprint",
	}

	if cnf.JKT != "test-thumbprint" {
		t.Errorf("JKT = %s, want test-thumbprint", cnf.JKT)
	}
}

func TestComputeTokenHash(t *testing.T) {
	token := "my-access-token"
	hash := ComputeTokenHash(token)

	// Verify hash is non-empty
	if hash == "" {
		t.Error("ComputeTokenHash() returned empty string")
	}

	// Verify determinism
	hash2 := ComputeTokenHash(token)
	if hash != hash2 {
		t.Errorf("ComputeTokenHash() not deterministic: %s != %s", hash, hash2)
	}

	// Verify different tokens produce different hashes
	hash3 := ComputeTokenHash("different-token")
	if hash == hash3 {
		t.Error("Different tokens should produce different hashes")
	}
}

func TestClaims_DPoPBinding_ChainedWithOtherMethods(t *testing.T) {
	cfg := DefaultConfig()
	claims := NewAccessClaims(cfg, uuid.New(), "test@example.com", "Test User").
		WithOrganization(uuid.New(), "org-slug", "admin", []string{"read", "write"}).
		WithPlatformAdmin(true).
		WithDPoPBinding("thumbprint-123")

	// Verify all settings are applied
	if claims.OrganizationSlug != "org-slug" {
		t.Errorf("OrganizationSlug = %s, want org-slug", claims.OrganizationSlug)
	}
	if !claims.IsPlatformAdmin {
		t.Error("IsPlatformAdmin = false, want true")
	}
	if !claims.IsDPoPBound() {
		t.Error("IsDPoPBound() = false, want true")
	}
	if claims.DPoPThumbprint() != "thumbprint-123" {
		t.Errorf("DPoPThumbprint() = %s, want thumbprint-123", claims.DPoPThumbprint())
	}
}
