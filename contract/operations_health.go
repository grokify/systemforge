package contract

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

// registerHealthEndpoints registers health endpoints.
func (a *API) registerHealthEndpoints() {
	base := a.provider.Config().BaseURL

	// Health check
	huma.Register(a.huma, huma.Operation{
		OperationID: "getHealth",
		Method:      "GET",
		Path:        base + "/health",
		Summary:     "Get application health",
		Description: "Returns application health status with component checks. This endpoint is public.",
		Tags:        []string{"Health"},
	}, a.getHealth)

	// Federation health
	huma.Register(a.huma, huma.Operation{
		OperationID: "getFederationHealth",
		Method:      "GET",
		Path:        base + "/health/federation",
		Summary:     "Get federation health",
		Description: "Returns federation status and sync health. Requires authentication in federated mode.",
		Tags:        []string{"Health", "Federation"},
		Security: []map[string][]string{
			{"bearer": {"health:read"}},
		},
	}, a.getFederationHealth)
}

func (a *API) getHealth(ctx context.Context, input *struct{}) (*HealthOutput, error) {
	// Run health checks
	checks := make(map[string]string)
	overallStatus := "healthy"

	for name, checker := range a.provider.HealthCheckers() {
		status := checker.Check(ctx)
		checks[name] = status

		// Degrade overall status if any check is not healthy
		if status != "healthy" {
			if status == "unhealthy" {
				overallStatus = "unhealthy"
			} else if overallStatus != "unhealthy" {
				overallStatus = "degraded"
			}
		}
	}

	return &HealthOutput{
		Body: struct {
			Status        string            `json:"status" doc:"Overall health status" enum:"healthy,degraded,unhealthy" example:"healthy"`
			Version       string            `json:"version" doc:"Application version" example:"1.2.0"`
			UptimeSeconds int64             `json:"uptime_seconds" doc:"Seconds since startup" example:"86400"`
			Checks        map[string]string `json:"checks,omitempty" doc:"Health check results by component"`
		}{
			Status:        overallStatus,
			Version:       a.provider.Config().Version,
			UptimeSeconds: a.provider.UptimeSeconds(),
			Checks:        checks,
		},
	}, nil
}

func (a *API) getFederationHealth(ctx context.Context, input *struct{}) (*FederationHealthOutput, error) {
	// Check permissions (only in federated mode)
	if a.provider.FederationState().IsFederated() {
		if err := a.checkPermission(ctx, PermissionHealthRead); err != nil {
			return nil, err
		}
	}

	state := a.provider.FederationState()

	var federationStatus string
	if !state.IsFederated() {
		federationStatus = "standalone"
	} else {
		// In federated mode, determine if we're connected or disconnected
		// based on sync lag (if > 5 minutes, consider disconnected)
		if state.SyncLagSeconds() > 300 {
			federationStatus = "disconnected"
		} else {
			federationStatus = "connected"
		}
	}

	checks := make(map[string]string)

	// Add federation-specific checks if federated
	if state.IsFederated() {
		// Check identity sync health
		if state.LastIdentitySync() != nil {
			checks["identity_sync"] = "healthy"
		} else {
			checks["identity_sync"] = "pending"
		}

		// Check policy sync health
		if state.LastPolicySync() != nil {
			checks["policy_sync"] = "healthy"
		} else {
			checks["policy_sync"] = "pending"
		}
	}

	return &FederationHealthOutput{
		Body: struct {
			FederationStatus string            `json:"federation_status" doc:"Federation connection status" enum:"standalone,connected,disconnected" example:"standalone"`
			FederationID     *uuid.UUID        `json:"federation_id,omitempty" doc:"Federation identifier" format:"uuid"`
			LastSync         *time.Time        `json:"last_sync,omitempty" doc:"Last sync timestamp" format:"date-time"`
			SyncLagSeconds   int               `json:"sync_lag_seconds,omitempty" doc:"Seconds since last sync" example:"5"`
			Checks           map[string]string `json:"checks,omitempty" doc:"Federation health check results"`
		}{
			FederationStatus: federationStatus,
			FederationID:     state.FederationID(),
			LastSync:         state.LastSync(),
			SyncLagSeconds:   state.SyncLagSeconds(),
			Checks:           checks,
		},
	}, nil
}
