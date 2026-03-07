package token

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/grokify/coreforge/identity/ent"
	"github.com/grokify/coreforge/identity/ent/principaltoken"
	"github.com/grokify/coreforge/identity/principal"
)

const (
	// TokenBytes is the number of random bytes for token generation.
	TokenBytes = 32
)

// DefaultService implements the Service interface.
type DefaultService struct {
	client *ent.Client
}

// NewService creates a new TokenService.
func NewService(client *ent.Client) Service {
	return &DefaultService{client: client}
}

// Issue issues a new token pair.
func (s *DefaultService) Issue(ctx context.Context, input IssueInput) (*IssuedToken, error) {
	// Generate access token
	accessToken, accessSig, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token (if TTL > 0)
	var refreshToken, refreshSig string
	var refreshExpiresAt *time.Time
	if input.RefreshTTL > 0 {
		refreshToken, refreshSig, err = generateToken()
		if err != nil {
			return nil, fmt.Errorf("failed to generate refresh token: %w", err)
		}
		t := time.Now().Add(input.RefreshTTL)
		refreshExpiresAt = &t
	}

	// Set default access TTL
	accessTTL := input.AccessTTL
	if accessTTL == 0 {
		accessTTL = 15 * time.Minute
	}
	accessExpiresAt := time.Now().Add(accessTTL)

	// Generate family ID
	familyID := uuid.New()

	// Create token record
	create := s.client.PrincipalToken.Create().
		SetPrincipalID(input.PrincipalID).
		SetPrincipalType(mapPrincipalType(input.PrincipalType)).
		SetAccessTokenSignature(accessSig).
		SetFamilyID(familyID).
		SetScopes(input.Scopes).
		SetAudience(input.Audience).
		SetCapabilities(input.Capabilities).
		SetDelegationChain(input.DelegationChain).
		SetAccessExpiresAt(accessExpiresAt)

	if input.IssuedByAppID != nil {
		create.SetIssuedByAppID(*input.IssuedByAppID)
	}
	if refreshSig != "" {
		create.SetRefreshTokenSignature(refreshSig)
	}
	if refreshExpiresAt != nil {
		create.SetRefreshExpiresAt(*refreshExpiresAt)
	}
	if input.DPoPJKT != "" {
		create.SetDpopJkt(input.DPoPJKT)
	}
	if input.SessionID != "" {
		create.SetSessionID(input.SessionID)
	}
	if input.ClientIP != "" {
		create.SetClientIP(input.ClientIP)
	}
	if input.UserAgent != "" {
		create.SetUserAgent(input.UserAgent)
	}
	if input.ParentTokenID != nil {
		create.SetParentTokenID(*input.ParentTokenID)
	}

	token, err := create.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create token: %w", err)
	}

	return &IssuedToken{
		Token:        entTokenToModel(token),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(accessTTL.Seconds()),
	}, nil
}

// Refresh refreshes a token using the refresh token.
func (s *DefaultService) Refresh(ctx context.Context, input RefreshInput) (*IssuedToken, error) {
	// Hash the refresh token to find it
	hash := sha256.Sum256([]byte(input.RefreshToken))
	refreshSig := hex.EncodeToString(hash[:])

	// Find the token
	oldToken, err := s.client.PrincipalToken.Query().
		Where(
			principaltoken.RefreshTokenSignatureEQ(refreshSig),
			principaltoken.RevokedEQ(false),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("invalid refresh token")
		}
		return nil, fmt.Errorf("failed to find token: %w", err)
	}

	// Check expiration
	if oldToken.RefreshExpiresAt != nil && oldToken.RefreshExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("refresh token expired")
	}

	// Check DPoP binding
	if oldToken.DpopJkt != "" && oldToken.DpopJkt != input.DPoPJKT {
		return nil, fmt.Errorf("DPoP binding mismatch")
	}

	// Determine scopes (use narrowed scopes if provided, otherwise keep original)
	scopes := oldToken.Scopes
	if len(input.Scopes) > 0 {
		// Verify narrowed scopes are subset of original
		scopeSet := make(map[string]bool)
		for _, s := range oldToken.Scopes {
			scopeSet[s] = true
		}
		for _, s := range input.Scopes {
			if !scopeSet[s] {
				return nil, fmt.Errorf("scope %q not in original grant", s)
			}
		}
		scopes = input.Scopes
	}

	// Calculate new expiration times based on original TTLs
	accessTTL := time.Until(oldToken.AccessExpiresAt)
	if accessTTL < 5*time.Minute {
		accessTTL = 15 * time.Minute // Default if very short
	}
	var refreshTTL time.Duration
	if oldToken.RefreshExpiresAt != nil {
		refreshTTL = time.Until(*oldToken.RefreshExpiresAt)
		if refreshTTL < 5*time.Minute {
			refreshTTL = 7 * 24 * time.Hour // Default if very short
		}
	}

	// Issue new token pair
	newToken, err := s.Issue(ctx, IssueInput{
		PrincipalID:     oldToken.PrincipalID,
		PrincipalType:   principal.Type(oldToken.PrincipalType.String()),
		IssuedByAppID:   oldToken.IssuedByAppID,
		Scopes:          scopes,
		Audience:        oldToken.Audience,
		Capabilities:    oldToken.Capabilities,
		DelegationChain: oldToken.DelegationChain,
		DPoPJKT:         input.DPoPJKT,
		SessionID:       oldToken.SessionID,
		AccessTTL:       accessTTL,
		RefreshTTL:      refreshTTL,
		ClientIP:        input.ClientIP,
		UserAgent:       input.UserAgent,
		ParentTokenID:   oldToken.ParentTokenID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to issue new token: %w", err)
	}

	// Update new token to use same family ID (for rotation tracking)
	_, err = s.client.PrincipalToken.UpdateOneID(newToken.Token.ID).
		SetFamilyID(oldToken.FamilyID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update token family: %w", err)
	}
	newToken.Token.FamilyID = oldToken.FamilyID

	// Revoke old token
	now := time.Now()
	_, err = s.client.PrincipalToken.UpdateOne(oldToken).
		SetRevoked(true).
		SetRevokedAt(now).
		SetRevokedReason("rotated").
		Save(ctx)
	if err != nil {
		// Log but don't fail
		_ = err
	}

	return newToken, nil
}

// Validate validates an access token and returns the token record.
func (s *DefaultService) Validate(ctx context.Context, accessToken string) (*Token, error) {
	// Hash the access token to find it
	hash := sha256.Sum256([]byte(accessToken))
	accessSig := hex.EncodeToString(hash[:])

	// Find the token
	token, err := s.client.PrincipalToken.Query().
		Where(
			principaltoken.AccessTokenSignatureEQ(accessSig),
			principaltoken.RevokedEQ(false),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("invalid access token")
		}
		return nil, fmt.Errorf("failed to find token: %w", err)
	}

	// Check expiration
	if token.AccessExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("access token expired")
	}

	// Update last used
	now := time.Now()
	_, _ = s.client.PrincipalToken.UpdateOne(token).
		SetLastUsedAt(now).
		Save(ctx)

	return entTokenToModel(token), nil
}

// Revoke revokes a token.
func (s *DefaultService) Revoke(ctx context.Context, tokenID uuid.UUID, revokeFamily bool, reason string) error {
	now := time.Now()

	if revokeFamily {
		// Get the token to find its family ID
		token, err := s.client.PrincipalToken.Get(ctx, tokenID)
		if err != nil {
			return fmt.Errorf("failed to get token: %w", err)
		}

		// Revoke all tokens in the family
		_, err = s.client.PrincipalToken.Update().
			Where(principaltoken.FamilyIDEQ(token.FamilyID)).
			SetRevoked(true).
			SetRevokedAt(now).
			SetRevokedReason(reason).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to revoke token family: %w", err)
		}
	} else {
		// Revoke single token
		_, err := s.client.PrincipalToken.UpdateOneID(tokenID).
			SetRevoked(true).
			SetRevokedAt(now).
			SetRevokedReason(reason).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to revoke token: %w", err)
		}
	}

	return nil
}

// RevokeBySignature revokes a token by its access token signature.
func (s *DefaultService) RevokeBySignature(ctx context.Context, accessTokenSignature string, revokeFamily bool, reason string) error {
	token, err := s.client.PrincipalToken.Query().
		Where(principaltoken.AccessTokenSignatureEQ(accessTokenSignature)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("token not found")
		}
		return fmt.Errorf("failed to find token: %w", err)
	}

	return s.Revoke(ctx, token.ID, revokeFamily, reason)
}

// GetCapabilitiesForType returns the default capabilities for a principal type.
func (s *DefaultService) GetCapabilitiesForType(principalType principal.Type) Capabilities {
	caps := principal.DefaultCapabilitiesForType(principalType)
	return Capabilities{
		CanAccessUI:       caps.CanAccessUI,
		CanManageProfile:  caps.CanManageProfile,
		CanActOnBehalf:    caps.CanActOnBehalf,
		CanDelegate:       caps.CanDelegate,
		RequiresApproval:  caps.RequiresApproval,
		CanBypassRLS:      caps.CanBypassRLS,
		CanRequestOffline: caps.CanRequestOffline,
	}
}

// ListForPrincipal lists all active tokens for a principal.
func (s *DefaultService) ListForPrincipal(ctx context.Context, principalID uuid.UUID) ([]*Token, error) {
	tokens, err := s.client.PrincipalToken.Query().
		Where(
			principaltoken.PrincipalIDEQ(principalID),
			principaltoken.RevokedEQ(false),
			principaltoken.AccessExpiresAtGT(time.Now()),
		).
		Order(ent.Desc(principaltoken.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens: %w", err)
	}

	result := make([]*Token, len(tokens))
	for i, t := range tokens {
		result[i] = entTokenToModel(t)
	}
	return result, nil
}

// RevokeAllForPrincipal revokes all tokens for a principal.
func (s *DefaultService) RevokeAllForPrincipal(ctx context.Context, principalID uuid.UUID, reason string) error {
	now := time.Now()
	_, err := s.client.PrincipalToken.Update().
		Where(
			principaltoken.PrincipalIDEQ(principalID),
			principaltoken.RevokedEQ(false),
		).
		SetRevoked(true).
		SetRevokedAt(now).
		SetRevokedReason(reason).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to revoke tokens: %w", err)
	}
	return nil
}

// Helper functions

func generateToken() (plainToken, signature string, err error) {
	bytes := make([]byte, TokenBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	plainToken = base64.RawURLEncoding.EncodeToString(bytes)
	hash := sha256.Sum256([]byte(plainToken))
	signature = hex.EncodeToString(hash[:])
	return plainToken, signature, nil
}

func entTokenToModel(t *ent.PrincipalToken) *Token {
	return &Token{
		ID:               t.ID,
		PrincipalID:      t.PrincipalID,
		PrincipalType:    principal.Type(t.PrincipalType.String()),
		IssuedByAppID:    t.IssuedByAppID,
		FamilyID:         t.FamilyID,
		ParentTokenID:    t.ParentTokenID,
		Scopes:           t.Scopes,
		Audience:         t.Audience,
		Capabilities:     t.Capabilities,
		DelegationChain:  t.DelegationChain,
		DPoPJKT:          t.DpopJkt,
		SessionID:        t.SessionID,
		AccessExpiresAt:  t.AccessExpiresAt,
		RefreshExpiresAt: t.RefreshExpiresAt,
		Revoked:          t.Revoked,
		RevokedAt:        t.RevokedAt,
		RevokedReason:    t.RevokedReason,
		ClientIP:         t.ClientIP,
		UserAgent:        t.UserAgent,
		LastUsedAt:       t.LastUsedAt,
		CreatedAt:        t.CreatedAt,
	}
}

func mapPrincipalType(t principal.Type) principaltoken.PrincipalType {
	switch t {
	case principal.TypeHuman:
		return principaltoken.PrincipalTypeHuman
	case principal.TypeApplication:
		return principaltoken.PrincipalTypeApplication
	case principal.TypeAgent:
		return principaltoken.PrincipalTypeAgent
	case principal.TypeService:
		return principaltoken.PrincipalTypeService
	default:
		return principaltoken.PrincipalTypeHuman
	}
}
