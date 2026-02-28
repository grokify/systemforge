// Package stores provides feature flag storage implementations.
package stores

import (
	"context"
	"errors"
	"sync"

	"github.com/grokify/coreforge/featureflags"
)

// ErrFlagNotFound is returned when a flag is not found.
var ErrFlagNotFound = errors.New("flag not found")

// MemoryStore is an in-memory feature flag store.
// Suitable for development or single-instance deployments.
type MemoryStore struct {
	mu    sync.RWMutex
	flags map[string]*featureflags.Flag
}

// NewMemoryStore creates a new in-memory flag store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		flags: make(map[string]*featureflags.Flag),
	}
}

// Get retrieves a flag by key.
func (s *MemoryStore) Get(ctx context.Context, key string) (*featureflags.Flag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	flag, ok := s.flags[key]
	if !ok {
		return nil, nil
	}

	// Return a copy to prevent mutation
	return copyFlag(flag), nil
}

// GetAll retrieves all flags.
func (s *MemoryStore) GetAll(ctx context.Context) ([]*featureflags.Flag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	flags := make([]*featureflags.Flag, 0, len(s.flags))
	for _, flag := range s.flags {
		flags = append(flags, copyFlag(flag))
	}

	return flags, nil
}

// Set creates or updates a flag.
func (s *MemoryStore) Set(ctx context.Context, flag *featureflags.Flag) error {
	if flag == nil || flag.Key == "" {
		return errors.New("invalid flag: key is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.flags[flag.Key] = copyFlag(flag)
	return nil
}

// Delete removes a flag.
func (s *MemoryStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.flags, key)
	return nil
}

// copyFlag creates a deep copy of a flag.
func copyFlag(f *featureflags.Flag) *featureflags.Flag {
	if f == nil {
		return nil
	}

	copy := &featureflags.Flag{
		Key:         f.Key,
		Name:        f.Name,
		Description: f.Description,
		Enabled:     f.Enabled,
		Percentage:  f.Percentage,
	}

	if len(f.Targets) > 0 {
		copy.Targets = make([]string, len(f.Targets))
		for i, t := range f.Targets {
			copy.Targets[i] = t
		}
	}

	if len(f.Metadata) > 0 {
		copy.Metadata = make(map[string]any, len(f.Metadata))
		for k, v := range f.Metadata {
			copy.Metadata[k] = v
		}
	}

	return copy
}

// Preload loads initial flags into the store.
func (s *MemoryStore) Preload(flags []*featureflags.Flag) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, flag := range flags {
		if flag != nil && flag.Key != "" {
			s.flags[flag.Key] = copyFlag(flag)
		}
	}
}
