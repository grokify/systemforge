// Package contract implements the CoreForge Product Contract specification,
// providing standardized endpoints for integration with CoreControl federation.
package contract

import (
	"fmt"
	"slices"
)

// Capability represents a supported contract capability.
type Capability string

const (
	// CapabilityIdentity indicates principal management support.
	CapabilityIdentity Capability = "identity"
	// CapabilityRBAC indicates role-based access control support.
	CapabilityRBAC Capability = "rbac"
	// CapabilityAudit indicates audit event emission support.
	CapabilityAudit Capability = "audit"
	// CapabilityTenancy indicates multi-tenant support.
	CapabilityTenancy Capability = "tenancy"
	// CapabilityDelegation indicates agent delegation support.
	CapabilityDelegation Capability = "delegation"
)

// DefaultContractVersion is the current contract specification version.
const DefaultContractVersion = "1.0"

// Config holds configuration for the contract endpoints.
type Config struct {
	// BaseURL is the base path for contract endpoints (default: "/coreforge").
	BaseURL string

	// AppID is the unique application identifier.
	AppID string

	// DisplayName is the human-readable application name.
	DisplayName string

	// Version is the application version (semver format).
	Version string

	// ContractVersion is the contract specification version implemented.
	ContractVersion string

	// Capabilities lists the supported contract capabilities.
	Capabilities []Capability

	// CoreControlIssuer is the expected JWT issuer for CoreControl tokens.
	// Required for federated mode authentication.
	CoreControlIssuer string

	// CoreControlPublicKey is the public key for validating CoreControl JWTs.
	// Can be *rsa.PublicKey, *ecdsa.PublicKey, or ed25519.PublicKey.
	CoreControlPublicKey any
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		BaseURL:         "/coreforge",
		ContractVersion: DefaultContractVersion,
		Capabilities: []Capability{
			CapabilityIdentity,
			CapabilityRBAC,
		},
	}
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	if c.AppID == "" {
		return fmt.Errorf("contract: app_id is required")
	}
	if c.DisplayName == "" {
		return fmt.Errorf("contract: display_name is required")
	}
	if c.Version == "" {
		return fmt.Errorf("contract: version is required")
	}
	if c.BaseURL == "" {
		c.BaseURL = "/coreforge"
	}
	if c.ContractVersion == "" {
		c.ContractVersion = DefaultContractVersion
	}
	return nil
}

// HasCapability checks if a capability is enabled.
func (c *Config) HasCapability(cap Capability) bool {
	return slices.Contains(c.Capabilities, cap)
}

// CapabilityStrings returns capabilities as a string slice.
func (c *Config) CapabilityStrings() []string {
	result := make([]string, len(c.Capabilities))
	for i, cap := range c.Capabilities {
		result[i] = string(cap)
	}
	return result
}

// EndpointPaths returns the endpoint paths based on configuration.
func (c *Config) EndpointPaths() map[string]string {
	paths := make(map[string]string)

	if c.HasCapability(CapabilityIdentity) {
		paths["identity"] = c.BaseURL + "/identity"
	}
	if c.HasCapability(CapabilityRBAC) {
		paths["policy"] = c.BaseURL + "/policy"
	}
	if c.HasCapability(CapabilityAudit) {
		paths["audit"] = c.BaseURL + "/audit"
	}
	paths["health"] = c.BaseURL + "/health"

	return paths
}
