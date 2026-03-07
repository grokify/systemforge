package filter

import (
	"testing"
)

func TestParseFilter(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		wantType  NodeType
		wantError bool
	}{
		{
			name:     "simple equality",
			filter:   `userName eq "bjensen"`,
			wantType: NodeComparison,
		},
		{
			name:     "equality with single quotes",
			filter:   `userName eq 'bjensen'`,
			wantType: NodeComparison,
		},
		{
			name:     "present operator",
			filter:   `title pr`,
			wantType: NodeComparison,
		},
		{
			name:     "starts with",
			filter:   `userName sw "J"`,
			wantType: NodeComparison,
		},
		{
			name:     "ends with",
			filter:   `userName ew "son"`,
			wantType: NodeComparison,
		},
		{
			name:     "contains",
			filter:   `name.formatted co "John"`,
			wantType: NodeComparison,
		},
		{
			name:     "greater than",
			filter:   `meta.lastModified gt "2011-05-13T04:42:34Z"`,
			wantType: NodeComparison,
		},
		{
			name:     "less than or equal",
			filter:   `meta.lastModified le "2011-05-13T04:42:34Z"`,
			wantType: NodeComparison,
		},
		{
			name:     "boolean value",
			filter:   `active eq true`,
			wantType: NodeComparison,
		},
		{
			name:     "null value",
			filter:   `title eq null`,
			wantType: NodeComparison,
		},
		{
			name:     "logical and",
			filter:   `userName eq "john" and active eq true`,
			wantType: NodeLogicalAnd,
		},
		{
			name:     "logical or",
			filter:   `userName eq "john" or userName eq "jane"`,
			wantType: NodeLogicalOr,
		},
		{
			name:     "not with parentheses",
			filter:   `not (active eq false)`,
			wantType: NodeLogicalNot,
		},
		{
			name:     "complex expression",
			filter:   `(userName eq "john" or userName eq "jane") and active eq true`,
			wantType: NodeLogicalAnd,
		},
		{
			name:     "value path filter",
			filter:   `emails[type eq "work"]`,
			wantType: NodeValuePath,
		},
		{
			name:     "empty filter",
			filter:   "",
			wantType: -1, // nil result
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ParseFilter(tt.filter)
			if (err != nil) != tt.wantError {
				t.Errorf("ParseFilter() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantType == -1 {
				if node != nil {
					t.Errorf("ParseFilter() expected nil, got %v", node)
				}
				return
			}

			if node == nil {
				t.Errorf("ParseFilter() returned nil, want type %v", tt.wantType)
				return
			}

			if node.Type() != tt.wantType {
				t.Errorf("ParseFilter() type = %v, want %v", node.Type(), tt.wantType)
			}
		})
	}
}

func TestComparisonNodeValues(t *testing.T) {
	tests := []struct {
		name     string
		filter   string
		wantAttr string
		wantOp   Operator
		wantVal  any
	}{
		{
			name:     "string value",
			filter:   `userName eq "john"`,
			wantAttr: "userName",
			wantOp:   OpEqual,
			wantVal:  "john",
		},
		{
			name:     "boolean true",
			filter:   `active eq true`,
			wantAttr: "active",
			wantOp:   OpEqual,
			wantVal:  true,
		},
		{
			name:     "boolean false",
			filter:   `active eq false`,
			wantAttr: "active",
			wantOp:   OpEqual,
			wantVal:  false,
		},
		{
			name:     "null value",
			filter:   `title eq null`,
			wantAttr: "title",
			wantOp:   OpEqual,
			wantVal:  nil,
		},
		{
			name:     "nested attribute",
			filter:   `name.givenName eq "John"`,
			wantAttr: "name.givenName",
			wantOp:   OpEqual,
			wantVal:  "John",
		},
		{
			name:     "present operator",
			filter:   `title pr`,
			wantAttr: "title",
			wantOp:   OpPresent,
			wantVal:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ParseFilter(tt.filter)
			if err != nil {
				t.Fatalf("ParseFilter() error = %v", err)
			}

			comp, ok := node.(*ComparisonNode)
			if !ok {
				t.Fatalf("expected ComparisonNode, got %T", node)
			}

			if comp.AttributePath != tt.wantAttr {
				t.Errorf("AttributePath = %q, want %q", comp.AttributePath, tt.wantAttr)
			}
			if comp.Operator != tt.wantOp {
				t.Errorf("Operator = %q, want %q", comp.Operator, tt.wantOp)
			}

			// For present operator, value should be nil
			if tt.wantOp == OpPresent {
				return
			}

			if comp.Value != tt.wantVal {
				t.Errorf("Value = %v, want %v", comp.Value, tt.wantVal)
			}
		})
	}
}

func TestParseAttributePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantAttr  string
		wantSub   string
		wantURI   string
		wantError bool
	}{
		{
			name:     "simple attribute",
			path:     "userName",
			wantAttr: "userName",
		},
		{
			name:     "nested attribute",
			path:     "name.givenName",
			wantAttr: "name",
			wantSub:  "givenName",
		},
		{
			name:     "with URI prefix",
			path:     "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:employeeNumber",
			wantURI:  "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User",
			wantAttr: "employeeNumber",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ap, err := ParseAttributePath(tt.path)
			if (err != nil) != tt.wantError {
				t.Errorf("ParseAttributePath() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if ap.AttributeName != tt.wantAttr {
				t.Errorf("AttributeName = %q, want %q", ap.AttributeName, tt.wantAttr)
			}
			if ap.SubAttribute != tt.wantSub {
				t.Errorf("SubAttribute = %q, want %q", ap.SubAttribute, tt.wantSub)
			}
			if ap.URIPrefix != tt.wantURI {
				t.Errorf("URIPrefix = %q, want %q", ap.URIPrefix, tt.wantURI)
			}
		})
	}
}

func TestFilterEvaluation(t *testing.T) {
	type testUser struct {
		UserName    string `json:"userName"`
		DisplayName string `json:"displayName"`
		Active      bool   `json:"active"`
		Title       string `json:"title"`
	}

	user := testUser{
		UserName:    "bjensen",
		DisplayName: "Barbara Jensen",
		Active:      true,
		Title:       "Manager",
	}

	tests := []struct {
		name   string
		filter string
		want   bool
	}{
		{
			name:   "matching equality",
			filter: `userName eq "bjensen"`,
			want:   true,
		},
		{
			name:   "non-matching equality",
			filter: `userName eq "jsmith"`,
			want:   false,
		},
		{
			name:   "case insensitive match",
			filter: `userName eq "BJENSEN"`,
			want:   true,
		},
		{
			name:   "not equal",
			filter: `userName ne "jsmith"`,
			want:   true,
		},
		{
			name:   "starts with",
			filter: `displayName sw "Barbara"`,
			want:   true,
		},
		{
			name:   "ends with",
			filter: `displayName ew "Jensen"`,
			want:   true,
		},
		{
			name:   "contains",
			filter: `displayName co "bara"`,
			want:   true,
		},
		{
			name:   "boolean equality",
			filter: `active eq true`,
			want:   true,
		},
		{
			name:   "present",
			filter: `title pr`,
			want:   true,
		},
		{
			name:   "not present",
			filter: `nickname pr`,
			want:   false,
		},
		{
			name:   "logical and",
			filter: `userName eq "bjensen" and active eq true`,
			want:   true,
		},
		{
			name:   "logical and false",
			filter: `userName eq "bjensen" and active eq false`,
			want:   false,
		},
		{
			name:   "logical or",
			filter: `userName eq "jsmith" or userName eq "bjensen"`,
			want:   true,
		},
		{
			name:   "logical not",
			filter: `not (active eq false)`,
			want:   true,
		},
	}

	evaluator := DefaultEvaluator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ParseFilter(tt.filter)
			if err != nil {
				t.Fatalf("ParseFilter() error = %v", err)
			}

			got := evaluator.Evaluate(node, user)
			if got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterResources(t *testing.T) {
	type user struct {
		UserName string `json:"userName"`
		Active   bool   `json:"active"`
	}

	users := []user{
		{UserName: "alice", Active: true},
		{UserName: "bob", Active: true},
		{UserName: "charlie", Active: false},
	}

	filtered, err := FilterResources(users, `active eq true`)
	if err != nil {
		t.Fatalf("FilterResources() error = %v", err)
	}

	if len(filtered) != 2 {
		t.Errorf("FilterResources() returned %d users, want 2", len(filtered))
	}

	for _, u := range filtered {
		if !u.Active {
			t.Errorf("FilterResources() returned inactive user: %s", u.UserName)
		}
	}
}
