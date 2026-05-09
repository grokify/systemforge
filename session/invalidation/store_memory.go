package invalidation

import (
	"context"
	"sync"
	"time"
)

// MemoryStore is an in-memory implementation of SessionStore.
// Suitable for single-instance deployments and development/testing.
type MemoryStore struct {
	sessions map[string]*Session
	byUser   map[string]map[string]struct{} // userID -> sessionIDs
	mu       sync.RWMutex
	closed   bool
}

// NewMemoryStore creates a new in-memory session store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]*Session),
		byUser:   make(map[string]map[string]struct{}),
	}
}

// Create implements SessionStore.
func (m *MemoryStore) Create(ctx context.Context, session *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStorageFailure
	}

	// Clone the session to prevent external modifications
	s := *session
	if session.Metadata != nil {
		s.Metadata = make(map[string]string, len(session.Metadata))
		for k, v := range session.Metadata {
			s.Metadata[k] = v
		}
	}

	m.sessions[session.ID] = &s

	// Index by user
	if m.byUser[session.UserID] == nil {
		m.byUser[session.UserID] = make(map[string]struct{})
	}
	m.byUser[session.UserID][session.ID] = struct{}{}

	return nil
}

// Get implements SessionStore.
func (m *MemoryStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrStorageFailure
	}

	session, ok := m.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}

	// Clone before returning
	s := *session
	if session.Metadata != nil {
		s.Metadata = make(map[string]string, len(session.Metadata))
		for k, v := range session.Metadata {
			s.Metadata[k] = v
		}
	}

	return &s, nil
}

// Update implements SessionStore.
func (m *MemoryStore) Update(ctx context.Context, session *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStorageFailure
	}

	if _, ok := m.sessions[session.ID]; !ok {
		return ErrSessionNotFound
	}

	// Clone the session
	s := *session
	if session.Metadata != nil {
		s.Metadata = make(map[string]string, len(session.Metadata))
		for k, v := range session.Metadata {
			s.Metadata[k] = v
		}
	}

	m.sessions[session.ID] = &s
	return nil
}

// Delete implements SessionStore.
func (m *MemoryStore) Delete(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStorageFailure
	}

	session, ok := m.sessions[sessionID]
	if !ok {
		return nil // Already deleted
	}

	delete(m.sessions, sessionID)

	// Remove from user index
	if userSessions, ok := m.byUser[session.UserID]; ok {
		delete(userSessions, sessionID)
		if len(userSessions) == 0 {
			delete(m.byUser, session.UserID)
		}
	}

	return nil
}

// ListByUser implements SessionStore.
func (m *MemoryStore) ListByUser(ctx context.Context, userID string) ([]*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrStorageFailure
	}

	sessionIDs, ok := m.byUser[userID]
	if !ok {
		return nil, nil
	}

	sessions := make([]*Session, 0, len(sessionIDs))
	for sessionID := range sessionIDs {
		if session, ok := m.sessions[sessionID]; ok {
			// Clone before adding
			s := *session
			if session.Metadata != nil {
				s.Metadata = make(map[string]string, len(session.Metadata))
				for k, v := range session.Metadata {
					s.Metadata[k] = v
				}
			}
			sessions = append(sessions, &s)
		}
	}

	return sessions, nil
}

// DeleteByUser implements SessionStore.
func (m *MemoryStore) DeleteByUser(ctx context.Context, userID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, ErrStorageFailure
	}

	sessionIDs, ok := m.byUser[userID]
	if !ok {
		return 0, nil
	}

	count := len(sessionIDs)
	for sessionID := range sessionIDs {
		delete(m.sessions, sessionID)
	}
	delete(m.byUser, userID)

	return count, nil
}

// DeleteByDevice implements SessionStore.
func (m *MemoryStore) DeleteByDevice(ctx context.Context, userID, deviceID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, ErrStorageFailure
	}

	sessionIDs, ok := m.byUser[userID]
	if !ok {
		return 0, nil
	}

	var toDelete []string
	for sessionID := range sessionIDs {
		if session, ok := m.sessions[sessionID]; ok {
			if session.DeviceID == deviceID {
				toDelete = append(toDelete, sessionID)
			}
		}
	}

	for _, sessionID := range toDelete {
		delete(m.sessions, sessionID)
		delete(sessionIDs, sessionID)
	}

	if len(sessionIDs) == 0 {
		delete(m.byUser, userID)
	}

	return len(toDelete), nil
}

// DeleteExpired implements SessionStore.
func (m *MemoryStore) DeleteExpired(ctx context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, ErrStorageFailure
	}

	now := time.Now()
	var toDelete []string

	for sessionID, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			toDelete = append(toDelete, sessionID)
		}
	}

	for _, sessionID := range toDelete {
		session := m.sessions[sessionID]
		delete(m.sessions, sessionID)

		if userSessions, ok := m.byUser[session.UserID]; ok {
			delete(userSessions, sessionID)
			if len(userSessions) == 0 {
				delete(m.byUser, session.UserID)
			}
		}
	}

	return len(toDelete), nil
}

// Close implements SessionStore.
func (m *MemoryStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true
	m.sessions = nil
	m.byUser = nil
	return nil
}

// Count returns the number of active sessions (for testing).
func (m *MemoryStore) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}
