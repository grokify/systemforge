package scim

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUserJSONMarshal(t *testing.T) {
	active := true
	created := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	modified := time.Date(2024, 2, 20, 14, 45, 0, 0, time.UTC)

	user := &User{
		Resource: Resource{
			Schemas:    []string{SchemaUser},
			ID:         "2819c223-7f76-453a-919d-413861904646",
			ExternalID: "ext-123",
			Meta: &Meta{
				ResourceType: ResourceTypeUser,
				Created:      &created,
				LastModified: &modified,
				Location:     "https://example.com/scim/v2/Users/2819c223-7f76-453a-919d-413861904646",
				Version:      "W/\"a330bc54f0671c9\"",
			},
		},
		UserName:    "bjensen@example.com",
		DisplayName: "Barbara Jensen",
		Name: &Name{
			Formatted:  "Ms. Barbara J Jensen III",
			FamilyName: "Jensen",
			GivenName:  "Barbara",
		},
		Active: &active,
		Emails: []MultiValue{
			{Value: "bjensen@example.com", Type: "work", Primary: true},
		},
	}

	data, err := json.Marshal(user) //nolint:gosec // G117: Testing SCIM user serialization
	if err != nil {
		t.Fatalf("failed to marshal user: %v", err)
	}

	// Unmarshal and verify
	var decoded User
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal user: %v", err)
	}

	if decoded.UserName != user.UserName {
		t.Errorf("userName mismatch: got %q, want %q", decoded.UserName, user.UserName)
	}
	if decoded.DisplayName != user.DisplayName {
		t.Errorf("displayName mismatch: got %q, want %q", decoded.DisplayName, user.DisplayName)
	}
	if decoded.Name == nil {
		t.Error("name is nil")
	} else if decoded.Name.Formatted != user.Name.Formatted {
		t.Errorf("name.formatted mismatch: got %q, want %q", decoded.Name.Formatted, user.Name.Formatted)
	}
	if decoded.Active == nil || *decoded.Active != *user.Active {
		t.Error("active mismatch")
	}
	if len(decoded.Emails) != 1 {
		t.Errorf("emails length mismatch: got %d, want 1", len(decoded.Emails))
	}
}

func TestGroupJSONMarshal(t *testing.T) {
	group := &Group{
		Resource: Resource{
			Schemas: []string{SchemaGroup},
			ID:      "e9e30dba-f08f-4109-8486-d5c6a331660a",
		},
		DisplayName: "Engineering",
		Members: []MemberRef{
			{Value: "user-1", Ref: "https://example.com/scim/v2/Users/user-1", Display: "User One", Type: "User"},
			{Value: "user-2", Ref: "https://example.com/scim/v2/Users/user-2", Display: "User Two", Type: "User"},
		},
	}

	data, err := json.Marshal(group)
	if err != nil {
		t.Fatalf("failed to marshal group: %v", err)
	}

	var decoded Group
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal group: %v", err)
	}

	if decoded.DisplayName != group.DisplayName {
		t.Errorf("displayName mismatch: got %q, want %q", decoded.DisplayName, group.DisplayName)
	}
	if len(decoded.Members) != 2 {
		t.Errorf("members length mismatch: got %d, want 2", len(decoded.Members))
	}
}

func TestListResponse(t *testing.T) {
	users := []any{
		&User{
			Resource: Resource{
				Schemas: []string{SchemaUser},
				ID:      "user-1",
			},
			UserName: "user1@example.com",
		},
		&User{
			Resource: Resource{
				Schemas: []string{SchemaUser},
				ID:      "user-2",
			},
			UserName: "user2@example.com",
		},
	}

	response := NewListResponse(users, 100, 1, 2)

	if response.TotalResults != 100 {
		t.Errorf("totalResults mismatch: got %d, want 100", response.TotalResults)
	}
	if response.StartIndex != 1 {
		t.Errorf("startIndex mismatch: got %d, want 1", response.StartIndex)
	}
	if response.ItemsPerPage != 2 {
		t.Errorf("itemsPerPage mismatch: got %d, want 2", response.ItemsPerPage)
	}
	if len(response.Resources) != 2 {
		t.Errorf("resources length mismatch: got %d, want 2", len(response.Resources))
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal list response: %v", err)
	}

	var decoded ListResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal list response: %v", err)
	}

	if decoded.TotalResults != response.TotalResults {
		t.Errorf("totalResults mismatch after roundtrip")
	}
}

func TestPatchRequest(t *testing.T) {
	patch := PatchRequest{
		Schemas: []string{SchemaPatchOp},
		Operations: []PatchOperation{
			{Op: "replace", Path: "displayName", Value: "New Name"},
			{Op: "add", Path: "emails", Value: []map[string]any{{"value": "new@example.com", "type": "work"}}},
			{Op: "remove", Path: "phoneNumbers[type eq \"fax\"]"},
		},
	}

	data, err := json.Marshal(patch)
	if err != nil {
		t.Fatalf("failed to marshal patch: %v", err)
	}

	var decoded PatchRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal patch: %v", err)
	}

	if len(decoded.Operations) != 3 {
		t.Errorf("operations length mismatch: got %d, want 3", len(decoded.Operations))
	}
	if decoded.Operations[0].Op != "replace" {
		t.Errorf("first operation op mismatch: got %q, want %q", decoded.Operations[0].Op, "replace")
	}
}

func TestEnterpriseUserExtension(t *testing.T) {
	user := &User{
		Resource: Resource{
			Schemas: []string{SchemaUser, SchemaEnterpriseUser},
			ID:      "user-123",
		},
		UserName: "jsmith@example.com",
		EnterpriseUser: &EnterpriseUser{
			EmployeeNumber: "EMP-001",
			CostCenter:     "CC-123",
			Organization:   "Acme Corp",
			Division:       "Engineering",
			Department:     "Platform",
			Manager: &ManagerRef{
				Value:       "manager-456",
				Ref:         "https://example.com/scim/v2/Users/manager-456",
				DisplayName: "Jane Doe",
			},
		},
	}

	data, err := json.Marshal(user) //nolint:gosec // G117: Testing SCIM user serialization
	if err != nil {
		t.Fatalf("failed to marshal user with enterprise extension: %v", err)
	}

	var decoded User
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal user with enterprise extension: %v", err)
	}

	if decoded.EnterpriseUser == nil {
		t.Fatal("enterprise user extension is nil after unmarshal")
	}
	if decoded.EnterpriseUser.EmployeeNumber != "EMP-001" {
		t.Errorf("employeeNumber mismatch: got %q, want %q", decoded.EnterpriseUser.EmployeeNumber, "EMP-001")
	}
	if decoded.EnterpriseUser.Manager == nil {
		t.Error("manager is nil")
	} else if decoded.EnterpriseUser.Manager.DisplayName != "Jane Doe" {
		t.Errorf("manager displayName mismatch: got %q, want %q", decoded.EnterpriseUser.Manager.DisplayName, "Jane Doe")
	}
}
