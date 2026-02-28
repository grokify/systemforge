package bff

import (
	"context"
	"errors"
	"sync"
	"time"
)

// MemoryStore is an in-memory session store for development and testing.
// It is thread-safe and supports automatic cleanup of expired sessions.
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	config   StoreConfig

	// cleanup
	stopCleanup chan struct{}
	cleanupDone chan struct{}
}

// NewMemoryStore creates a new in-memory session store.
func NewMemoryStore(config StoreConfig) *MemoryStore {
	store := &MemoryStore{
		sessions:    make(map[string]*Session),
		config:      config,
		stopCleanup: make(chan struct{}),
		cleanupDone: make(chan struct{}),
	}

	// Start background cleanup if interval is set
	if config.CleanupInterval > 0 {
		go store.runCleanup()
	}

	return store
}

// Create stores a new session.
func (s *MemoryStore) Create(ctx context.Context, session *Session) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if session == nil || session.ID == "" {
		return ErrInvalidSession
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check max sessions limit
	if s.config.MaxSessions > 0 && len(s.sessions) >= s.config.MaxSessions {
		// Try to clean up expired sessions first
		s.cleanupLocked()
		if len(s.sessions) >= s.config.MaxSessions {
			return errors.New("maximum sessions exceeded")
		}
	}

	// Make a copy to prevent external modification
	sessionCopy := *session
	s.sessions[session.ID] = &sessionCopy

	return nil
}

// Get retrieves a session by ID.
func (s *MemoryStore) Get(ctx context.Context, id string) (*Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	session, exists := s.sessions[id]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrSessionNotFound
	}

	if session.IsExpired() {
		// Clean up expired session
		s.mu.Lock()
		delete(s.sessions, id)
		s.mu.Unlock()
		return nil, ErrSessionExpired
	}

	// Return a copy to prevent external modification
	sessionCopy := *session
	return &sessionCopy, nil
}

// Update updates an existing session.
func (s *MemoryStore) Update(ctx context.Context, session *Session) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if session == nil || session.ID == "" {
		return ErrInvalidSession
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[session.ID]; !exists {
		return ErrSessionNotFound
	}

	session.UpdatedAt = time.Now()
	sessionCopy := *session
	s.sessions[session.ID] = &sessionCopy

	return nil
}

// Delete removes a session by ID.
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[id]; !exists {
		return ErrSessionNotFound
	}

	delete(s.sessions, id)
	return nil
}

// DeleteByUserID removes all sessions for a user.
func (s *MemoryStore) DeleteByUserID(ctx context.Context, userID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for id, session := range s.sessions {
		if session.UserID.String() == userID {
			delete(s.sessions, id)
			count++
		}
	}

	return count, nil
}

// Touch updates the LastAccessedAt timestamp.
func (s *MemoryStore) Touch(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[id]
	if !exists {
		return ErrSessionNotFound
	}

	session.LastAccessedAt = time.Now()
	return nil
}

// Cleanup removes expired sessions.
func (s *MemoryStore) Cleanup(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.cleanupLocked(), nil
}

// cleanupLocked removes expired sessions. Must be called with mu locked.
func (s *MemoryStore) cleanupLocked() int {
	count := 0
	now := time.Now()

	for id, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, id)
			count++
		}
	}

	return count
}

// runCleanup runs periodic cleanup in the background.
func (s *MemoryStore) runCleanup() {
	ticker := time.NewTicker(time.Duration(s.config.CleanupInterval) * time.Second)
	defer ticker.Stop()
	defer close(s.cleanupDone)

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			s.cleanupLocked()
			s.mu.Unlock()
		case <-s.stopCleanup:
			return
		}
	}
}

// Close stops the cleanup goroutine and releases resources.
func (s *MemoryStore) Close() error {
	if s.config.CleanupInterval > 0 {
		close(s.stopCleanup)
		<-s.cleanupDone
	}
	return nil
}

// Count returns the current number of sessions (for testing).
func (s *MemoryStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}
