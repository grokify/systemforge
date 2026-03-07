package scim

import (
	"github.com/grokify/coreforge/identity/scim/schema"
)

// ListUsersOutput is the response for listing users.
type ListUsersOutput struct {
	Body *ListResponse
}

// GetUserOutput is the response for getting a single user.
type GetUserOutput struct {
	Body *User
	ETag string `header:"ETag" doc:"Entity tag for caching and conditional requests"`
}

// CreateUserOutput is the response for creating a user.
type CreateUserOutput struct {
	Body     *User
	Location string `header:"Location" doc:"URI of the created resource"`
	ETag     string `header:"ETag" doc:"Entity tag for caching and conditional requests"`
}

// UpdateUserOutput is the response for updating a user.
type UpdateUserOutput struct {
	Body *User
	ETag string `header:"ETag" doc:"Entity tag for caching and conditional requests"`
}

// ListGroupsOutput is the response for listing groups.
type ListGroupsOutput struct {
	Body *ListResponse
}

// GetGroupOutput is the response for getting a single group.
type GetGroupOutput struct {
	Body *Group
	ETag string `header:"ETag" doc:"Entity tag for caching and conditional requests"`
}

// CreateGroupOutput is the response for creating a group.
type CreateGroupOutput struct {
	Body     *Group
	Location string `header:"Location" doc:"URI of the created resource"`
	ETag     string `header:"ETag" doc:"Entity tag for caching and conditional requests"`
}

// UpdateGroupOutput is the response for updating a group.
type UpdateGroupOutput struct {
	Body *Group
	ETag string `header:"ETag" doc:"Entity tag for caching and conditional requests"`
}

// ServiceProviderConfigOutput is the response for the ServiceProviderConfig endpoint.
type ServiceProviderConfigOutput struct {
	Body schema.ServiceProviderConfig
}

// ListSchemasOutput is the response for listing schemas.
type ListSchemasOutput struct {
	Body *ListResponse
}

// GetSchemaOutput is the response for getting a single schema.
type GetSchemaOutput struct {
	Body *schema.Schema
}

// ListResourceTypesOutput is the response for listing resource types.
type ListResourceTypesOutput struct {
	Body *ListResponse
}

// GetResourceTypeOutput is the response for getting a single resource type.
type GetResourceTypeOutput struct {
	Body *schema.ResourceType
}

// BulkOutput is the response for bulk operations.
type BulkOutput struct {
	Body *BulkResponse
}

// MeOutput is the response for /Me endpoint.
type MeOutput struct {
	Body *User
	ETag string `header:"ETag" doc:"Entity tag for caching and conditional requests"`
}

// SearchOutput is the response for search operations.
type SearchOutput struct {
	Body *ListResponse
}
