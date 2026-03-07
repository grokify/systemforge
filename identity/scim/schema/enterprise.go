package schema

// EnterpriseUserSchema returns the SCIM Enterprise User extension schema definition.
func EnterpriseUserSchema() Schema {
	return Schema{
		ID:          URIEnterpriseUser,
		Name:        "Enterprise User",
		Description: "Enterprise User Extension",
		Attributes:  enterpriseUserAttributes(),
		Meta: &SchemaMeta{
			ResourceType: "Schema",
		},
	}
}

// enterpriseUserAttributes returns the attribute definitions for the Enterprise User extension.
func enterpriseUserAttributes() []Attribute {
	return []Attribute{
		{
			Name:        "employeeNumber",
			Type:        TypeString,
			MultiValued: false,
			Description: "Numeric or alphanumeric identifier assigned to a person.",
			Required:    false,
			CaseExact:   false,
			Mutability:  MutabilityReadWrite,
			Returned:    ReturnedDefault,
			Uniqueness:  UniquenessNone,
		},
		{
			Name:        "costCenter",
			Type:        TypeString,
			MultiValued: false,
			Description: "Identifies the name of a cost center.",
			Required:    false,
			CaseExact:   false,
			Mutability:  MutabilityReadWrite,
			Returned:    ReturnedDefault,
			Uniqueness:  UniquenessNone,
		},
		{
			Name:        "organization",
			Type:        TypeString,
			MultiValued: false,
			Description: "Identifies the name of an organization.",
			Required:    false,
			CaseExact:   false,
			Mutability:  MutabilityReadWrite,
			Returned:    ReturnedDefault,
			Uniqueness:  UniquenessNone,
		},
		{
			Name:        "division",
			Type:        TypeString,
			MultiValued: false,
			Description: "Identifies the name of a division.",
			Required:    false,
			CaseExact:   false,
			Mutability:  MutabilityReadWrite,
			Returned:    ReturnedDefault,
			Uniqueness:  UniquenessNone,
		},
		{
			Name:        "department",
			Type:        TypeString,
			MultiValued: false,
			Description: "Identifies the name of a department.",
			Required:    false,
			CaseExact:   false,
			Mutability:  MutabilityReadWrite,
			Returned:    ReturnedDefault,
			Uniqueness:  UniquenessNone,
		},
		{
			Name:        "manager",
			Type:        TypeComplex,
			MultiValued: false,
			Description: "The User's manager.",
			Required:    false,
			Mutability:  MutabilityReadWrite,
			Returned:    ReturnedDefault,
			Uniqueness:  UniquenessNone,
			SubAttributes: []Attribute{
				{
					Name:        "value",
					Type:        TypeString,
					MultiValued: false,
					Description: "The id of the SCIM resource representing the User's manager.",
					Required:    false,
					Mutability:  MutabilityReadWrite,
					Returned:    ReturnedDefault,
					Uniqueness:  UniquenessNone,
				},
				{
					Name:           "$ref",
					Type:           TypeReference,
					MultiValued:    false,
					Description:    "The URI of the SCIM resource representing the User's manager.",
					Required:       false,
					Mutability:     MutabilityReadWrite,
					Returned:       ReturnedDefault,
					Uniqueness:     UniquenessNone,
					ReferenceTypes: []string{"User"},
				},
				{
					Name:        "displayName",
					Type:        TypeString,
					MultiValued: false,
					Description: "The displayName of the User's manager.",
					Required:    false,
					Mutability:  MutabilityReadOnly,
					Returned:    ReturnedDefault,
					Uniqueness:  UniquenessNone,
				},
			},
		},
	}
}
