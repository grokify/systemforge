package invalidation

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryStore_CreateAndGet(t *testing.T) {
	store := NewMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	session := &Session{
		ID:           "session-123",
		UserID:       "user-456",
		DeviceID:     "device-789",
		IPAddress:    "192.168.1.1",
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
		ExpiresAt:    time.Now().Add(time.Hour),
		Metadata:     map[string]string{"key": "value"},
	}

	// Create
	err := store.Create(ctx, session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get
	retrieved, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("expected ID %s, got %s", session.ID, retrieved.ID)
	}
	if retrieved.UserID != session.UserID {
		t.Errorf("expected UserID %s, got %s", session.UserID, retrieved.UserID)
	}
	if retrieved.Metadata["key"] != "value" {
		t.Error("metadata not preserved")
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	session := &Session{
		ID:        "session-123",
		UserID:    "user-456",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	_ = store.Create(ctx, session)

	// Delete
	err := store.Delete(ctx, session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get should fail
	_, err = store.Get(ctx, session.ID)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestMemoryStore_ListByUser(t *testing.T) {
	store := NewMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	userID := "user-456"

	// Create multiple sessions
	for i := 0; i < 3; i++ {
		session := &Session{
			ID:        "session-" + string(rune('a'+i)),
			UserID:    userID,
			ExpiresAt: time.Now().Add(time.Hour),
		}
		_ = store.Create(ctx, session)
	}

	// List
	sessions, err := store.ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}
}

func TestMemoryStore_DeleteByUser(t *testing.T) {
	store := NewMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	userID := "user-456"

	// Create multiple sessions
	for i := 0; i < 3; i++ {
		session := &Session{
			ID:        "session-" + string(rune('a'+i)),
			UserID:    userID,
			ExpiresAt: time.Now().Add(time.Hour),
		}
		_ = store.Create(ctx, session)
	}

	// Delete all
	count, err := store.DeleteByUser(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 3 {
		t.Errorf("expected 3 deleted, got %d", count)
	}

	// List should be empty
	sessions, _ := store.ListByUser(ctx, userID)
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestMemoryStore_DeleteByDevice(t *testing.T) {
	store := NewMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	userID := "user-456"

	// Create sessions on different devices
	_ = store.Create(ctx, &Session{
		ID:        "session-a",
		UserID:    userID,
		DeviceID:  "device-1",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	_ = store.Create(ctx, &Session{
		ID:        "session-b",
		UserID:    userID,
		DeviceID:  "device-1",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	_ = store.Create(ctx, &Session{
		ID:        "session-c",
		UserID:    userID,
		DeviceID:  "device-2",
		ExpiresAt: time.Now().Add(time.Hour),
	})

	// Delete device-1 sessions
	count, err := store.DeleteByDevice(ctx, userID, "device-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 deleted, got %d", count)
	}

	// Should have 1 session left
	sessions, _ := store.ListByUser(ctx, userID)
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}
}

func TestMemoryStore_DeleteExpired(t *testing.T) {
	store := NewMemoryStore()
	defer func() { _ = store.Close() }()

	ctx := context.Background()

	// Create expired and valid sessions
	_ = store.Create(ctx, &Session{
		ID:        "expired",
		UserID:    "user-1",
		ExpiresAt: time.Now().Add(-time.Hour), // Already expired
	})
	_ = store.Create(ctx, &Session{
		ID:        "valid",
		UserID:    "user-2",
		ExpiresAt: time.Now().Add(time.Hour),
	})

	// Delete expired
	count, err := store.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 deleted, got %d", count)
	}

	// Valid session should still exist
	_, err = store.Get(ctx, "valid")
	if err != nil {
		t.Error("valid session should still exist")
	}

	// Expired session should be gone
	_, err = store.Get(ctx, "expired")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Error("expired session should be deleted")
	}
}

func TestManager_CreateSession(t *testing.T) {
	store := NewMemoryStore()
	manager := NewManager(store)
	defer func() { _ = manager.Close() }()

	ctx := context.Background()

	session, err := manager.CreateSession(ctx, "user-123",
		WithDeviceID("device-456"),
		WithIPAddress("192.168.1.1"),
		WithMetadata("browser", "Chrome"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session.ID == "" {
		t.Error("session ID should not be empty")
	}
	if session.UserID != "user-123" {
		t.Errorf("expected UserID user-123, got %s", session.UserID)
	}
	if session.DeviceID != "device-456" {
		t.Errorf("expected DeviceID device-456, got %s", session.DeviceID)
	}
	if session.Metadata["browser"] != "Chrome" {
		t.Error("metadata not set correctly")
	}
}

func TestManager_ValidateSession(t *testing.T) {
	store := NewMemoryStore()
	manager := NewManager(store)
	defer func() { _ = manager.Close() }()

	ctx := context.Background()

	// Create session
	session, _ := manager.CreateSession(ctx, "user-123")

	// Validate
	validated, err := manager.ValidateSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if validated.ID != session.ID {
		t.Error("session ID mismatch")
	}

	// LastActiveAt should be updated
	if !validated.LastActiveAt.After(session.LastActiveAt) && validated.LastActiveAt != session.LastActiveAt {
		t.Error("LastActiveAt should be updated")
	}
}

func TestManager_ValidateSession_Expired(t *testing.T) {
	store := NewMemoryStore()
	manager := NewManager(store, WithSessionTTL(time.Millisecond))
	defer func() { _ = manager.Close() }()

	ctx := context.Background()

	// Create session
	session, _ := manager.CreateSession(ctx, "user-123")

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Validate should fail
	_, err := manager.ValidateSession(ctx, session.ID)
	if !errors.Is(err, ErrSessionExpired) {
		t.Errorf("expected ErrSessionExpired, got %v", err)
	}
}

func TestManager_InvalidateSession(t *testing.T) {
	store := NewMemoryStore()
	manager := NewManager(store)
	defer func() { _ = manager.Close() }()

	ctx := context.Background()

	// Create session
	session, _ := manager.CreateSession(ctx, "user-123")

	// Invalidate
	err := manager.InvalidateSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Validate should fail
	_, err = manager.ValidateSession(ctx, session.ID)
	if !errors.Is(err, ErrSessionInvalid) {
		t.Errorf("expected ErrSessionInvalid, got %v", err)
	}
}

func TestManager_InvalidateAllSessions(t *testing.T) {
	store := NewMemoryStore()
	manager := NewManager(store)
	defer func() { _ = manager.Close() }()

	ctx := context.Background()
	userID := "user-123"

	// Create multiple sessions
	for i := 0; i < 3; i++ {
		_, _ = manager.CreateSession(ctx, userID)
	}

	// Invalidate all
	count, err := manager.InvalidateAllSessions(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 3 {
		t.Errorf("expected 3 invalidated, got %d", count)
	}

	// List should be empty
	sessions, _ := manager.ListSessions(ctx, userID)
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestManager_InvalidateOtherSessions(t *testing.T) {
	store := NewMemoryStore()
	manager := NewManager(store)
	defer func() { _ = manager.Close() }()

	ctx := context.Background()
	userID := "user-123"

	// Create multiple sessions
	var currentSession *Session
	for i := 0; i < 3; i++ {
		s, _ := manager.CreateSession(ctx, userID)
		if i == 0 {
			currentSession = s
		}
	}

	// Invalidate others
	count, err := manager.InvalidateOtherSessions(ctx, userID, currentSession.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 invalidated, got %d", count)
	}

	// Current session should still exist
	_, err = manager.ValidateSession(ctx, currentSession.ID)
	if err != nil {
		t.Error("current session should still be valid")
	}
}

func TestManager_MaxSessionsPerUser(t *testing.T) {
	store := NewMemoryStore()
	manager := NewManager(store, WithMaxSessionsPerUser(2))
	defer func() { _ = manager.Close() }()

	ctx := context.Background()
	userID := "user-123"

	// Create 3 sessions (should keep only 2)
	var sessions []*Session
	for i := 0; i < 3; i++ {
		s, _ := manager.CreateSession(ctx, userID)
		sessions = append(sessions, s)
		time.Sleep(time.Millisecond) // Ensure different creation times
	}

	// Should have 2 sessions
	list, _ := manager.ListSessions(ctx, userID)
	if len(list) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(list))
	}

	// Oldest session should be removed
	_, err := manager.GetSession(ctx, sessions[0].ID)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Error("oldest session should be removed")
	}
}

func TestManager_RefreshSession(t *testing.T) {
	store := NewMemoryStore()
	manager := NewManager(store, WithSessionTTL(time.Hour))
	defer func() { _ = manager.Close() }()

	ctx := context.Background()

	// Create session
	session, _ := manager.CreateSession(ctx, "user-123")
	originalExpiry := session.ExpiresAt

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Refresh
	err := manager.RefreshSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get updated session
	updated, _ := manager.GetSession(ctx, session.ID)
	if !updated.ExpiresAt.After(originalExpiry) {
		t.Error("expiration should be extended")
	}
}

func TestSession_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{"future", time.Now().Add(time.Hour), false},
		{"past", time.Now().Add(-time.Hour), true},
		{"now", time.Now(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Session{ExpiresAt: tt.expiresAt}
			if s.IsExpired() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, s.IsExpired())
			}
		})
	}
}
