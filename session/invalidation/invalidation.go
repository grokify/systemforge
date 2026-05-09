// Package invalidation provides session tracking and invalidation for user sessions.
// It enables features like "logout all devices" and session management.
package invalidation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

// Common errors returned by the session invalidation service.
var (
	ErrSessionNotFound  = errors.New("session not found")
	ErrSessionExpired   = errors.New("session expired")
	ErrSessionInvalid   = errors.New("session has been invalidated")
	ErrStorageFailure   = errors.New("session storage failure")
	ErrInvalidSessionID = errors.New("invalid session ID")
)

// Session represents an active user session.
type Session struct {
	// ID is the unique session identifier.
	ID string

	// UserID is the user who owns this session.
	UserID string

	// DeviceID is an optional device identifier.
	DeviceID string

	// DeviceInfo contains information about the device (user agent, etc.).
	DeviceInfo string

	// IPAddress is the IP address of the client.
	IPAddress string

	// CreatedAt is when the session was created.
	CreatedAt time.Time

	// LastActiveAt is when the session was last used.
	LastActiveAt time.Time

	// ExpiresAt is when the session expires.
	ExpiresAt time.Time

	// Metadata contains additional session data.
	Metadata map[string]string
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// SessionStore defines the storage interface for session tracking.
type SessionStore interface {
	// Create creates a new session.
	Create(ctx context.Context, session *Session) error

	// Get retrieves a session by ID.
	Get(ctx context.Context, sessionID string) (*Session, error)

	// Update updates a session (e.g., LastActiveAt).
	Update(ctx context.Context, session *Session) error

	// Delete deletes a session.
	Delete(ctx context.Context, sessionID string) error

	// ListByUser lists all sessions for a user.
	ListByUser(ctx context.Context, userID string) ([]*Session, error)

	// DeleteByUser deletes all sessions for a user.
	DeleteByUser(ctx context.Context, userID string) (int, error)

	// DeleteByDevice deletes sessions for a specific device.
	DeleteByDevice(ctx context.Context, userID, deviceID string) (int, error)

	// DeleteExpired removes expired sessions.
	DeleteExpired(ctx context.Context) (int, error)

	// Close releases resources.
	Close() error
}

// Config configures the session invalidation service.
type Config struct {
	// SessionTTL is the default session lifetime.
	// Default: 24 hours
	SessionTTL time.Duration

	// CleanupInterval is how often to clean up expired sessions.
	// Default: 1 hour
	CleanupInterval time.Duration

	// MaxSessionsPerUser limits sessions per user (0 = unlimited).
	// When exceeded, oldest sessions are removed.
	MaxSessionsPerUser int
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		SessionTTL:         24 * time.Hour,
		CleanupInterval:    time.Hour,
		MaxSessionsPerUser: 0,
	}
}

// Manager manages user sessions and provides invalidation capabilities.
type Manager struct {
	store  SessionStore
	config Config

	stopCleanup chan struct{}
	wg          sync.WaitGroup
}

// NewManager creates a new session manager.
func NewManager(store SessionStore, opts ...ManagerOption) *Manager {
	m := &Manager{
		store:       store,
		config:      DefaultConfig(),
		stopCleanup: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(m)
	}

	// Start cleanup goroutine
	m.wg.Add(1)
	go m.cleanupLoop()

	return m
}

// ManagerOption configures a Manager.
type ManagerOption func(*Manager)

// WithConfig sets the manager configuration.
func WithConfig(cfg Config) ManagerOption {
	return func(m *Manager) {
		m.config = cfg
	}
}

// WithSessionTTL sets the default session TTL.
func WithSessionTTL(ttl time.Duration) ManagerOption {
	return func(m *Manager) {
		m.config.SessionTTL = ttl
	}
}

// WithMaxSessionsPerUser limits sessions per user.
func WithMaxSessionsPerUser(max int) ManagerOption {
	return func(m *Manager) {
		m.config.MaxSessionsPerUser = max
	}
}

// CreateSession creates a new session for a user.
func (m *Manager) CreateSession(ctx context.Context, userID string, opts ...SessionOption) (*Session, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	session := &Session{
		ID:           sessionID,
		UserID:       userID,
		CreatedAt:    now,
		LastActiveAt: now,
		ExpiresAt:    now.Add(m.config.SessionTTL),
		Metadata:     make(map[string]string),
	}

	for _, opt := range opts {
		opt(session)
	}

	// Enforce max sessions if configured
	if m.config.MaxSessionsPerUser > 0 {
		if err := m.enforceMaxSessions(ctx, userID); err != nil {
			return nil, err
		}
	}

	if err := m.store.Create(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// SessionOption configures a session during creation.
type SessionOption func(*Session)

// WithDeviceID sets the device ID.
func WithDeviceID(deviceID string) SessionOption {
	return func(s *Session) {
		s.DeviceID = deviceID
	}
}

// WithDeviceInfo sets the device info.
func WithDeviceInfo(info string) SessionOption {
	return func(s *Session) {
		s.DeviceInfo = info
	}
}

// WithIPAddress sets the IP address.
func WithIPAddress(ip string) SessionOption {
	return func(s *Session) {
		s.IPAddress = ip
	}
}

// WithTTL sets a custom TTL for this session.
func WithTTL(ttl time.Duration) SessionOption {
	return func(s *Session) {
		s.ExpiresAt = s.CreatedAt.Add(ttl)
	}
}

// WithMetadata sets session metadata.
func WithMetadata(key, value string) SessionOption {
	return func(s *Session) {
		if s.Metadata == nil {
			s.Metadata = make(map[string]string)
		}
		s.Metadata[key] = value
	}
}

// ValidateSession checks if a session is valid and updates LastActiveAt.
func (m *Manager) ValidateSession(ctx context.Context, sessionID string) (*Session, error) {
	session, err := m.store.Get(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil, ErrSessionInvalid
		}
		return nil, err
	}

	if session.IsExpired() {
		// Clean up expired session
		_ = m.store.Delete(ctx, sessionID)
		return nil, ErrSessionExpired
	}

	// Update last active time
	session.LastActiveAt = time.Now()
	if err := m.store.Update(ctx, session); err != nil {
		// Log but don't fail - session is still valid
		_ = err
	}

	return session, nil
}

// GetSession retrieves a session without updating LastActiveAt.
func (m *Manager) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	return m.store.Get(ctx, sessionID)
}

// InvalidateSession invalidates a specific session.
func (m *Manager) InvalidateSession(ctx context.Context, sessionID string) error {
	return m.store.Delete(ctx, sessionID)
}

// InvalidateAllSessions invalidates all sessions for a user.
// Returns the number of sessions invalidated.
func (m *Manager) InvalidateAllSessions(ctx context.Context, userID string) (int, error) {
	return m.store.DeleteByUser(ctx, userID)
}

// InvalidateDeviceSessions invalidates sessions for a specific device.
// Returns the number of sessions invalidated.
func (m *Manager) InvalidateDeviceSessions(ctx context.Context, userID, deviceID string) (int, error) {
	return m.store.DeleteByDevice(ctx, userID, deviceID)
}

// InvalidateOtherSessions invalidates all sessions except the current one.
// Returns the number of sessions invalidated.
func (m *Manager) InvalidateOtherSessions(ctx context.Context, userID, currentSessionID string) (int, error) {
	sessions, err := m.store.ListByUser(ctx, userID)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, s := range sessions {
		if s.ID != currentSessionID {
			if err := m.store.Delete(ctx, s.ID); err == nil {
				count++
			}
		}
	}

	return count, nil
}

// ListSessions lists all active sessions for a user.
func (m *Manager) ListSessions(ctx context.Context, userID string) ([]*Session, error) {
	sessions, err := m.store.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Filter out expired sessions
	var active []*Session
	for _, s := range sessions {
		if !s.IsExpired() {
			active = append(active, s)
		}
	}

	return active, nil
}

// RefreshSession extends the session expiration.
func (m *Manager) RefreshSession(ctx context.Context, sessionID string) error {
	session, err := m.store.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.IsExpired() {
		return ErrSessionExpired
	}

	session.ExpiresAt = time.Now().Add(m.config.SessionTTL)
	session.LastActiveAt = time.Now()

	return m.store.Update(ctx, session)
}

// Close shuts down the manager and releases resources.
func (m *Manager) Close() error {
	close(m.stopCleanup)
	m.wg.Wait()
	return m.store.Close()
}

func (m *Manager) enforceMaxSessions(ctx context.Context, userID string) error {
	sessions, err := m.store.ListByUser(ctx, userID)
	if err != nil {
		return err
	}

	// Remove expired sessions first
	var active []*Session
	for _, s := range sessions {
		if s.IsExpired() {
			_ = m.store.Delete(ctx, s.ID)
		} else {
			active = append(active, s)
		}
	}

	// If still over limit, remove oldest sessions
	for len(active) >= m.config.MaxSessionsPerUser {
		oldest := active[0]
		for _, s := range active[1:] {
			if s.CreatedAt.Before(oldest.CreatedAt) {
				oldest = s
			}
		}

		if err := m.store.Delete(ctx, oldest.ID); err != nil {
			return err
		}

		// Remove from active list
		for i, s := range active {
			if s.ID == oldest.ID {
				active = append(active[:i], active[i+1:]...)
				break
			}
		}
	}

	return nil
}

func (m *Manager) cleanupLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCleanup:
			return
		case <-ticker.C:
			_, _ = m.store.DeleteExpired(context.Background())
		}
	}
}

func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
