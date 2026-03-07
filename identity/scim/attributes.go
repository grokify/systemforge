package scim

import (
	"encoding/json"
	"strings"
)

// AttributeFilter filters SCIM resources to include/exclude specific attributes.
type AttributeFilter struct {
	// Attributes to include (if empty, include all)
	Attributes []string

	// Attributes to exclude
	ExcludedAttributes []string
}

// NewAttributeFilter creates a new attribute filter.
func NewAttributeFilter(attributes, excludedAttributes []string) *AttributeFilter {
	return &AttributeFilter{
		Attributes:         normalizeAttributes(attributes),
		ExcludedAttributes: normalizeAttributes(excludedAttributes),
	}
}

// normalizeAttributes converts attributes to lowercase for case-insensitive matching.
func normalizeAttributes(attrs []string) []string {
	result := make([]string, len(attrs))
	for i, a := range attrs {
		result[i] = strings.ToLower(strings.TrimSpace(a))
	}
	return result
}

// IsEmpty returns true if no filtering is configured.
func (f *AttributeFilter) IsEmpty() bool {
	return len(f.Attributes) == 0 && len(f.ExcludedAttributes) == 0
}

// FilterUser applies attribute filtering to a User resource.
func (f *AttributeFilter) FilterUser(user *User) *User {
	if f.IsEmpty() {
		return user
	}

	// Convert to map, filter, convert back
	data, err := json.Marshal(user)
	if err != nil {
		return user
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return user
	}

	filtered := f.filterMap(m, "")

	// Always include required fields
	f.ensureRequiredFields(filtered, m, []string{"schemas", "id", "meta"})

	data, err = json.Marshal(filtered)
	if err != nil {
		return user
	}

	var result User
	if err := json.Unmarshal(data, &result); err != nil {
		return user
	}

	return &result
}

// FilterGroup applies attribute filtering to a Group resource.
func (f *AttributeFilter) FilterGroup(group *Group) *Group {
	if f.IsEmpty() {
		return group
	}

	data, err := json.Marshal(group)
	if err != nil {
		return group
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return group
	}

	filtered := f.filterMap(m, "")

	// Always include required fields
	f.ensureRequiredFields(filtered, m, []string{"schemas", "id", "meta"})

	data, err = json.Marshal(filtered)
	if err != nil {
		return group
	}

	var result Group
	if err := json.Unmarshal(data, &result); err != nil {
		return group
	}

	return &result
}

// FilterResource applies attribute filtering to any SCIM resource.
func (f *AttributeFilter) FilterResource(resource any) any {
	switch r := resource.(type) {
	case *User:
		return f.FilterUser(r)
	case *Group:
		return f.FilterGroup(r)
	default:
		return resource
	}
}

// filterMap filters a map based on attribute rules.
func (f *AttributeFilter) filterMap(m map[string]any, prefix string) map[string]any {
	result := make(map[string]any)

	for key, value := range m {
		attrPath := key
		if prefix != "" {
			attrPath = prefix + "." + key
		}
		attrPathLower := strings.ToLower(attrPath)
		keyLower := strings.ToLower(key)

		// Check if excluded
		if f.isExcluded(attrPathLower, keyLower) {
			continue
		}

		// If specific attributes are requested, check if included
		if len(f.Attributes) > 0 && !f.isIncluded(attrPathLower, keyLower) {
			continue
		}

		// Handle nested objects
		if nested, ok := value.(map[string]any); ok {
			filtered := f.filterMap(nested, attrPath)
			if len(filtered) > 0 {
				result[key] = filtered
			}
		} else {
			result[key] = value
		}
	}

	return result
}

// isIncluded checks if an attribute should be included.
func (f *AttributeFilter) isIncluded(attrPath, key string) bool {
	for _, a := range f.Attributes {
		// Exact match
		if a == attrPath || a == key {
			return true
		}
		// Parent match (e.g., "name" includes "name.givenName")
		if strings.HasPrefix(attrPath, a+".") {
			return true
		}
		// Child match (e.g., "name.givenName" includes parent "name")
		if strings.HasPrefix(a, attrPath+".") {
			return true
		}
	}
	return false
}

// isExcluded checks if an attribute should be excluded.
func (f *AttributeFilter) isExcluded(attrPath, key string) bool {
	for _, a := range f.ExcludedAttributes {
		// Exact match
		if a == attrPath || a == key {
			return true
		}
		// Parent match excludes children
		if strings.HasPrefix(attrPath, a+".") {
			return true
		}
	}
	return false
}

// ensureRequiredFields ensures required fields are present in the filtered result.
func (f *AttributeFilter) ensureRequiredFields(filtered, original map[string]any, required []string) {
	for _, field := range required {
		if _, exists := filtered[field]; !exists {
			if val, ok := original[field]; ok {
				filtered[field] = val
			}
		}
	}
}

// FilterListResponse applies attribute filtering to all resources in a list response.
func (f *AttributeFilter) FilterListResponse(response *ListResponse) *ListResponse {
	if f.IsEmpty() || response == nil {
		return response
	}

	filtered := make([]any, len(response.Resources))
	for i, r := range response.Resources {
		filtered[i] = f.FilterResource(r)
	}

	return &ListResponse{
		Schemas:      response.Schemas,
		TotalResults: response.TotalResults,
		StartIndex:   response.StartIndex,
		ItemsPerPage: response.ItemsPerPage,
		Resources:    filtered,
	}
}
