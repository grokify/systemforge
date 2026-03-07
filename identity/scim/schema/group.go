package schema

// GroupSchema returns the SCIM Group schema definition.
func GroupSchema() Schema {
	return Schema{
		ID:          URIGroup,
		Name:        "Group",
		Description: "Group",
		Attributes:  groupAttributes(),
		Meta: &SchemaMeta{
			ResourceType: "Schema",
		},
	}
}

// groupAttributes returns the attribute definitions for the Group schema.
func groupAttributes() []Attribute {
	return []Attribute{
		{
			Name:        "displayName",
			Type:        TypeString,
			MultiValued: false,
			Description: "A human-readable name for the Group.",
			Required:    true,
			CaseExact:   false,
			Mutability:  MutabilityReadWrite,
			Returned:    ReturnedDefault,
			Uniqueness:  UniquenessNone,
		},
		{
			Name:        "members",
			Type:        TypeComplex,
			MultiValued: true,
			Description: "A list of members of the Group.",
			Required:    false,
			Mutability:  MutabilityReadWrite,
			Returned:    ReturnedDefault,
			Uniqueness:  UniquenessNone,
			SubAttributes: []Attribute{
				{
					Name:        "value",
					Type:        TypeString,
					MultiValued: false,
					Description: "Identifier of the member of this Group.",
					Required:    false,
					Mutability:  MutabilityImmutable,
					Returned:    ReturnedDefault,
					Uniqueness:  UniquenessNone,
				},
				{
					Name:           "$ref",
					Type:           TypeReference,
					MultiValued:    false,
					Description:    "The URI corresponding to a SCIM resource that is a member of this Group.",
					Required:       false,
					Mutability:     MutabilityImmutable,
					Returned:       ReturnedDefault,
					Uniqueness:     UniquenessNone,
					ReferenceTypes: []string{"User", "Group"},
				},
				{
					Name:        "display",
					Type:        TypeString,
					MultiValued: false,
					Description: "A human-readable name for the member.",
					Required:    false,
					Mutability:  MutabilityReadOnly,
					Returned:    ReturnedDefault,
					Uniqueness:  UniquenessNone,
				},
				{
					Name:            "type",
					Type:            TypeString,
					MultiValued:     false,
					Description:     "A label indicating the type of resource.",
					Required:        false,
					Mutability:      MutabilityImmutable,
					Returned:        ReturnedDefault,
					Uniqueness:      UniquenessNone,
					CanonicalValues: []string{"User", "Group"},
				},
			},
		},
	}
}

// GroupResourceType returns the SCIM Group resource type definition.
func GroupResourceType(baseURL string) ResourceType {
	return ResourceType{
		ID:          "Group",
		Name:        "Group",
		Description: "Group",
		Endpoint:    "/Groups",
		Schema:      URIGroup,
		Meta: &ResourceTypeMeta{
			ResourceType: "ResourceType",
			Location:     baseURL + "/ResourceTypes/Group",
		},
	}
}
