package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents the CoreForge JWT claims structure.
// It embeds standard JWT claims and adds CoreForge-specific fields.
type Claims struct {
	jwt.RegisteredClaims

	// UserID is the CoreForge user ID.
	UserID uuid.UUID `json:"uid,omitempty"`

	// Email is the user's email address.
	Email string `json:"email,omitempty"`

	// Name is the user's display name.
	Name string `json:"name,omitempty"`

	// OrganizationID is the current organization context (optional).
	OrganizationID *uuid.UUID `json:"org_id,omitempty"`

	// OrganizationSlug is the current organization slug (optional).
	OrganizationSlug string `json:"org_slug,omitempty"`

	// Role is the user's role in the current organization (optional).
	Role string `json:"role,omitempty"`

	// Permissions are fine-grained permissions (optional).
	Permissions []string `json:"perms,omitempty"`

	// IsPlatformAdmin indicates cross-organization admin access.
	IsPlatformAdmin bool `json:"platform_admin,omitempty"`

	// TokenType distinguishes access tokens from refresh tokens.
	TokenType TokenType `json:"type,omitempty"`

	// TokenFamily is used for refresh token rotation tracking.
	TokenFamily string `json:"family,omitempty"`

	// Confirmation contains the DPoP binding (cnf claim with jkt).
	// When set, the token is bound to a specific DPoP key pair.
	Confirmation *CNFClaim `json:"cnf,omitempty"`
}

// TokenType identifies the type of token.
type TokenType string

const (
	// AccessToken is a short-lived token for API access.
	AccessToken TokenType = "access"
	// RefreshToken is a long-lived token for obtaining new access tokens.
	RefreshToken TokenType = "refresh"
)

// NewAccessClaims creates claims for a new access token.
func NewAccessClaims(cfg *Config, userID uuid.UUID, email, name string) *Claims {
	now := time.Now()
	return &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Audience:  cfg.Audience,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.AccessTokenExpiry)),
			ID:        uuid.NewString(),
		},
		UserID:    userID,
		Email:     email,
		Name:      name,
		TokenType: AccessToken,
	}
}

// NewRefreshClaims creates claims for a new refresh token.
func NewRefreshClaims(cfg *Config, userID uuid.UUID, family string) *Claims {
	now := time.Now()
	return &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Audience:  cfg.Audience,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.RefreshTokenExpiry)),
			ID:        uuid.NewString(),
		},
		UserID:      userID,
		TokenType:   RefreshToken,
		TokenFamily: family,
	}
}

// WithOrganization adds organization context to the claims.
func (c *Claims) WithOrganization(orgID uuid.UUID, slug, role string, permissions []string) *Claims {
	c.OrganizationID = &orgID
	c.OrganizationSlug = slug
	c.Role = role
	c.Permissions = permissions
	return c
}

// WithPlatformAdmin marks the user as a platform administrator.
func (c *Claims) WithPlatformAdmin(isPlatformAdmin bool) *Claims {
	c.IsPlatformAdmin = isPlatformAdmin
	return c
}

// IsExpired returns true if the token has expired.
func (c *Claims) IsExpired() bool {
	if c.ExpiresAt == nil {
		return true
	}
	return c.ExpiresAt.Before(time.Now())
}

// IsAccessToken returns true if this is an access token.
func (c *Claims) IsAccessToken() bool {
	return c.TokenType == AccessToken
}

// IsRefreshToken returns true if this is a refresh token.
func (c *Claims) IsRefreshToken() bool {
	return c.TokenType == RefreshToken
}
