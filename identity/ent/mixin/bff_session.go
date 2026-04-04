// Package mixin provides reusable Ent schema mixins for CoreForge entities.
package mixin

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
	"github.com/google/uuid"
)

// BFFSession provides common fields for BFF session entities.
// Apps use this mixin in their Ent schema and implement EntClientInterface
// to connect CoreForge's EntStore to their generated Ent client.
//
// Example usage in app schema:
//
//	func (BFFSession) Mixin() []ent.Mixin {
//	    return []ent.Mixin{
//	        mixin.BFFSession{},
//	    }
//	}
type BFFSession struct {
	mixin.Schema
}

// Fields of the BFFSession mixin.
func (BFFSession) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			NotEmpty().
			Unique().
			Immutable().
			Comment("Unique session identifier"),

		field.UUID("user_id", uuid.UUID{}).
			Comment("Owner of this session"),

		field.UUID("organization_id", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("Organization context (optional)"),

		field.Bytes("access_token_encrypted").
			Sensitive().
			Comment("Encrypted access token"),

		field.Bytes("refresh_token_encrypted").
			Sensitive().
			Comment("Encrypted refresh token"),

		field.Time("access_token_expires_at").
			Comment("When the access token expires"),

		field.Time("refresh_token_expires_at").
			Comment("When the refresh token expires"),

		field.Bytes("dpop_key_pair_encrypted").
			Optional().
			Sensitive().
			Comment("Encrypted DPoP key pair (if DPoP enabled)"),

		field.String("dpop_thumbprint").
			Optional().
			MaxLen(64).
			Comment("DPoP JWK thumbprint"),

		field.String("ip_address").
			Optional().
			MaxLen(45).
			Comment("Client IP address"),

		field.String("user_agent").
			Optional().
			MaxLen(500).
			Comment("Client User-Agent"),

		field.JSON("metadata", map[string]string{}).
			Optional().
			Comment("Additional session metadata"),

		field.Time("last_accessed_at").
			Default(time.Now).
			Comment("When the session was last accessed"),

		field.Time("expires_at").
			Comment("When the session expires completely"),

		field.Time("created_at").
			Default(time.Now).
			Immutable(),

		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Indexes of the BFFSession mixin.
func (BFFSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("expires_at"),
		index.Fields("organization_id"),
	}
}
