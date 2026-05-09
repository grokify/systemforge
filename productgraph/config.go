package productgraph

import (
	"net/http"
	"os"
	"strconv"
	"time"
)

// Config holds ProductGraph client configuration.
type Config struct {
	// ProjectID is the ProductGraph project identifier (required).
	ProjectID string

	// Endpoint is the ProductGraph API endpoint (required).
	// Example: "https://api.productgraph.io/v1/events"
	Endpoint string

	// APIKey is the API key for authentication (optional).
	// Sent as X-PG-API-Key header.
	APIKey string

	// BatchSize is the number of events to buffer before flushing.
	// Default: 50
	BatchSize int

	// BatchInterval is the maximum time to wait before flushing buffered events.
	// Default: 5 seconds
	BatchInterval time.Duration

	// HTTPClient is the HTTP client to use for requests.
	// Default: http.DefaultClient with 10s timeout
	HTTPClient *http.Client

	// Debug enables debug logging.
	Debug bool
}

// ConfigFromEnv creates a Config from environment variables.
//
// Environment variables:
//   - PRODUCTGRAPH_PROJECT_ID: Project identifier (required)
//   - PRODUCTGRAPH_ENDPOINT: API endpoint (required)
//   - PRODUCTGRAPH_API_KEY: API key for authentication
//   - PRODUCTGRAPH_BATCH_SIZE: Events per batch (default: 50)
//   - PRODUCTGRAPH_BATCH_INTERVAL: Flush interval in seconds (default: 5)
//   - PRODUCTGRAPH_DEBUG: Enable debug logging (true/false)
func ConfigFromEnv() Config {
	cfg := Config{
		ProjectID:     os.Getenv("PRODUCTGRAPH_PROJECT_ID"),
		Endpoint:      os.Getenv("PRODUCTGRAPH_ENDPOINT"),
		APIKey:        os.Getenv("PRODUCTGRAPH_API_KEY"),
		BatchSize:     50,
		BatchInterval: 5 * time.Second,
		Debug:         os.Getenv("PRODUCTGRAPH_DEBUG") == "true",
	}

	if v := os.Getenv("PRODUCTGRAPH_BATCH_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.BatchSize = n
		}
	}

	if v := os.Getenv("PRODUCTGRAPH_BATCH_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.BatchInterval = time.Duration(n) * time.Second
		}
	}

	return cfg
}

// IsEnabled returns true if ProductGraph integration is configured.
func (c Config) IsEnabled() bool {
	return c.ProjectID != "" && c.Endpoint != ""
}

// Validate returns an error if the configuration is invalid.
func (c Config) Validate() error {
	if c.ProjectID == "" {
		return ErrMissingProjectID
	}
	if c.Endpoint == "" {
		return ErrMissingEndpoint
	}
	return nil
}

// WithDefaults returns a copy of the config with default values applied.
func (c Config) WithDefaults() Config {
	if c.BatchSize <= 0 {
		c.BatchSize = 50
	}
	if c.BatchInterval <= 0 {
		c.BatchInterval = 5 * time.Second
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	return c
}
