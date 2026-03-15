package coreapi

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Common errors returned by the policy store.
var (
	ErrPolicyNotFound  = errors.New("policy not found")
	ErrPolicyExists    = errors.New("policy already exists")
	ErrBindingNotFound = errors.New("client binding not found")
)

// MemoryPolicyStore is an in-memory implementation of PolicyStore.
// Suitable for development, testing, or single-instance deployments.
type MemoryPolicyStore struct {
	policies        map[string]*RateLimitPolicy
	bindings        map[string]string // clientID -> policyID
	defaultPolicyID string
	mu              sync.RWMutex
}

// NewMemoryPolicyStore creates a new in-memory policy store.
// It initializes with a default "standard" policy.
func NewMemoryPolicyStore() *MemoryPolicyStore {
	store := &MemoryPolicyStore{
		policies:        make(map[string]*RateLimitPolicy),
		bindings:        make(map[string]string),
		defaultPolicyID: "standard",
	}

	// Initialize with standard policy as default
	now := time.Now()
	defaultPolicy := *StandardPolicy
	defaultPolicy.CreatedAt = now
	defaultPolicy.UpdatedAt = now
	store.policies["standard"] = &defaultPolicy

	return store
}

// NewMemoryPolicyStoreWithPolicies creates a store initialized with the given policies.
func NewMemoryPolicyStoreWithPolicies(policies []*RateLimitPolicy, defaultPolicyID string) *MemoryPolicyStore {
	store := &MemoryPolicyStore{
		policies:        make(map[string]*RateLimitPolicy),
		bindings:        make(map[string]string),
		defaultPolicyID: defaultPolicyID,
	}

	now := time.Now()
	for _, p := range policies {
		copy := *p
		if copy.CreatedAt.IsZero() {
			copy.CreatedAt = now
		}
		if copy.UpdatedAt.IsZero() {
			copy.UpdatedAt = now
		}
		store.policies[p.ID] = &copy
	}

	return store
}

// GetPolicy implements PolicyStore.
func (s *MemoryPolicyStore) GetPolicy(ctx context.Context, policyID string) (*RateLimitPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policy, ok := s.policies[policyID]
	if !ok {
		return nil, ErrPolicyNotFound
	}

	// Return a copy to prevent modification
	copy := *policy
	return &copy, nil
}

// ListPolicies implements PolicyStore.
func (s *MemoryPolicyStore) ListPolicies(ctx context.Context) ([]*RateLimitPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policies := make([]*RateLimitPolicy, 0, len(s.policies))
	for _, p := range s.policies {
		copy := *p
		policies = append(policies, &copy)
	}
	return policies, nil
}

// CreatePolicy implements PolicyStore.
func (s *MemoryPolicyStore) CreatePolicy(ctx context.Context, policy *RateLimitPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.policies[policy.ID]; exists {
		return ErrPolicyExists
	}

	copy := *policy
	now := time.Now()
	copy.CreatedAt = now
	copy.UpdatedAt = now
	s.policies[policy.ID] = &copy
	return nil
}

// UpdatePolicy implements PolicyStore.
func (s *MemoryPolicyStore) UpdatePolicy(ctx context.Context, policy *RateLimitPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.policies[policy.ID]
	if !ok {
		return ErrPolicyNotFound
	}

	copy := *policy
	copy.CreatedAt = existing.CreatedAt
	copy.UpdatedAt = time.Now()
	s.policies[policy.ID] = &copy
	return nil
}

// DeletePolicy implements PolicyStore.
func (s *MemoryPolicyStore) DeletePolicy(ctx context.Context, policyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.policies[policyID]; !ok {
		return ErrPolicyNotFound
	}

	// Don't allow deleting the default policy
	if policyID == s.defaultPolicyID {
		return errors.New("cannot delete the default policy")
	}

	delete(s.policies, policyID)

	// Remove any bindings to this policy
	for clientID, boundPolicyID := range s.bindings {
		if boundPolicyID == policyID {
			delete(s.bindings, clientID)
		}
	}

	return nil
}

// GetDefaultPolicy implements PolicyStore.
func (s *MemoryPolicyStore) GetDefaultPolicy(ctx context.Context) (*RateLimitPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policy, ok := s.policies[s.defaultPolicyID]
	if !ok {
		// Fallback to standard if default doesn't exist
		policy = StandardPolicy
	}

	copy := *policy
	return &copy, nil
}

// SetDefaultPolicy implements PolicyStore.
func (s *MemoryPolicyStore) SetDefaultPolicy(ctx context.Context, policyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.policies[policyID]; !ok {
		return ErrPolicyNotFound
	}

	s.defaultPolicyID = policyID
	return nil
}

// GetPolicyForClient implements PolicyStore.
func (s *MemoryPolicyStore) GetPolicyForClient(ctx context.Context, clientID string) (*RateLimitPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policyID, ok := s.bindings[clientID]
	if !ok {
		// Return default policy
		policy, ok := s.policies[s.defaultPolicyID]
		if !ok {
			return StandardPolicy, nil
		}
		copy := *policy
		return &copy, nil
	}

	policy, ok := s.policies[policyID]
	if !ok {
		// Bound policy doesn't exist, return default
		policy, ok = s.policies[s.defaultPolicyID]
		if !ok {
			return StandardPolicy, nil
		}
	}

	if !policy.Enabled {
		// Policy disabled, return default
		policy, ok = s.policies[s.defaultPolicyID]
		if !ok {
			return StandardPolicy, nil
		}
	}

	copy := *policy
	return &copy, nil
}

// BindClientToPolicy implements PolicyStore.
func (s *MemoryPolicyStore) BindClientToPolicy(ctx context.Context, clientID, policyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.policies[policyID]; !ok {
		return ErrPolicyNotFound
	}

	s.bindings[clientID] = policyID
	return nil
}

// UnbindClient implements PolicyStore.
func (s *MemoryPolicyStore) UnbindClient(ctx context.Context, clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.bindings, clientID)
	return nil
}

// ListClientBindings implements PolicyStore.
func (s *MemoryPolicyStore) ListClientBindings(ctx context.Context) ([]*ClientPolicyBinding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bindings := make([]*ClientPolicyBinding, 0, len(s.bindings))
	for clientID, policyID := range s.bindings {
		bindings = append(bindings, &ClientPolicyBinding{
			ClientID: clientID,
			PolicyID: policyID,
			Enabled:  true,
		})
	}
	return bindings, nil
}

// ListClientsForPolicy implements PolicyStore.
func (s *MemoryPolicyStore) ListClientsForPolicy(ctx context.Context, policyID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var clients []string
	for clientID, boundPolicyID := range s.bindings {
		if boundPolicyID == policyID {
			clients = append(clients, clientID)
		}
	}
	return clients, nil
}
