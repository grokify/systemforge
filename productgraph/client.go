package productgraph

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Common errors.
var (
	ErrMissingProjectID = errors.New("productgraph: missing project_id")
	ErrMissingEndpoint  = errors.New("productgraph: missing endpoint")
	ErrClientClosed     = errors.New("productgraph: client is closed")
)

// Client sends events to ProductGraph with batching and async dispatch.
type Client struct {
	config Config
	logger *slog.Logger

	mu      sync.Mutex
	buffer  []Event
	closed  bool
	done    chan struct{}
	wg      sync.WaitGroup
	flushCh chan struct{}
}

// New creates a new ProductGraph client.
// Returns an error if the configuration is invalid.
func New(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cfg = cfg.WithDefaults()

	c := &Client{
		config:  cfg,
		logger:  slog.Default(),
		buffer:  make([]Event, 0, cfg.BatchSize),
		done:    make(chan struct{}),
		flushCh: make(chan struct{}, 1),
	}

	c.wg.Add(1)
	go c.flusher()

	return c, nil
}

// MustNew creates a new ProductGraph client and panics on error.
func MustNew(cfg Config) *Client {
	c, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return c
}

// SetLogger sets the logger for the client.
func (c *Client) SetLogger(logger *slog.Logger) {
	c.logger = logger
}

// Track queues an event for dispatch to ProductGraph.
// The event is buffered and sent asynchronously in batches.
func (c *Client) Track(ctx context.Context, event Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrClientClosed
	}

	// Fill in defaults
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	if event.ProjectID == "" {
		event.ProjectID = c.config.ProjectID
	}
	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	// Extract correlation IDs from context if not set
	if event.SessionID == "" {
		event.SessionID = SessionIDFromContext(ctx)
	}
	if event.UserID == "" {
		event.UserID = UserIDFromContext(ctx)
	}

	c.buffer = append(c.buffer, event)

	// Trigger flush if buffer is full
	if len(c.buffer) >= c.config.BatchSize {
		select {
		case c.flushCh <- struct{}{}:
		default:
		}
	}

	return nil
}

// TrackAPICall is a convenience method for tracking API calls.
func (c *Client) TrackAPICall(ctx context.Context, method, path string, statusCode int, duration time.Duration) error {
	return c.Track(ctx, APIResponseEvent(method, path, statusCode, int(duration.Milliseconds())))
}

// TrackError is a convenience method for tracking errors.
func (c *Client) TrackError(ctx context.Context, errType, message string) error {
	return c.Track(ctx, ErrorEvent(errType, message))
}

// TrackJourneyStep is a convenience method for tracking journey steps.
func (c *Client) TrackJourneyStep(ctx context.Context, journeyID, stepID, stepName string) error {
	return c.Track(ctx, JourneyStepEvent(journeyID, stepID, stepName))
}

// Flush forces all buffered events to be sent immediately.
func (c *Client) Flush(ctx context.Context) error {
	c.mu.Lock()
	events := c.buffer
	c.buffer = make([]Event, 0, c.config.BatchSize)
	c.mu.Unlock()

	if len(events) == 0 {
		return nil
	}

	return c.send(ctx, events)
}

// Close gracefully shuts down the client, flushing any remaining events.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	close(c.done)
	c.wg.Wait()

	// Final flush
	return c.Flush(context.Background())
}

// flusher is the background goroutine that flushes events periodically.
func (c *Client) flusher() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.BatchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.flushAsync()
		case <-c.flushCh:
			c.flushAsync()
		case <-c.done:
			return
		}
	}
}

// flushAsync flushes events without blocking.
func (c *Client) flushAsync() {
	c.mu.Lock()
	if len(c.buffer) == 0 {
		c.mu.Unlock()
		return
	}
	events := c.buffer
	c.buffer = make([]Event, 0, c.config.BatchSize)
	c.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.send(ctx, events); err != nil {
		c.logger.Error("productgraph: failed to send events",
			"error", err,
			"count", len(events),
		)
	} else if c.config.Debug {
		c.logger.Debug("productgraph: sent events",
			"count", len(events),
		)
	}
}

// send dispatches events to the ProductGraph API.
func (c *Client) send(ctx context.Context, events []Event) error {
	payload := Payload{Events: events}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("productgraph: failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.Endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("productgraph: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("X-PG-API-Key", c.config.APIKey)
	}

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("productgraph: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("productgraph: server returned status %d", resp.StatusCode)
	}

	return nil
}

// IsEnabled returns true if the client is configured and ready.
func (c *Client) IsEnabled() bool {
	return c != nil && c.config.IsEnabled()
}

// Config returns the client configuration.
func (c *Client) Config() Config {
	return c.config
}
