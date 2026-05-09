package mapper

import (
	"context"

	"github.com/grokify/systemforge/identity/scim"
)

// EnterpriseExtension contains enterprise user extension data.
// This is stored as metadata on the user in SystemForge.
type EnterpriseExtension struct {
	EmployeeNumber string `json:"employee_number,omitempty"`
	CostCenter     string `json:"cost_center,omitempty"`
	Organization   string `json:"organization,omitempty"`
	Division       string `json:"division,omitempty"`
	Department     string `json:"department,omitempty"`
	ManagerID      string `json:"manager_id,omitempty"`
}

// ExtensionMapper maps enterprise extension data.
type ExtensionMapper struct {
	config *Config
}

// NewExtensionMapper creates a new extension mapper.
func NewExtensionMapper(config *Config) *ExtensionMapper {
	if config == nil {
		config = DefaultConfig()
	}
	return &ExtensionMapper{config: config}
}

// ToSCIM converts enterprise extension metadata to SCIM EnterpriseUser.
func (m *ExtensionMapper) ToSCIM(ctx context.Context, ext *EnterpriseExtension) *scim.EnterpriseUser {
	if ext == nil {
		return nil
	}

	enterprise := &scim.EnterpriseUser{
		EmployeeNumber: ext.EmployeeNumber,
		CostCenter:     ext.CostCenter,
		Organization:   ext.Organization,
		Division:       ext.Division,
		Department:     ext.Department,
	}

	// Add manager reference if present
	if ext.ManagerID != "" {
		enterprise.Manager = &scim.ManagerRef{
			Value: ext.ManagerID,
			Ref:   m.config.BaseURL + "/Users/" + ext.ManagerID,
		}
	}

	return enterprise
}

// FromSCIM converts SCIM EnterpriseUser to enterprise extension metadata.
func (m *ExtensionMapper) FromSCIM(ctx context.Context, enterprise *scim.EnterpriseUser) *EnterpriseExtension {
	if enterprise == nil {
		return nil
	}

	ext := &EnterpriseExtension{
		EmployeeNumber: enterprise.EmployeeNumber,
		CostCenter:     enterprise.CostCenter,
		Organization:   enterprise.Organization,
		Division:       enterprise.Division,
		Department:     enterprise.Department,
	}

	if enterprise.Manager != nil {
		ext.ManagerID = enterprise.Manager.Value
	}

	return ext
}

// ToMetadata converts enterprise extension to a metadata map.
func (m *ExtensionMapper) ToMetadata(ext *EnterpriseExtension) map[string]any {
	if ext == nil {
		return nil
	}

	metadata := make(map[string]any)

	if ext.EmployeeNumber != "" {
		metadata["employee_number"] = ext.EmployeeNumber
	}
	if ext.CostCenter != "" {
		metadata["cost_center"] = ext.CostCenter
	}
	if ext.Organization != "" {
		metadata["organization"] = ext.Organization
	}
	if ext.Division != "" {
		metadata["division"] = ext.Division
	}
	if ext.Department != "" {
		metadata["department"] = ext.Department
	}
	if ext.ManagerID != "" {
		metadata["manager_id"] = ext.ManagerID
	}

	return metadata
}

// FromMetadata converts a metadata map to enterprise extension.
func (m *ExtensionMapper) FromMetadata(metadata map[string]any) *EnterpriseExtension {
	if metadata == nil {
		return nil
	}

	ext := &EnterpriseExtension{}

	if v, ok := metadata["employee_number"].(string); ok {
		ext.EmployeeNumber = v
	}
	if v, ok := metadata["cost_center"].(string); ok {
		ext.CostCenter = v
	}
	if v, ok := metadata["organization"].(string); ok {
		ext.Organization = v
	}
	if v, ok := metadata["division"].(string); ok {
		ext.Division = v
	}
	if v, ok := metadata["department"].(string); ok {
		ext.Department = v
	}
	if v, ok := metadata["manager_id"].(string); ok {
		ext.ManagerID = v
	}

	return ext
}

// IsEmpty returns true if the extension has no data.
func (ext *EnterpriseExtension) IsEmpty() bool {
	if ext == nil {
		return true
	}
	return ext.EmployeeNumber == "" &&
		ext.CostCenter == "" &&
		ext.Organization == "" &&
		ext.Division == "" &&
		ext.Department == "" &&
		ext.ManagerID == ""
}
