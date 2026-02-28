package bff

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewMemoryStore(t *testing.T) {
	store := NewMemoryStore(StoreConfig{})
	defer func() { _ = store.Close() }()

	if store == nil {
		t.Fatal("NewMemoryStore returned nil")
	}
	if store.Count() != 0 {
		t.Errorf("Count() = %d, want 0", store.Count())
	}
}

func TestMemoryStore_Create(t *testing.T) {
	store := NewMemoryStore(StoreConfig{})
	defer func() { _ = store.Close() }()

	session, err := NewSession(uuid.New(), "access-token", "refresh-token", 15*time.Minute, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("NewSession() error: %v", err)
	}

	err = store.Create(context.Background(), session)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if store.Count() != 1 {
		t.Errorf("Count() = %d, want 1", store.Count())
	}
}

func TestMemoryStore_Create_NilSession(t *testing.T) {
	store := NewMemoryStore(StoreConfig{})
	defer func() { _ = store.Close() }()

	err := store.Create(context.Background(), nil)
	if !errors.Is(err, ErrInvalidSession) {
		t.Errorf("Create(nil) error = %v, want ErrInvalidSession", err)
	}
}

func TestMemoryStore_Create_EmptyID(t *testing.T) {
	store := NewMemoryStore(StoreConfig{})
	defer func() { _ = store.Close() }()

	session := &Session{}
	err := store.Create(context.Background(), session)
	if !errors.Is(err, ErrInvalidSession) {
		t.Errorf("Create() with empty ID error = %v, want ErrInvalidSession", err)
	}
}

func TestMemoryStore_Create_MaxSessions(t *testing.T) {
	store := NewMemoryStore(StoreConfig{
		MaxSessions: 2,
	})
	defer func() { _ = store.Close() }()

	// Create two sessions
	for i := range 2 {
		session, _ := NewSession(uuid.New(), "token", "refresh", time.Hour, time.Hour)
		if err := store.Create(context.Background(), session); err != nil {
			t.Fatalf("Create() %d error: %v", i, err)
		}
	}

	// Third should fail
	session, _ := NewSession(uuid.New(), "token", "refresh", time.Hour, time.Hour)
	err := store.Create(context.Background(), session)
	if err == nil {
		t.Error("Create() should fail when max sessions exceeded")
	}
}

func TestMemoryStore_Get(t *testing.T) {
	store := NewMemoryStore(StoreConfig{})
	defer func() { _ = store.Close() }()

	userID := uuid.New()
	session, _ := NewSession(userID, "access-token", "refresh-token", 15*time.Minute, 7*24*time.Hour)
	_ = store.Create(context.Background(), session)

	retrieved, err := store.Get(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("ID = %s, want %s", retrieved.ID, session.ID)
	}
	if retrieved.UserID != userID {
		t.Errorf("UserID = %s, want %s", retrieved.UserID, userID)
	}
	if retrieved.AccessToken != "access-token" {
		t.Errorf("AccessToken = %s, want access-token", retrieved.AccessToken)
	}
}

func TestMemoryStore_Get_NotFound(t *testing.T) {
	store := NewMemoryStore(StoreConfig{})
	defer func() { _ = store.Close() }()

	_, err := store.Get(context.Background(), "non-existent-id")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Get() error = %v, want ErrSessionNotFound", err)
	}
}

func TestMemoryStore_Get_Expired(t *testing.T) {
	store := NewMemoryStore(StoreConfig{})
	defer func() { _ = store.Close() }()

	// Create an already-expired session
	session, _ := NewSession(uuid.New(), "token", "refresh", -time.Hour, -time.Hour)
	session.ExpiresAt = time.Now().Add(-time.Hour)
	_ = store.Create(context.Background(), session)

	_, err := store.Get(context.Background(), session.ID)
	if !errors.Is(err, ErrSessionExpired) {
		t.Errorf("Get() error = %v, want ErrSessionExpired", err)
	}

	// Session should be removed
	if store.Count() != 0 {
		t.Errorf("Count() = %d, want 0 (expired session should be removed)", store.Count())
	}
}

func TestMemoryStore_Update(t *testing.T) {
	store := NewMemoryStore(StoreConfig{})
	defer func() { _ = store.Close() }()

	session, _ := NewSession(uuid.New(), "token1", "refresh1", time.Hour, time.Hour)
	_ = store.Create(context.Background(), session)

	// Update the session
	session.AccessToken = "token2"
	err := store.Update(context.Background(), session)
	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	// Verify update
	retrieved, _ := store.Get(context.Background(), session.ID)
	if retrieved.AccessToken != "token2" {
		t.Errorf("AccessToken = %s, want token2", retrieved.AccessToken)
	}
}

func TestMemoryStore_Update_NotFound(t *testing.T) {
	store := NewMemoryStore(StoreConfig{})
	defer func() { _ = store.Close() }()

	session := &Session{ID: "non-existent"}
	err := store.Update(context.Background(), session)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Update() error = %v, want ErrSessionNotFound", err)
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore(StoreConfig{})
	defer func() { _ = store.Close() }()

	session, _ := NewSession(uuid.New(), "token", "refresh", time.Hour, time.Hour)
	_ = store.Create(context.Background(), session)

	err := store.Delete(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	if store.Count() != 0 {
		t.Errorf("Count() = %d, want 0", store.Count())
	}
}

func TestMemoryStore_Delete_NotFound(t *testing.T) {
	store := NewMemoryStore(StoreConfig{})
	defer func() { _ = store.Close() }()

	err := store.Delete(context.Background(), "non-existent")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Delete() error = %v, want ErrSessionNotFound", err)
	}
}

func TestMemoryStore_DeleteByUserID(t *testing.T) {
	store := NewMemoryStore(StoreConfig{})
	defer func() { _ = store.Close() }()

	userID := uuid.New()
	otherUserID := uuid.New()

	// Create 3 sessions for userID
	for range 3 {
		session, _ := NewSession(userID, "token", "refresh", time.Hour, time.Hour)
		_ = store.Create(context.Background(), session)
	}

	// Create 1 session for otherUserID
	session, _ := NewSession(otherUserID, "token", "refresh", time.Hour, time.Hour)
	_ = store.Create(context.Background(), session)

	count, err := store.DeleteByUserID(context.Background(), userID.String())
	if err != nil {
		t.Fatalf("DeleteByUserID() error: %v", err)
	}

	if count != 3 {
		t.Errorf("DeleteByUserID() count = %d, want 3", count)
	}

	if store.Count() != 1 {
		t.Errorf("Count() = %d, want 1", store.Count())
	}
}

func TestMemoryStore_Touch(t *testing.T) {
	store := NewMemoryStore(StoreConfig{})
	defer func() { _ = store.Close() }()

	session, _ := NewSession(uuid.New(), "token", "refresh", time.Hour, time.Hour)
	originalLastAccessed := session.LastAccessedAt
	_ = store.Create(context.Background(), session)

	time.Sleep(10 * time.Millisecond)

	err := store.Touch(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("Touch() error: %v", err)
	}

	retrieved, _ := store.Get(context.Background(), session.ID)
	if !retrieved.LastAccessedAt.After(originalLastAccessed) {
		t.Error("LastAccessedAt should be updated after Touch()")
	}
}

func TestMemoryStore_Touch_NotFound(t *testing.T) {
	store := NewMemoryStore(StoreConfig{})
	defer func() { _ = store.Close() }()

	err := store.Touch(context.Background(), "non-existent")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("Touch() error = %v, want ErrSessionNotFound", err)
	}
}

func TestMemoryStore_Cleanup(t *testing.T) {
	store := NewMemoryStore(StoreConfig{})
	defer func() { _ = store.Close() }()

	// Create active session
	activeSession, _ := NewSession(uuid.New(), "token", "refresh", time.Hour, time.Hour)
	_ = store.Create(context.Background(), activeSession)

	// Create expired session
	expiredSession, _ := NewSession(uuid.New(), "token", "refresh", -time.Hour, -time.Hour)
	expiredSession.ExpiresAt = time.Now().Add(-time.Hour)
	_ = store.Create(context.Background(), expiredSession)

	if store.Count() != 2 {
		t.Fatalf("Count() = %d, want 2", store.Count())
	}

	count, err := store.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("Cleanup() error: %v", err)
	}

	if count != 1 {
		t.Errorf("Cleanup() count = %d, want 1", count)
	}

	if store.Count() != 1 {
		t.Errorf("Count() = %d, want 1", store.Count())
	}
}

func TestMemoryStore_ContextCancellation(t *testing.T) {
	store := NewMemoryStore(StoreConfig{})
	defer func() { _ = store.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	session, _ := NewSession(uuid.New(), "token", "refresh", time.Hour, time.Hour)

	// All operations should respect context cancellation
	if err := store.Create(ctx, session); !errors.Is(err, context.Canceled) {
		t.Errorf("Create() error = %v, want context.Canceled", err)
	}

	if _, err := store.Get(ctx, "id"); !errors.Is(err, context.Canceled) {
		t.Errorf("Get() error = %v, want context.Canceled", err)
	}

	if err := store.Update(ctx, session); !errors.Is(err, context.Canceled) {
		t.Errorf("Update() error = %v, want context.Canceled", err)
	}

	if err := store.Delete(ctx, "id"); !errors.Is(err, context.Canceled) {
		t.Errorf("Delete() error = %v, want context.Canceled", err)
	}

	if _, err := store.DeleteByUserID(ctx, "uid"); !errors.Is(err, context.Canceled) {
		t.Errorf("DeleteByUserID() error = %v, want context.Canceled", err)
	}

	if err := store.Touch(ctx, "id"); !errors.Is(err, context.Canceled) {
		t.Errorf("Touch() error = %v, want context.Canceled", err)
	}

	if _, err := store.Cleanup(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("Cleanup() error = %v, want context.Canceled", err)
	}
}

func TestMemoryStore_AutoCleanup(t *testing.T) {
	store := NewMemoryStore(StoreConfig{
		CleanupInterval: 1, // 1 second
	})
	defer func() { _ = store.Close() }()

	// Create expired session
	expiredSession, _ := NewSession(uuid.New(), "token", "refresh", -time.Hour, -time.Hour)
	expiredSession.ExpiresAt = time.Now().Add(-time.Hour)
	_ = store.Create(context.Background(), expiredSession)

	if store.Count() != 1 {
		t.Fatalf("Count() = %d, want 1", store.Count())
	}

	// Wait for auto cleanup
	time.Sleep(1500 * time.Millisecond)

	if store.Count() != 0 {
		t.Errorf("Count() = %d after auto cleanup, want 0", store.Count())
	}
}

func TestMemoryStore_IsolatedCopies(t *testing.T) {
	store := NewMemoryStore(StoreConfig{})
	defer func() { _ = store.Close() }()

	session, _ := NewSession(uuid.New(), "original-token", "refresh", time.Hour, time.Hour)
	_ = store.Create(context.Background(), session)

	// Modify the original session
	session.AccessToken = "modified-token"

	// Get should return the stored copy, not the modified one
	retrieved, _ := store.Get(context.Background(), session.ID)
	if retrieved.AccessToken != "original-token" {
		t.Error("Store should keep isolated copy of session")
	}

	// Modify the retrieved session
	retrieved.AccessToken = "modified-again"

	// Get again should return original
	retrieved2, _ := store.Get(context.Background(), session.ID)
	if retrieved2.AccessToken != "original-token" {
		t.Error("Get() should return isolated copy")
	}
}
