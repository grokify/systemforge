package audit

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryStore is an in-memory implementation of the audit Store interface.
// This is suitable for development and testing but should not be used in production.
type MemoryStore struct {
	mu               sync.RWMutex
	events           []*Event
	lastSequence     int64
	lastAcknowledged int64
	maxEvents        int // Maximum events to keep in memory
}

// NewMemoryStore creates a new in-memory audit store.
func NewMemoryStore(maxEvents int) *MemoryStore {
	if maxEvents <= 0 {
		maxEvents = 10000
	}
	return &MemoryStore{
		events:    make([]*Event, 0),
		maxEvents: maxEvents,
	}
}

// Record stores an audit event and returns its sequence number.
func (s *MemoryStore) Record(ctx context.Context, event *Event) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastSequence++
	event.Sequence = s.lastSequence

	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	s.events = append(s.events, event)

	// Trim if over max
	if len(s.events) > s.maxEvents {
		s.events = s.events[len(s.events)-s.maxEvents:]
	}

	return s.lastSequence, nil
}

// GetBySequence retrieves events starting from a sequence number.
func (s *MemoryStore) GetBySequence(ctx context.Context, fromSequence int64, limit int) ([]*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Event, 0)
	for _, e := range s.events {
		if e.Sequence >= fromSequence {
			result = append(result, e)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

// GetByTimeRange retrieves events within a time range.
func (s *MemoryStore) GetByTimeRange(ctx context.Context, from, to time.Time, limit int) ([]*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Event, 0)
	for _, e := range s.events {
		if (e.Timestamp.After(from) || e.Timestamp.Equal(from)) && e.Timestamp.Before(to) {
			result = append(result, e)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

// GetByActor retrieves events for a specific actor.
func (s *MemoryStore) GetByActor(ctx context.Context, actorID uuid.UUID, limit int) ([]*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Event, 0)
	for _, e := range s.events {
		if e.Actor.ID == actorID {
			result = append(result, e)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

// GetByResource retrieves events for a specific resource.
func (s *MemoryStore) GetByResource(ctx context.Context, resourceType string, resourceID uuid.UUID, limit int) ([]*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Event, 0)
	for _, e := range s.events {
		if e.Resource.Type == resourceType && e.Resource.ID == resourceID {
			result = append(result, e)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

// GetLastSequence returns the last recorded sequence number.
func (s *MemoryStore) GetLastSequence(ctx context.Context) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastSequence, nil
}

// Acknowledge marks events up to a sequence as acknowledged.
func (s *MemoryStore) Acknowledge(ctx context.Context, sequence int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sequence > s.lastAcknowledged {
		s.lastAcknowledged = sequence
	}
	return nil
}

// GetLastAcknowledged returns the last acknowledged sequence number.
func (s *MemoryStore) GetLastAcknowledged(ctx context.Context) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastAcknowledged, nil
}

// Ensure MemoryStore implements Store.
var _ Store = (*MemoryStore)(nil)
