package scim

// ListResourcesInput contains common query parameters for list operations.
type ListResourcesInput struct {
	Filter             string `query:"filter" doc:"SCIM filter expression (RFC 7644)"`
	Attributes         string `query:"attributes" doc:"Comma-separated list of attributes to return"`
	ExcludedAttributes string `query:"excludedAttributes" doc:"Comma-separated list of attributes to exclude"`
	SortBy             string `query:"sortBy" doc:"Attribute path to sort by"`
	SortOrder          string `query:"sortOrder" enum:"ascending,descending" default:"ascending" doc:"Sort order"`
	StartIndex         int    `query:"startIndex" minimum:"1" default:"1" doc:"1-based index of first result"`
	Count              int    `query:"count" minimum:"0" doc:"Number of resources to return per page"`
}

// ToListOptions converts input parameters to ListOptions.
func (i *ListResourcesInput) ToListOptions(defaultCount int) ListOptions {
	count := i.Count
	if count == 0 {
		count = defaultCount
	}
	startIndex := i.StartIndex
	if startIndex == 0 {
		startIndex = 1
	}
	sortOrder := i.SortOrder
	if sortOrder == "" {
		sortOrder = "ascending"
	}

	var attributes []string
	if i.Attributes != "" {
		attributes = splitAndTrim(i.Attributes, ",")
	}

	return ListOptions{
		Filter:     i.Filter,
		StartIndex: startIndex,
		Count:      count,
		SortBy:     i.SortBy,
		SortOrder:  sortOrder,
		Attributes: attributes,
	}
}

// ToAttributeFilter creates an AttributeFilter from the input parameters.
func (i *ListResourcesInput) ToAttributeFilter() *AttributeFilter {
	var attributes, excludedAttributes []string
	if i.Attributes != "" {
		attributes = splitAndTrim(i.Attributes, ",")
	}
	if i.ExcludedAttributes != "" {
		excludedAttributes = splitAndTrim(i.ExcludedAttributes, ",")
	}
	return NewAttributeFilter(attributes, excludedAttributes)
}

// GetResourceInput contains parameters for getting a single resource.
type GetResourceInput struct {
	ID                 string `path:"id" doc:"Resource ID"`
	Attributes         string `query:"attributes" doc:"Comma-separated list of attributes to return"`
	ExcludedAttributes string `query:"excludedAttributes" doc:"Comma-separated list of attributes to exclude"`
	IfNoneMatch        string `header:"If-None-Match" doc:"ETag for conditional request"`
}

// ToAttributeFilter creates an AttributeFilter from the input parameters.
func (i *GetResourceInput) ToAttributeFilter() *AttributeFilter {
	var attributes, excludedAttributes []string
	if i.Attributes != "" {
		attributes = splitAndTrim(i.Attributes, ",")
	}
	if i.ExcludedAttributes != "" {
		excludedAttributes = splitAndTrim(i.ExcludedAttributes, ",")
	}
	return NewAttributeFilter(attributes, excludedAttributes)
}

// CreateUserInput contains the request body for creating a user.
type CreateUserInput struct {
	Body *User `doc:"User resource to create"`
}

// UpdateUserInput contains parameters for replacing a user.
type UpdateUserInput struct {
	ID      string `path:"id" doc:"User ID"`
	IfMatch string `header:"If-Match" doc:"ETag for conditional update"`
	Body    *User  `doc:"User resource to replace"`
}

// PatchResourceInput contains parameters for patching a resource.
type PatchResourceInput struct {
	ID      string        `path:"id" doc:"Resource ID"`
	IfMatch string        `header:"If-Match" doc:"ETag for conditional update"`
	Body    *PatchRequest `doc:"PATCH operations to apply"`
}

// DeleteResourceInput contains parameters for deleting a resource.
type DeleteResourceInput struct {
	ID string `path:"id" doc:"Resource ID"`
}

// CreateGroupInput contains the request body for creating a group.
type CreateGroupInput struct {
	Body *Group `doc:"Group resource to create"`
}

// UpdateGroupInput contains parameters for replacing a group.
type UpdateGroupInput struct {
	ID      string `path:"id" doc:"Group ID"`
	IfMatch string `header:"If-Match" doc:"ETag for conditional update"`
	Body    *Group `doc:"Group resource to replace"`
}

// SchemaIDInput contains the schema ID path parameter.
type SchemaIDInput struct {
	ID string `path:"id" doc:"Schema URN"`
}

// ResourceTypeNameInput contains the resource type name path parameter.
type ResourceTypeNameInput struct {
	Name string `path:"name" doc:"Resource type name"`
}

// SearchRequest represents a SCIM search request body (POST /.search).
type SearchRequest struct {
	Schemas            []string `json:"schemas,omitempty"`
	Attributes         []string `json:"attributes,omitempty"`
	ExcludedAttributes []string `json:"excludedAttributes,omitempty"`
	Filter             string   `json:"filter,omitempty"`
	SortBy             string   `json:"sortBy,omitempty"`
	SortOrder          string   `json:"sortOrder,omitempty"`
	StartIndex         int      `json:"startIndex,omitempty"`
	Count              int      `json:"count,omitempty"`
}

// ToListOptions converts a SearchRequest to ListOptions.
func (s *SearchRequest) ToListOptions(defaultCount int) ListOptions {
	count := s.Count
	if count == 0 {
		count = defaultCount
	}
	startIndex := s.StartIndex
	if startIndex == 0 {
		startIndex = 1
	}
	sortOrder := s.SortOrder
	if sortOrder == "" {
		sortOrder = "ascending"
	}

	return ListOptions{
		Filter:     s.Filter,
		StartIndex: startIndex,
		Count:      count,
		SortBy:     s.SortBy,
		SortOrder:  sortOrder,
		Attributes: s.Attributes,
	}
}

// ToAttributeFilter creates an AttributeFilter from the search request.
func (s *SearchRequest) ToAttributeFilter() *AttributeFilter {
	return NewAttributeFilter(s.Attributes, s.ExcludedAttributes)
}

// SearchInput contains the request body for a search operation.
type SearchInput struct {
	Body *SearchRequest `doc:"Search request body"`
}

// BulkInput contains the request body for bulk operations.
type BulkInput struct {
	Body *BulkRequest `doc:"Bulk request containing multiple operations"`
}

// MePatchInput contains parameters for patching the current user.
type MePatchInput struct {
	IfMatch string        `header:"If-Match" doc:"ETag for conditional update"`
	Body    *PatchRequest `doc:"PATCH operations to apply"`
}
