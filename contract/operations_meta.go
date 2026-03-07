package contract

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
)

// registerMetaEndpoints registers metadata endpoints.
func (a *API) registerMetaEndpoints() {
	huma.Register(a.huma, huma.Operation{
		OperationID: "getMeta",
		Method:      "GET",
		Path:        a.provider.Config().BaseURL + "/meta",
		Summary:     "Get application metadata",
		Description: "Returns application metadata including capabilities, version, and federation status. This endpoint is public and does not require authentication.",
		Tags:        []string{"Metadata"},
	}, func(ctx context.Context, input *struct{}) (*MetadataResponse, error) {
		config := a.provider.Config()
		state := a.provider.FederationState()

		return &MetadataResponse{
			Body: struct {
				AppID           string            `json:"app_id" doc:"Unique application identifier" example:"my-saas-app"`
				DisplayName     string            `json:"display_name" doc:"Human-readable application name" example:"My SaaS Application"`
				Version         string            `json:"version" doc:"Application version (semver)" example:"1.2.0"`
				ContractVersion string            `json:"contract_version" doc:"Contract specification version" example:"1.0"`
				Capabilities    []string          `json:"capabilities" doc:"Supported contract capabilities" example:"[\"identity\", \"rbac\", \"audit\"]"`
				Endpoints       map[string]string `json:"endpoints" doc:"Endpoint paths by capability"`
				Federation      FederationStatus  `json:"federation" doc:"Current federation status"`
			}{
				AppID:           config.AppID,
				DisplayName:     config.DisplayName,
				Version:         config.Version,
				ContractVersion: config.ContractVersion,
				Capabilities:    config.CapabilityStrings(),
				Endpoints:       config.EndpointPaths(),
				Federation:      state.Status(),
			},
		}, nil
	})
}
