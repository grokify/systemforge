package jwt

import (
	"crypto/sha256"
	"encoding/base64"
)

// CNFClaim represents the confirmation claim (cnf) for DPoP-bound tokens.
// Per RFC 9449, this contains the JWK thumbprint of the DPoP public key.
type CNFClaim struct {
	// JKT is the JWK thumbprint of the DPoP public key (RFC 7638).
	JKT string `json:"jkt,omitempty"`
}

// DPoPBinding contains DPoP binding information for a token.
type DPoPBinding struct {
	// Thumbprint is the JWK thumbprint of the DPoP key pair.
	Thumbprint string
}

// WithDPoPBinding adds DPoP binding to access token claims.
// The thumbprint should be computed using RFC 7638 (JWK thumbprint).
func (c *Claims) WithDPoPBinding(thumbprint string) *Claims {
	c.Confirmation = &CNFClaim{
		JKT: thumbprint,
	}
	return c
}

// IsDPoPBound returns true if this token is bound to a DPoP key.
func (c *Claims) IsDPoPBound() bool {
	return c.Confirmation != nil && c.Confirmation.JKT != ""
}

// DPoPThumbprint returns the DPoP JWK thumbprint if the token is DPoP-bound.
// Returns empty string if not bound.
func (c *Claims) DPoPThumbprint() string {
	if c.Confirmation == nil {
		return ""
	}
	return c.Confirmation.JKT
}

// ComputeTokenHash computes the base64url-encoded SHA-256 hash of a token string.
// This is used for the ath (access token hash) claim in DPoP proofs.
func ComputeTokenHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
