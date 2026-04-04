package apikey

import (
	"context"
	"errors"

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
//	func (w *EntClientWrapper) CreateAPIKey(ctx context.Context, key *apikey.APIKey, keyHash string) error {
//	    _, err := w.client.APIKey.Create().
//	        SetID(key.ID).
//	        SetName(key.Name).
//	        SetKeyHash(keyHash).
//	        // ... set other fields
//	        Save(ctx)
//	    return err
//	}
type EntClientInterface interface {
	// CreateAPIKey stores a new API key.
	CreateAPIKey(ctx context.Context, key *APIKey, keyHash string) error

	// GetAPIKeyByPrefix retrieves a key by its prefix.
	// Returns the key and its hash for validation.
	// Returns ErrKeyNotFound if not found.
	GetAPIKeyByPrefix(ctx context.Context, prefix string) (*APIKey, string, error)

	// GetAPIKeyByID retrieves a key by its ID.
	// Returns ErrKeyNotFound if not found.
	GetAPIKeyByID(ctx context.Context, id uuid.UUID) (*APIKey, error)

	// ListAPIKeysByOwner lists all keys for a user.
	ListAPIKeysByOwner(ctx context.Context, ownerID uuid.UUID) ([]*APIKey, error)

	// ListAPIKeysByOrganization lists all keys for an organization.
	ListAPIKeysByOrganization(ctx context.Context, orgID uuid.UUID) ([]*APIKey, error)

	// UpdateAPIKey updates key metadata.
	UpdateAPIKey(ctx context.Context, key *APIKey) error

	// DeleteAPIKey permanently removes a key.
	DeleteAPIKey(ctx context.Context, id uuid.UUID) error

	// UpdateAPIKeyLastUsed updates the last used timestamp and IP.
	UpdateAPIKeyLastUsed(ctx context.Context, id uuid.UUID, ip string) error
}

// EntStoreConfig configures the Ent-backed API key store.
type EntStoreConfig struct {
	// Client is the Ent client wrapper.
	// Apps must provide a wrapper that implements EntClientInterface.
	Client EntClientInterface
}

// EntStore implements Store using Ent.
type EntStore struct {
	config EntStoreConfig
}

// NewEntStore creates a new Ent-backed API key store.
func NewEntStore(config EntStoreConfig) (*EntStore, error) {
	if config.Client == nil {
		return nil, errors.New("client is required")
	}
	return &EntStore{config: config}, nil
}

// Create stores a new API key.
func (s *EntStore) Create(ctx context.Context, key *APIKey, keyHash string) error {
	return s.config.Client.CreateAPIKey(ctx, key, keyHash)
}

// GetByPrefix retrieves a key by its prefix.
func (s *EntStore) GetByPrefix(ctx context.Context, prefix string) (*APIKey, string, error) {
	return s.config.Client.GetAPIKeyByPrefix(ctx, prefix)
}

// GetByID retrieves a key by its ID.
func (s *EntStore) GetByID(ctx context.Context, id uuid.UUID) (*APIKey, error) {
	return s.config.Client.GetAPIKeyByID(ctx, id)
}

// ListByOwner lists all keys for a user.
func (s *EntStore) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]*APIKey, error) {
	return s.config.Client.ListAPIKeysByOwner(ctx, ownerID)
}

// ListByOrganization lists all keys for an organization.
func (s *EntStore) ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]*APIKey, error) {
	return s.config.Client.ListAPIKeysByOrganization(ctx, orgID)
}

// Update updates key metadata.
func (s *EntStore) Update(ctx context.Context, key *APIKey) error {
	return s.config.Client.UpdateAPIKey(ctx, key)
}

// Delete permanently removes a key.
func (s *EntStore) Delete(ctx context.Context, id uuid.UUID) error {
	return s.config.Client.DeleteAPIKey(ctx, id)
}

// UpdateLastUsed updates the last used timestamp and IP.
func (s *EntStore) UpdateLastUsed(ctx context.Context, id uuid.UUID, ip string) error {
	return s.config.Client.UpdateAPIKeyLastUsed(ctx, id, ip)
}

// Verify EntStore implements Store interface.
var _ Store = (*EntStore)(nil)
