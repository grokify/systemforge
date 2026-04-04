package bff

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// EntClientInterface abstracts the Ent client to avoid import cycles.
// Apps implement this interface wrapping their generated Ent client.
//
// Example implementation:
//
//	type EntClientWrapper struct {
//	    client *ent.Client
//	}
//
//	func (w *EntClientWrapper) CreateBFFSession(ctx context.Context, session *bff.Session) error {
//	    _, err := w.client.BFFSession.Create().
//	        SetID(session.ID).
//	        SetUserID(session.UserID).
//	        // ... set other fields
//	        Save(ctx)
//	    return err
//	}
type EntClientInterface interface {
	// CreateBFFSession stores a new session.
	CreateBFFSession(ctx context.Context, session *Session) error

	// GetBFFSession retrieves a session by ID.
	// Returns ErrSessionNotFound if not found.
	GetBFFSession(ctx context.Context, id string) (*Session, error)

	// UpdateBFFSession updates an existing session.
	UpdateBFFSession(ctx context.Context, session *Session) error

	// DeleteBFFSession removes a session by ID.
	DeleteBFFSession(ctx context.Context, id string) error

	// DeleteBFFSessionsByUserID removes all sessions for a user.
	// Returns the number of sessions deleted.
	DeleteBFFSessionsByUserID(ctx context.Context, userID uuid.UUID) (int, error)

	// TouchBFFSession updates the LastAccessedAt timestamp.
	TouchBFFSession(ctx context.Context, id string) error

	// CleanupExpiredBFFSessions removes expired sessions.
	// Returns the number of sessions removed.
	CleanupExpiredBFFSessions(ctx context.Context, limit int) (int, error)
}

// EntStoreConfig configures the Ent-backed session store.
type EntStoreConfig struct {
	// Client is the Ent client wrapper.
	// Apps must provide a wrapper that implements EntClientInterface.
	Client EntClientInterface

	// EncryptionKey for encrypting tokens at rest.
	// Must be 32 bytes for AES-256.
	EncryptionKey []byte

	// CleanupInterval is how often to run automatic cleanup.
	// Set to 0 to disable automatic cleanup.
	CleanupInterval time.Duration

	// CleanupBatchSize limits sessions deleted per cleanup run.
	// Default is 100 if not specified.
	CleanupBatchSize int
}

// EntStore implements Store using Ent.
type EntStore struct {
	config    EntStoreConfig
	encryptor *Encryptor
	cleanup   *time.Ticker
	done      chan struct{}
}

// NewEntStore creates a new Ent-backed session store.
func NewEntStore(config EntStoreConfig) (*EntStore, error) {
	if config.Client == nil {
		return nil, errors.New("client is required")
	}
	if len(config.EncryptionKey) != 32 {
		return nil, ErrInvalidKeySize
	}

	encryptor, err := NewEncryptor(config.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("creating encryptor: %w", err)
	}

	if config.CleanupBatchSize <= 0 {
		config.CleanupBatchSize = 100
	}

	store := &EntStore{
		config:    config,
		encryptor: encryptor,
		done:      make(chan struct{}),
	}

	if config.CleanupInterval > 0 {
		store.startCleanup()
	}

	return store, nil
}

// Create stores a new session.
func (s *EntStore) Create(ctx context.Context, session *Session) error {
	// Encrypt tokens before storage
	encryptedSession, err := s.encryptSession(session)
	if err != nil {
		return fmt.Errorf("encrypting session: %w", err)
	}

	return s.config.Client.CreateBFFSession(ctx, encryptedSession)
}

// Get retrieves a session by ID.
func (s *EntStore) Get(ctx context.Context, id string) (*Session, error) {
	session, err := s.config.Client.GetBFFSession(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check expiration
	if session.IsExpired() {
		return nil, ErrSessionExpired
	}

	// Decrypt tokens
	decryptedSession, err := s.decryptSession(session)
	if err != nil {
		return nil, fmt.Errorf("decrypting session: %w", err)
	}

	return decryptedSession, nil
}

// Update updates an existing session.
func (s *EntStore) Update(ctx context.Context, session *Session) error {
	// Encrypt tokens before storage
	encryptedSession, err := s.encryptSession(session)
	if err != nil {
		return fmt.Errorf("encrypting session: %w", err)
	}

	encryptedSession.UpdatedAt = time.Now()
	return s.config.Client.UpdateBFFSession(ctx, encryptedSession)
}

// Delete removes a session by ID.
func (s *EntStore) Delete(ctx context.Context, id string) error {
	return s.config.Client.DeleteBFFSession(ctx, id)
}

// DeleteByUserID removes all sessions for a user.
func (s *EntStore) DeleteByUserID(ctx context.Context, userID string) (int, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID: %w", err)
	}
	return s.config.Client.DeleteBFFSessionsByUserID(ctx, uid)
}

// Touch updates the LastAccessedAt timestamp.
func (s *EntStore) Touch(ctx context.Context, id string) error {
	return s.config.Client.TouchBFFSession(ctx, id)
}

// Cleanup removes expired sessions.
func (s *EntStore) Cleanup(ctx context.Context) (int, error) {
	return s.config.Client.CleanupExpiredBFFSessions(ctx, s.config.CleanupBatchSize)
}

// Close stops the cleanup goroutine.
func (s *EntStore) Close() error {
	if s.cleanup != nil {
		s.cleanup.Stop()
		close(s.done)
	}
	return nil
}

// startCleanup starts the background cleanup goroutine.
func (s *EntStore) startCleanup() {
	s.cleanup = time.NewTicker(s.config.CleanupInterval)
	go func() {
		for {
			select {
			case <-s.cleanup.C:
				// Run cleanup in a background context
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				_, _ = s.Cleanup(ctx) // Ignore errors in background cleanup
				cancel()
			case <-s.done:
				return
			}
		}
	}()
}

// encryptSession encrypts the tokens in a session.
// Returns a new Session with encrypted token fields.
func (s *EntStore) encryptSession(session *Session) (*Session, error) {
	// Make a copy to avoid modifying the original
	encrypted := *session

	// Encrypt access token
	accessEncrypted, err := s.encryptor.EncryptString(session.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("encrypting access token: %w", err)
	}

	// Encrypt refresh token
	refreshEncrypted, err := s.encryptor.EncryptString(session.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("encrypting refresh token: %w", err)
	}

	// Store encrypted values in metadata (the actual encrypted bytes will be
	// passed to the client interface separately)
	encrypted.AccessToken = ""
	encrypted.RefreshToken = ""

	// Use a special field to pass encrypted data to the client
	// The client wrapper is responsible for storing these bytes
	encrypted.encryptedAccessToken = accessEncrypted
	encrypted.encryptedRefreshToken = refreshEncrypted

	// Encrypt DPoP key pair if present
	if len(session.DPoPKeyPairJSON) > 0 {
		dpopEncrypted, err := s.encryptor.Encrypt(session.DPoPKeyPairJSON)
		if err != nil {
			return nil, fmt.Errorf("encrypting dpop key pair: %w", err)
		}
		encrypted.DPoPKeyPairJSON = nil
		encrypted.encryptedDPoPKeyPair = dpopEncrypted
	}

	return &encrypted, nil
}

// decryptSession decrypts the tokens in a session.
// Returns a new Session with decrypted token fields.
func (s *EntStore) decryptSession(session *Session) (*Session, error) {
	// Make a copy to avoid modifying the original
	decrypted := *session

	// Decrypt access token
	if len(session.encryptedAccessToken) > 0 {
		accessToken, err := s.encryptor.DecryptString(session.encryptedAccessToken)
		if err != nil {
			return nil, fmt.Errorf("decrypting access token: %w", err)
		}
		decrypted.AccessToken = accessToken
	}

	// Decrypt refresh token
	if len(session.encryptedRefreshToken) > 0 {
		refreshToken, err := s.encryptor.DecryptString(session.encryptedRefreshToken)
		if err != nil {
			return nil, fmt.Errorf("decrypting refresh token: %w", err)
		}
		decrypted.RefreshToken = refreshToken
	}

	// Decrypt DPoP key pair if present
	if len(session.encryptedDPoPKeyPair) > 0 {
		dpopKeyPair, err := s.encryptor.Decrypt(session.encryptedDPoPKeyPair)
		if err != nil {
			return nil, fmt.Errorf("decrypting dpop key pair: %w", err)
		}
		decrypted.DPoPKeyPairJSON = dpopKeyPair
	}

	// Clear encrypted fields
	decrypted.encryptedAccessToken = nil
	decrypted.encryptedRefreshToken = nil
	decrypted.encryptedDPoPKeyPair = nil

	return &decrypted, nil
}

// Verify EntStore implements Store interface.
var _ Store = (*EntStore)(nil)
