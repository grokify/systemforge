package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				AppID:       "test-app",
				DisplayName: "Test App",
				Version:     "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "missing app_id",
			config: &Config{
				DisplayName: "Test App",
				Version:     "1.0.0",
			},
			wantErr: true,
		},
		{
			name: "missing display_name",
			config: &Config{
				AppID:   "test-app",
				Version: "1.0.0",
			},
			wantErr: true,
		},
		{
			name: "missing version",
			config: &Config{
				AppID:       "test-app",
				DisplayName: "Test App",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigHasCapability(t *testing.T) {
	config := &Config{
		Capabilities: []Capability{CapabilityIdentity, CapabilityRBAC},
	}

	if !config.HasCapability(CapabilityIdentity) {
		t.Error("expected HasCapability(Identity) to return true")
	}
	if !config.HasCapability(CapabilityRBAC) {
		t.Error("expected HasCapability(RBAC) to return true")
	}
	if config.HasCapability(CapabilityAudit) {
		t.Error("expected HasCapability(Audit) to return false")
	}
}

func TestFederationState(t *testing.T) {
	state := NewFederationState()

	// Test initial state
	if state.IsFederated() {
		t.Error("expected initial state to be standalone")
	}

	status := state.Status()
	if status.Status != FederationStatusStandalone {
		t.Errorf("expected status %q, got %q", FederationStatusStandalone, status.Status)
	}

	// Test sync in progress
	if !state.StartSync() {
		t.Error("expected StartSync to return true")
	}
	if state.StartSync() {
		t.Error("expected second StartSync to return false")
	}
	if !state.IsSyncInProgress() {
		t.Error("expected IsSyncInProgress to return true")
	}

	state.EndSync()
	if state.IsSyncInProgress() {
		t.Error("expected IsSyncInProgress to return false after EndSync")
	}
}

// metaResponseBody represents the body structure for deserialization.
type metaResponseBody struct {
	AppID           string            `json:"app_id"`
	DisplayName     string            `json:"display_name"`
	Version         string            `json:"version"`
	ContractVersion string            `json:"contract_version"`
	Capabilities    []string          `json:"capabilities"`
	Endpoints       map[string]string `json:"endpoints"`
	Federation      FederationStatus  `json:"federation"`
}

// healthResponseBody represents the body structure for deserialization.
type healthResponseBody struct {
	Status        string            `json:"status"`
	Version       string            `json:"version"`
	UptimeSeconds int64             `json:"uptime_seconds"`
	Checks        map[string]string `json:"checks,omitempty"`
}

// federationHealthResponseBody represents the body structure for deserialization.
type federationHealthResponseBody struct {
	FederationStatus string            `json:"federation_status"`
	SyncLagSeconds   int               `json:"sync_lag_seconds,omitempty"`
	Checks           map[string]string `json:"checks,omitempty"`
}

func TestMetaEndpoint(t *testing.T) {
	config := &Config{
		AppID:        "test-app",
		DisplayName:  "Test Application",
		Version:      "1.0.0",
		BaseURL:      "/systemforge",
		Capabilities: []Capability{CapabilityIdentity, CapabilityRBAC},
	}

	provider, err := NewProvider(config, nil)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	api, err := NewAPI(provider)
	if err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/systemforge/meta", nil)
	w := httptest.NewRecorder()

	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response metaResponseBody
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.AppID != config.AppID {
		t.Errorf("expected app_id %q, got %q", config.AppID, response.AppID)
	}
	if response.DisplayName != config.DisplayName {
		t.Errorf("expected display_name %q, got %q", config.DisplayName, response.DisplayName)
	}
	if response.Version != config.Version {
		t.Errorf("expected version %q, got %q", config.Version, response.Version)
	}
	if response.Federation.Status != FederationStatusStandalone {
		t.Errorf("expected federation status %q, got %q", FederationStatusStandalone, response.Federation.Status)
	}
}

func TestMetaEndpointMethodNotAllowed(t *testing.T) {
	config := &Config{
		AppID:       "test-app",
		DisplayName: "Test Application",
		Version:     "1.0.0",
		BaseURL:     "/systemforge",
	}

	provider, err := NewProvider(config, nil)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	api, err := NewAPI(provider)
	if err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/systemforge/meta", nil)
	w := httptest.NewRecorder()

	api.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestHealthEndpoint(t *testing.T) {
	config := &Config{
		AppID:       "test-app",
		DisplayName: "Test Application",
		Version:     "1.0.0",
		BaseURL:     "/systemforge",
	}

	provider, err := NewProvider(config, nil)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	api, err := NewAPI(provider)
	if err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/systemforge/health", nil)
	w := httptest.NewRecorder()

	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response healthResponseBody
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("expected status %q, got %q", "healthy", response.Status)
	}
	if response.Version != config.Version {
		t.Errorf("expected version %q, got %q", config.Version, response.Version)
	}
}

func TestFederationHealthEndpoint(t *testing.T) {
	config := &Config{
		AppID:       "test-app",
		DisplayName: "Test Application",
		Version:     "1.0.0",
		BaseURL:     "/systemforge",
	}

	provider, err := NewProvider(config, nil)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	api, err := NewAPI(provider)
	if err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/systemforge/health/federation", nil)
	w := httptest.NewRecorder()

	api.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response federationHealthResponseBody
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.FederationStatus != "standalone" {
		t.Errorf("expected federation_status %q, got %q", "standalone", response.FederationStatus)
	}
}
