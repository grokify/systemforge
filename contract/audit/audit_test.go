package audit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMemoryStoreRecord(t *testing.T) {
	store := NewMemoryStore(100)
	ctx := context.Background()

	event := &Event{
		EventType: "test.event",
		Action:    "create",
		Actor: Actor{
			ID:         uuid.New(),
			Type:       "human",
			Identifier: "user@example.com",
		},
		Resource: Resource{
			Type: "document",
			ID:   uuid.New(),
		},
		Outcome: "success",
	}

	seq, err := store.Record(ctx, event)
	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	if seq != 1 {
		t.Errorf("expected sequence 1, got %d", seq)
	}

	if event.Sequence != 1 {
		t.Errorf("expected event.Sequence 1, got %d", event.Sequence)
	}

	if event.ID == uuid.Nil {
		t.Error("expected event.ID to be set")
	}

	if event.Timestamp.IsZero() {
		t.Error("expected event.Timestamp to be set")
	}
}

func TestMemoryStoreGetBySequence(t *testing.T) {
	store := NewMemoryStore(100)
	ctx := context.Background()

	// Record multiple events
	for range 5 {
		event := &Event{
			EventType: "test.event",
			Action:    "create",
		}
		_, err := store.Record(ctx, event)
		if err != nil {
			t.Fatalf("Record() error = %v", err)
		}
	}

	// Get events from sequence 3
	events, err := store.GetBySequence(ctx, 3, 10)
	if err != nil {
		t.Fatalf("GetBySequence() error = %v", err)
	}

	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}

	if events[0].Sequence != 3 {
		t.Errorf("expected first event sequence 3, got %d", events[0].Sequence)
	}
}

func TestMemoryStoreGetByTimeRange(t *testing.T) {
	store := NewMemoryStore(100)
	ctx := context.Background()

	// Record an event
	now := time.Now()
	event := &Event{
		EventType: "test.event",
		Timestamp: now,
	}
	_, err := store.Record(ctx, event)
	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	// Query within range
	events, err := store.GetByTimeRange(ctx, now.Add(-time.Hour), now.Add(time.Hour), 10)
	if err != nil {
		t.Fatalf("GetByTimeRange() error = %v", err)
	}

	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}

	// Query outside range
	events, err = store.GetByTimeRange(ctx, now.Add(time.Hour), now.Add(2*time.Hour), 10)
	if err != nil {
		t.Fatalf("GetByTimeRange() error = %v", err)
	}

	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestMemoryStoreAcknowledge(t *testing.T) {
	store := NewMemoryStore(100)
	ctx := context.Background()

	// Initial acknowledged should be 0
	ack, err := store.GetLastAcknowledged(ctx)
	if err != nil {
		t.Fatalf("GetLastAcknowledged() error = %v", err)
	}
	if ack != 0 {
		t.Errorf("expected initial acknowledged 0, got %d", ack)
	}

	// Acknowledge sequence 5
	err = store.Acknowledge(ctx, 5)
	if err != nil {
		t.Fatalf("Acknowledge() error = %v", err)
	}

	ack, err = store.GetLastAcknowledged(ctx)
	if err != nil {
		t.Fatalf("GetLastAcknowledged() error = %v", err)
	}
	if ack != 5 {
		t.Errorf("expected acknowledged 5, got %d", ack)
	}

	// Acknowledging lower sequence should not change
	err = store.Acknowledge(ctx, 3)
	if err != nil {
		t.Fatalf("Acknowledge() error = %v", err)
	}

	ack, err = store.GetLastAcknowledged(ctx)
	if err != nil {
		t.Fatalf("GetLastAcknowledged() error = %v", err)
	}
	if ack != 5 {
		t.Errorf("expected acknowledged still 5, got %d", ack)
	}
}

func TestMemoryStoreMaxEvents(t *testing.T) {
	maxEvents := 5
	store := NewMemoryStore(maxEvents)
	ctx := context.Background()

	// Record more events than max
	for range 10 {
		event := &Event{
			EventType: "test.event",
		}
		_, err := store.Record(ctx, event)
		if err != nil {
			t.Fatalf("Record() error = %v", err)
		}
	}

	// Should only keep last maxEvents
	events, err := store.GetBySequence(ctx, 1, 100)
	if err != nil {
		t.Fatalf("GetBySequence() error = %v", err)
	}

	if len(events) != maxEvents {
		t.Errorf("expected %d events, got %d", maxEvents, len(events))
	}

	// First event should have sequence 6 (oldest after trimming)
	if events[0].Sequence != 6 {
		t.Errorf("expected first event sequence 6, got %d", events[0].Sequence)
	}
}
