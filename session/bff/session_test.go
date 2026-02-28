package bff

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grokify/coreforge/session/dpop"
)

func TestNewSession(t *testing.T) {
	userID := uuid.New()
	accessToken := "access-token"
	refreshToken := "refresh-token"
	accessExpiry := 15 * time.Minute
	refreshExpiry := 7 * 24 * time.Hour

	session, err := NewSession(userID, accessToken, refreshToken, accessExpiry, refreshExpiry)
	if err != nil {
		t.Fatalf("NewSession() error: %v", err)
	}

	if session.ID == "" {
		t.Error("ID should not be empty")
	}
	if session.UserID != userID {
		t.Errorf("UserID = %s, want %s", session.UserID, userID)
	}
	if session.AccessToken != accessToken {
		t.Errorf("AccessToken = %s, want %s", session.AccessToken, accessToken)
	}
	if session.RefreshToken != refreshToken {
		t.Errorf("RefreshToken = %s, want %s", session.RefreshToken, refreshToken)
	}
	if session.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if session.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
	if session.LastAccessedAt.IsZero() {
		t.Error("LastAccessedAt should not be zero")
	}
}

func TestSession_IsExpired(t *testing.T) {
	userID := uuid.New()

	// Not expired
	session, _ := NewSession(userID, "token", "refresh", time.Hour, time.Hour)
	if session.IsExpired() {
		t.Error("IsExpired() = true for fresh session, want false")
	}

	// Expired
	session.ExpiresAt = time.Now().Add(-time.Hour)
	if !session.IsExpired() {
		t.Error("IsExpired() = false for expired session, want true")
	}
}

func TestSession_IsAccessTokenExpired(t *testing.T) {
	userID := uuid.New()

	// Not expired
	session, _ := NewSession(userID, "token", "refresh", time.Hour, time.Hour)
	if session.IsAccessTokenExpired() {
		t.Error("IsAccessTokenExpired() = true for fresh token, want false")
	}

	// Expired
	session.AccessTokenExpiresAt = time.Now().Add(-time.Minute)
	if !session.IsAccessTokenExpired() {
		t.Error("IsAccessTokenExpired() = false for expired token, want true")
	}
}

func TestSession_IsRefreshTokenExpired(t *testing.T) {
	userID := uuid.New()

	// Not expired
	session, _ := NewSession(userID, "token", "refresh", time.Hour, time.Hour)
	if session.IsRefreshTokenExpired() {
		t.Error("IsRefreshTokenExpired() = true for fresh token, want false")
	}

	// Expired
	session.RefreshTokenExpiresAt = time.Now().Add(-time.Minute)
	if !session.IsRefreshTokenExpired() {
		t.Error("IsRefreshTokenExpired() = false for expired token, want true")
	}
}

func TestSession_NeedsRefresh(t *testing.T) {
	userID := uuid.New()
	session, _ := NewSession(userID, "token", "refresh", 15*time.Minute, time.Hour)

	// Doesn't need refresh (expires in 15 min, threshold is 5 min)
	if session.NeedsRefresh(5 * time.Minute) {
		t.Error("NeedsRefresh(5m) = true for 15m expiry, want false")
	}

	// Needs refresh (expires in 15 min, threshold is 20 min)
	if !session.NeedsRefresh(20 * time.Minute) {
		t.Error("NeedsRefresh(20m) = false for 15m expiry, want true")
	}

	// Already expired
	session.AccessTokenExpiresAt = time.Now().Add(-time.Minute)
	if !session.NeedsRefresh(0) {
		t.Error("NeedsRefresh(0) = false for expired token, want true")
	}
}

func TestSession_HasDPoP(t *testing.T) {
	userID := uuid.New()
	session, _ := NewSession(userID, "token", "refresh", time.Hour, time.Hour)

	// No DPoP
	if session.HasDPoP() {
		t.Error("HasDPoP() = true for session without DPoP, want false")
	}

	// With DPoP
	session.DPoPKeyPairJSON = []byte(`{"d":"key"}`)
	session.DPoPThumbprint = "thumbprint"
	if !session.HasDPoP() {
		t.Error("HasDPoP() = false for session with DPoP, want true")
	}

	// Missing thumbprint
	session.DPoPThumbprint = ""
	if session.HasDPoP() {
		t.Error("HasDPoP() = true for session with missing thumbprint, want false")
	}

	// Missing key pair
	session.DPoPThumbprint = "thumbprint"
	session.DPoPKeyPairJSON = nil
	if session.HasDPoP() {
		t.Error("HasDPoP() = true for session with missing key pair, want false")
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1, err := GenerateSessionID()
	if err != nil {
		t.Fatalf("GenerateSessionID() error: %v", err)
	}

	if id1 == "" {
		t.Error("GenerateSessionID() returned empty string")
	}

	// Should be ~43 characters (32 bytes base64url encoded)
	if len(id1) < 40 || len(id1) > 50 {
		t.Errorf("GenerateSessionID() length = %d, expected ~43", len(id1))
	}

	// Should be unique
	id2, _ := GenerateSessionID()
	if id1 == id2 {
		t.Error("GenerateSessionID() should generate unique IDs")
	}
}

func TestNewSession_UniqueIDs(t *testing.T) {
	userID := uuid.New()

	session1, _ := NewSession(userID, "token", "refresh", time.Hour, time.Hour)
	session2, _ := NewSession(userID, "token", "refresh", time.Hour, time.Hour)

	if session1.ID == session2.ID {
		t.Error("NewSession() should generate unique session IDs")
	}
}

func TestDefaultStoreConfig(t *testing.T) {
	cfg := DefaultStoreConfig()

	if cfg.CleanupInterval != 300 {
		t.Errorf("CleanupInterval = %d, want 300", cfg.CleanupInterval)
	}
	if cfg.MaxSessions != 0 {
		t.Errorf("MaxSessions = %d, want 0 (unlimited)", cfg.MaxSessions)
	}
}

func TestSession_GetDPoPKeyPair_NoDPoP(t *testing.T) {
	userID := uuid.New()
	session, _ := NewSession(userID, "token", "refresh", time.Hour, time.Hour)

	_, err := session.GetDPoPKeyPair()
	if err != ErrNoDPoPKeyPair {
		t.Errorf("GetDPoPKeyPair() error = %v, want ErrNoDPoPKeyPair", err)
	}
}

func TestSession_SetDPoPKeyPair(t *testing.T) {
	userID := uuid.New()
	session, _ := NewSession(userID, "token", "refresh", time.Hour, time.Hour)

	// Generate a key pair
	kp, err := dpop.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	// Set the key pair
	err = session.SetDPoPKeyPair(kp)
	if err != nil {
		t.Fatalf("SetDPoPKeyPair() error: %v", err)
	}

	// Verify it was stored
	if !session.HasDPoP() {
		t.Error("HasDPoP() = false after SetDPoPKeyPair, want true")
	}
	if session.DPoPThumbprint != kp.Thumbprint {
		t.Errorf("DPoPThumbprint = %s, want %s", session.DPoPThumbprint, kp.Thumbprint)
	}
}

func TestSession_GetDPoPKeyPair(t *testing.T) {
	userID := uuid.New()
	session, _ := NewSession(userID, "token", "refresh", time.Hour, time.Hour)

	// Generate and set a key pair
	kp, _ := dpop.GenerateKeyPair()
	err := session.SetDPoPKeyPair(kp)
	if err != nil {
		t.Fatalf("SetDPoPKeyPair() error: %v", err)
	}

	// Get it back
	retrieved, err := session.GetDPoPKeyPair()
	if err != nil {
		t.Fatalf("GetDPoPKeyPair() error: %v", err)
	}

	// Verify it matches
	if retrieved.Thumbprint != kp.Thumbprint {
		t.Errorf("Thumbprint = %s, want %s", retrieved.Thumbprint, kp.Thumbprint)
	}
}

func TestSession_SetDPoPKeyPair_Nil(t *testing.T) {
	userID := uuid.New()
	session, _ := NewSession(userID, "token", "refresh", time.Hour, time.Hour)

	// First set a key pair
	kp, _ := dpop.GenerateKeyPair()
	_ = session.SetDPoPKeyPair(kp)

	// Clear it with nil
	err := session.SetDPoPKeyPair(nil)
	if err != nil {
		t.Fatalf("SetDPoPKeyPair(nil) error: %v", err)
	}

	// Verify it was cleared
	if session.HasDPoP() {
		t.Error("HasDPoP() = true after SetDPoPKeyPair(nil), want false")
	}
	if session.DPoPThumbprint != "" {
		t.Errorf("DPoPThumbprint = %s, want empty", session.DPoPThumbprint)
	}
	if session.DPoPKeyPairJSON != nil {
		t.Error("DPoPKeyPairJSON should be nil")
	}
}

func TestSession_DPoPKeyPair_RoundTrip(t *testing.T) {
	userID := uuid.New()
	session, _ := NewSession(userID, "token", "refresh", time.Hour, time.Hour)

	// Generate key pair
	original, _ := dpop.GenerateKeyPair()

	// Set it
	_ = session.SetDPoPKeyPair(original)

	// Get it back
	retrieved, _ := session.GetDPoPKeyPair()

	// Use both to create proofs and verify they're equivalent
	proof1, err := dpop.CreateProof(original, "GET", "https://example.com/api")
	if err != nil {
		t.Fatalf("CreateProof(original) error: %v", err)
	}

	proof2, err := dpop.CreateProof(retrieved, "GET", "https://example.com/api")
	if err != nil {
		t.Fatalf("CreateProof(retrieved) error: %v", err)
	}

	// Both should be valid proofs (different because of jti/iat)
	if proof1 == "" || proof2 == "" {
		t.Error("Proofs should not be empty")
	}

	// But thumbprints should match
	if original.Thumbprint != retrieved.Thumbprint {
		t.Error("Thumbprints should match after round-trip")
	}
}
