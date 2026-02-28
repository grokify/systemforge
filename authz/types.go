package authz

import (
	"github.com/google/uuid"
)

// Principal represents an entity requesting access (user, service, etc.).
type Principal struct {
	// ID is the unique identifier of the principal.
	ID uuid.UUID

	// Type identifies the kind of principal (user, service, api_key, etc.).
	Type PrincipalType

	// Attributes contains additional principal attributes for ABAC.
	Attributes map[string]any
}

// PrincipalType identifies the kind of principal.
type PrincipalType string

const (
	PrincipalTypeUser    PrincipalType = "user"
	PrincipalTypeService PrincipalType = "service"
	PrincipalTypeAPIKey  PrincipalType = "api_key"
	PrincipalTypeSystem  PrincipalType = "system"
)

// NewUserPrincipal creates a Principal for a user.
func NewUserPrincipal(userID uuid.UUID) Principal {
	return Principal{
		ID:         userID,
		Type:       PrincipalTypeUser,
		Attributes: make(map[string]any),
	}
}

// NewUserPrincipalWithAttrs creates a Principal for a user with attributes.
func NewUserPrincipalWithAttrs(userID uuid.UUID, attrs map[string]any) Principal {
	return Principal{
		ID:         userID,
		Type:       PrincipalTypeUser,
		Attributes: attrs,
	}
}

// NewServicePrincipal creates a Principal for a service.
func NewServicePrincipal(serviceID uuid.UUID) Principal {
	return Principal{
		ID:         serviceID,
		Type:       PrincipalTypeService,
		Attributes: make(map[string]any),
	}
}

// WithAttr returns a copy of the Principal with an additional attribute.
func (p Principal) WithAttr(key string, value any) Principal {
	attrs := make(map[string]any, len(p.Attributes)+1)
	for k, v := range p.Attributes {
		attrs[k] = v
	}
	attrs[key] = value
	return Principal{
		ID:         p.ID,
		Type:       p.Type,
		Attributes: attrs,
	}
}

// Action represents an operation being performed.
type Action string

// Common action constants.
const (
	ActionCreate Action = "create"
	ActionRead   Action = "read"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
	ActionList   Action = "list"
	ActionManage Action = "manage" // Full CRUD + admin operations
)

// Resource represents something being accessed.
type Resource struct {
	// Type identifies the kind of resource (course, organization, etc.).
	Type ResourceType

	// ID is the unique identifier of the resource (nil for type-level checks).
	ID *uuid.UUID

	// OwnerID is the owner of the resource (for ownership-based access).
	OwnerID *uuid.UUID

	// OrgID is the organization this resource belongs to.
	OrgID *uuid.UUID

	// Attributes contains additional resource attributes for ABAC.
	Attributes map[string]any
}

// ResourceType identifies the kind of resource.
type ResourceType string

// Common resource type constants.
const (
	ResourceTypeOrganization ResourceType = "organization"
	ResourceTypeMember       ResourceType = "member"
	ResourceTypeUser         ResourceType = "user"
)

// NewResource creates a Resource with the given type.
func NewResource(resourceType ResourceType) Resource {
	return Resource{
		Type:       resourceType,
		Attributes: make(map[string]any),
	}
}

// NewResourceWithID creates a Resource with type and ID.
func NewResourceWithID(resourceType ResourceType, id uuid.UUID) Resource {
	return Resource{
		Type:       resourceType,
		ID:         &id,
		Attributes: make(map[string]any),
	}
}

// NewOrgResource creates a Resource scoped to an organization.
func NewOrgResource(resourceType ResourceType, orgID uuid.UUID) Resource {
	return Resource{
		Type:       resourceType,
		OrgID:      &orgID,
		Attributes: make(map[string]any),
	}
}

// NewOwnedResource creates a Resource with an owner.
func NewOwnedResource(resourceType ResourceType, id, ownerID uuid.UUID) Resource {
	return Resource{
		Type:       resourceType,
		ID:         &id,
		OwnerID:    &ownerID,
		Attributes: make(map[string]any),
	}
}

// WithOwner returns a copy of the Resource with the specified owner.
func (r Resource) WithOwner(ownerID uuid.UUID) Resource {
	return Resource{
		Type:       r.Type,
		ID:         r.ID,
		OwnerID:    &ownerID,
		OrgID:      r.OrgID,
		Attributes: r.Attributes,
	}
}

// WithOrg returns a copy of the Resource with the specified organization.
func (r Resource) WithOrg(orgID uuid.UUID) Resource {
	return Resource{
		Type:       r.Type,
		ID:         r.ID,
		OwnerID:    r.OwnerID,
		OrgID:      &orgID,
		Attributes: r.Attributes,
	}
}

// WithAttr returns a copy of the Resource with an additional attribute.
func (r Resource) WithAttr(key string, value any) Resource {
	attrs := make(map[string]any, len(r.Attributes)+1)
	for k, v := range r.Attributes {
		attrs[k] = v
	}
	attrs[key] = value
	return Resource{
		Type:       r.Type,
		ID:         r.ID,
		OwnerID:    r.OwnerID,
		OrgID:      r.OrgID,
		Attributes: attrs,
	}
}

// IsOwner checks if the principal owns this resource.
func (r Resource) IsOwner(principalID uuid.UUID) bool {
	return r.OwnerID != nil && *r.OwnerID == principalID
}
