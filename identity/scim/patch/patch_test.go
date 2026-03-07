package patch

import (
	"encoding/json"
	"testing"
)

func TestParsePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantAttr string
		wantSub  string
		wantFilt string
		wantURI  string
	}{
		{
			name:     "simple attribute",
			path:     "displayName",
			wantAttr: "displayName",
		},
		{
			name:     "nested attribute",
			path:     "name.givenName",
			wantAttr: "name",
			wantSub:  "givenName",
		},
		{
			name:     "with filter",
			path:     "emails[type eq \"work\"]",
			wantAttr: "emails",
			wantFilt: "type eq \"work\"",
		},
		{
			name:     "with filter and sub-attribute",
			path:     "emails[type eq \"work\"].value",
			wantAttr: "emails",
			wantFilt: "type eq \"work\"",
			wantSub:  "value",
		},
		{
			name:     "with URI prefix",
			path:     "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:employeeNumber",
			wantURI:  "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User",
			wantAttr: "employeeNumber",
		},
		{
			name:     "empty path",
			path:     "",
			wantAttr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := ParsePath(tt.path)
			if err != nil {
				t.Fatalf("ParsePath() error = %v", err)
			}

			if p == nil && tt.path != "" {
				t.Fatal("ParsePath() returned nil")
			}

			if p == nil {
				return
			}

			if p.Attribute != tt.wantAttr {
				t.Errorf("Attribute = %q, want %q", p.Attribute, tt.wantAttr)
			}
			if p.SubAttribute != tt.wantSub {
				t.Errorf("SubAttribute = %q, want %q", p.SubAttribute, tt.wantSub)
			}
			if p.Filter != tt.wantFilt {
				t.Errorf("Filter = %q, want %q", p.Filter, tt.wantFilt)
			}
			if p.URIPrefix != tt.wantURI {
				t.Errorf("URIPrefix = %q, want %q", p.URIPrefix, tt.wantURI)
			}
		})
	}
}

func TestParseFilter(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		wantAttr  string
		wantOp    string
		wantValue any
	}{
		{
			name:      "string equality",
			filter:    "type eq 'work'",
			wantAttr:  "type",
			wantOp:    "eq",
			wantValue: "work",
		},
		{
			name:      "string with double quotes",
			filter:    `value eq "john@example.com"`,
			wantAttr:  "value",
			wantOp:    "eq",
			wantValue: "john@example.com",
		},
		{
			name:      "boolean value",
			filter:    "primary eq true",
			wantAttr:  "primary",
			wantOp:    "eq",
			wantValue: true,
		},
		{
			name:      "null value",
			filter:    "value eq null",
			wantAttr:  "value",
			wantOp:    "eq",
			wantValue: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector, err := ParseFilter(tt.filter)
			if err != nil {
				t.Fatalf("ParseFilter() error = %v", err)
			}

			if selector.FilterAttr != tt.wantAttr {
				t.Errorf("FilterAttr = %q, want %q", selector.FilterAttr, tt.wantAttr)
			}
			if selector.FilterOp != tt.wantOp {
				t.Errorf("FilterOp = %q, want %q", selector.FilterOp, tt.wantOp)
			}
			if selector.FilterValue != tt.wantValue {
				t.Errorf("FilterValue = %v, want %v", selector.FilterValue, tt.wantValue)
			}
		})
	}
}

func TestSelectorMatches(t *testing.T) {
	tests := []struct {
		name   string
		filter string
		value  map[string]any
		want   bool
	}{
		{
			name:   "matching string",
			filter: "type eq 'work'",
			value:  map[string]any{"type": "work", "value": "john@example.com"},
			want:   true,
		},
		{
			name:   "non-matching string",
			filter: "type eq 'work'",
			value:  map[string]any{"type": "home", "value": "john@example.com"},
			want:   false,
		},
		{
			name:   "matching boolean",
			filter: "primary eq true",
			value:  map[string]any{"primary": true, "value": "john@example.com"},
			want:   true,
		},
		{
			name:   "case insensitive",
			filter: "type eq 'WORK'",
			value:  map[string]any{"type": "work"},
			want:   true,
		},
		{
			name:   "contains operator",
			filter: "value co '@example'",
			value:  map[string]any{"value": "john@example.com"},
			want:   true,
		},
		{
			name:   "starts with operator",
			filter: "value sw 'john'",
			value:  map[string]any{"value": "john@example.com"},
			want:   true,
		},
		{
			name:   "ends with operator",
			filter: "value ew '.com'",
			value:  map[string]any{"value": "john@example.com"},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector, err := ParseFilter(tt.filter)
			if err != nil {
				t.Fatalf("ParseFilter() error = %v", err)
			}

			got := selector.Matches(tt.value)
			if got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOperationValidate(t *testing.T) {
	tests := []struct {
		name      string
		op        Operation
		wantError bool
	}{
		{
			name:      "valid add",
			op:        Operation{Op: OpAdd, Path: "displayName", Value: "John"},
			wantError: false,
		},
		{
			name:      "add without value",
			op:        Operation{Op: OpAdd, Path: "displayName"},
			wantError: true,
		},
		{
			name:      "valid replace",
			op:        Operation{Op: OpReplace, Path: "displayName", Value: "John"},
			wantError: false,
		},
		{
			name:      "replace without value",
			op:        Operation{Op: OpReplace, Path: "displayName"},
			wantError: true,
		},
		{
			name:      "valid remove",
			op:        Operation{Op: OpRemove, Path: "displayName"},
			wantError: false,
		},
		{
			name:      "invalid operation",
			op:        Operation{Op: "invalid", Path: "displayName"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestOperationJSONMarshal(t *testing.T) {
	op := Operation{
		Op:    OpReplace,
		Path:  "displayName",
		Value: "New Name",
	}

	data, err := json.Marshal(op)
	if err != nil {
		t.Fatalf("failed to marshal operation: %v", err)
	}

	var decoded Operation
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal operation: %v", err)
	}

	if decoded.Op != op.Op {
		t.Errorf("Op = %q, want %q", decoded.Op, op.Op)
	}
	if decoded.Path != op.Path {
		t.Errorf("Path = %q, want %q", decoded.Path, op.Path)
	}
}

type testResource struct {
	DisplayName string `json:"displayName"`
	Active      *bool  `json:"active"`
	Title       string `json:"title"`
	Name        *struct {
		GivenName  string `json:"givenName"`
		FamilyName string `json:"familyName"`
	} `json:"name"`
	Emails []struct {
		Value   string `json:"value"`
		Type    string `json:"type"`
		Primary bool   `json:"primary"`
	} `json:"emails"`
}

func TestApplierAdd(t *testing.T) {
	applier := NewApplier()

	resource := &testResource{}
	ops := []Operation{
		{Op: OpAdd, Path: "displayName", Value: "John Doe"},
	}

	err := applier.Apply(resource, ops)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if resource.DisplayName != "John Doe" {
		t.Errorf("DisplayName = %q, want %q", resource.DisplayName, "John Doe")
	}
}

func TestApplierReplace(t *testing.T) {
	applier := NewApplier()

	resource := &testResource{
		DisplayName: "Old Name",
	}
	ops := []Operation{
		{Op: OpReplace, Path: "displayName", Value: "New Name"},
	}

	err := applier.Apply(resource, ops)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if resource.DisplayName != "New Name" {
		t.Errorf("DisplayName = %q, want %q", resource.DisplayName, "New Name")
	}
}

func TestApplierRemove(t *testing.T) {
	applier := NewApplier()

	resource := &testResource{
		DisplayName: "John Doe",
		Title:       "Manager",
	}
	ops := []Operation{
		{Op: OpRemove, Path: "title"},
	}

	err := applier.Apply(resource, ops)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if resource.Title != "" {
		t.Errorf("Title = %q, want empty", resource.Title)
	}
}

func TestApplierNestedAttribute(t *testing.T) {
	applier := NewApplier()

	resource := &testResource{
		Name: &struct {
			GivenName  string `json:"givenName"`
			FamilyName string `json:"familyName"`
		}{},
	}
	ops := []Operation{
		{Op: OpAdd, Path: "name.givenName", Value: "John"},
		{Op: OpAdd, Path: "name.familyName", Value: "Doe"},
	}

	err := applier.Apply(resource, ops)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if resource.Name.GivenName != "John" {
		t.Errorf("Name.GivenName = %q, want %q", resource.Name.GivenName, "John")
	}
	if resource.Name.FamilyName != "Doe" {
		t.Errorf("Name.FamilyName = %q, want %q", resource.Name.FamilyName, "Doe")
	}
}
