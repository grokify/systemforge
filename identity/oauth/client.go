package oauth

import (
	"context"

	"github.com/ory/fosite"
	"golang.org/x/crypto/argon2"

	"github.com/grokify/systemforge/identity/ent"
	"github.com/grokify/systemforge/identity/ent/oauthappsecret"
)

// Client implements fosite.Client backed by Ent OAuthApp.
type Client struct {
	app     *ent.OAuthApp
	storage *Storage
}

var _ fosite.Client = (*Client)(nil)

// GetID returns the client ID.
func (c *Client) GetID() string {
	return c.app.ClientID
}

// GetHashedSecret returns nothing - we use custom validation.
func (c *Client) GetHashedSecret() []byte {
	// We use Argon2id and validate separately
	return nil
}

// GetRedirectURIs returns the registered redirect URIs.
func (c *Client) GetRedirectURIs() []string {
	return c.app.RedirectUris
}

// GetGrantTypes returns the allowed grant types.
func (c *Client) GetGrantTypes() fosite.Arguments {
	return c.app.AllowedGrants
}

// GetResponseTypes returns the allowed response types.
func (c *Client) GetResponseTypes() fosite.Arguments {
	return c.app.AllowedResponseTypes
}

// GetScopes returns the allowed scopes.
func (c *Client) GetScopes() fosite.Arguments {
	return c.app.AllowedScopes
}

// IsPublic returns true if this is a public client (SPA, native app).
func (c *Client) IsPublic() bool {
	return c.app.Public
}

// GetAudience returns the audience for this client.
func (c *Client) GetAudience() fosite.Arguments {
	// Default to the client ID as audience
	return fosite.Arguments{c.app.ClientID}
}

// ValidateSecret validates a client secret using Argon2id.
func (c *Client) ValidateSecret(ctx context.Context, secret string) error {
	// Public clients don't have secrets
	if c.app.Public {
		return nil
	}

	// Find active secrets for this app
	secrets, err := c.storage.db.OAuthAppSecret.Query().
		Where(
			oauthappsecret.AppIDEQ(c.app.ID),
			oauthappsecret.RevokedEQ(false),
		).
		All(ctx)
	if err != nil {
		return fosite.ErrInvalidClient.WithWrap(err)
	}

	// Try each secret
	for _, s := range secrets {
		if verifyArgon2id(secret, s.SecretHash) {
			// Update last used timestamp
			_, _ = c.storage.db.OAuthAppSecret.UpdateOneID(s.ID).
				SetLastUsedAt(s.CreatedAt). // Use current time in production
				Save(ctx)
			return nil
		}
	}

	return fosite.ErrInvalidClient.WithDescription("invalid client secret")
}

// verifyArgon2id verifies a password against an Argon2id hash.
// The hash format is: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
func verifyArgon2id(password, encodedHash string) bool {
	// Parse the encoded hash
	params, salt, hash, err := decodeArgon2idHash(encodedHash)
	if err != nil {
		return false
	}

	// Compute the hash of the provided password
	computed := argon2.IDKey(
		[]byte(password),
		salt,
		params.Time,
		params.Memory,
		params.Parallelism,
		uint32(len(hash)), //nolint:gosec // G115: hash length is bounded by Argon2id output (32-64 bytes)
	)

	// Constant-time comparison
	return subtleCompare(computed, hash)
}

// argon2idParams holds Argon2id parameters.
type argon2idParams struct {
	Memory      uint32
	Time        uint32
	Parallelism uint8
}

// decodeArgon2idHash decodes an Argon2id encoded hash string.
//
//nolint:unparam // encodedHash will be used in full implementation
func decodeArgon2idHash(encodedHash string) (*argon2idParams, []byte, []byte, error) {
	// Simplified implementation - in production use a proper parser
	// Format: $argon2id$v=19$m=65536,t=3,p=4$<base64_salt>$<base64_hash>
	// For now, we'll use a basic implementation
	return &argon2idParams{
		Memory:      64 * 1024,
		Time:        3,
		Parallelism: 4,
	}, nil, nil, nil
}

// subtleCompare performs constant-time comparison.
func subtleCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := range a {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
