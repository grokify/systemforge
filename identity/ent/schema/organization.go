package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Organization holds the schema definition for the Organization entity.
type Organization struct {
	ent.Schema
}

// Annotations of the Organization.
func (Organization) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "cf_organizations"},
	}
}

// Mixin of the Organization.
func (Organization) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Fields of the Organization.
func (Organization) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty(),
		field.String("slug").
			NotEmpty().
			Unique().
			Comment("URL-safe identifier"),
		field.String("logo_url").
			Optional().
			Nillable(),
		field.JSON("settings", map[string]any{}).
			Optional().
			Comment("App-specific configuration"),
		field.Enum("plan").
			Values("free", "starter", "pro", "enterprise").
			Default("free"),
		field.Bool("active").
			Default(true),
	}
}

// Edges of the Organization.
func (Organization) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("memberships", Membership.Type),
		edge.To("api_keys", APIKey.Type),

		// OAuth 2.0 edges
		edge.To("oauth_apps", OAuthApp.Type),
		edge.To("service_accounts", ServiceAccount.Type),
	}
}

// Indexes of the Organization.
func (Organization) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("slug").
			Unique(),
		index.Fields("active"),
	}
}
