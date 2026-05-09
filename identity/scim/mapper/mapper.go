// Package mapper provides mapping between SCIM resources and SystemForge entities.
package mapper

import (
	"context"

	"github.com/grokify/systemforge/identity/scim/patch"
)

// Mapper defines the interface for mapping between SCIM resources and SystemForge entities.
type Mapper[S any, E any] interface {
	// ToSCIM converts a SystemForge entity to a SCIM resource.
	ToSCIM(ctx context.Context, entity E) (S, error)

	// FromSCIM converts a SCIM resource to a SystemForge entity or update input.
	// Returns an interface{} that can be either a create input or update input.
	FromSCIM(ctx context.Context, resource S) (any, error)

	// ApplyPatch applies PATCH operations to an entity.
	ApplyPatch(ctx context.Context, entity E, ops []patch.Operation) (E, error)
}

// Config contains configuration for mappers.
type Config struct {
	// BaseURL is the base URL for SCIM resources.
	BaseURL string

	// IncludeGroups determines whether to include group memberships in user resources.
	IncludeGroups bool

	// IncludeMembers determines whether to include members in group resources.
	IncludeMembers bool
}

// DefaultConfig returns the default mapper configuration.
func DefaultConfig() *Config {
	return &Config{
		BaseURL:        "/scim/v2",
		IncludeGroups:  true,
		IncludeMembers: true,
	}
}
