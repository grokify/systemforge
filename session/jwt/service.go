package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Service provides JWT token operations.
type Service struct {
	config *Config
}

// NewService creates a new JWT service with the given configuration.
func NewService(cfg *Config) (*Service, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid jwt config: %w", err)
	}

	return &Service{config: cfg}, nil
}

var (
	// ErrInvalidToken is returned when a token cannot be parsed or validated.
	ErrInvalidToken = errors.New("invalid token")
	// ErrTokenExpired is returned when a token has expired.
	ErrTokenExpired = errors.New("token expired")
	// ErrWrongTokenType is returned when the token type doesn't match expectations.
	ErrWrongTokenType = errors.New("wrong token type")
)

// GenerateAccessToken creates a new access token for the given user.
func (s *Service) GenerateAccessToken(userID uuid.UUID, email, name string) (string, error) {
	claims := NewAccessClaims(s.config, userID, email, name)
	return s.signToken(claims)
}

// TokenOptions contains optional parameters for token generation.
type TokenOptions struct {
	// DPoPThumbprint binds the token to a DPoP key pair.
	// When set, the token will include a cnf.jkt claim.
	DPoPThumbprint string
}

// GenerateAccessTokenWithOptions creates a new access token with optional parameters.
func (s *Service) GenerateAccessTokenWithOptions(userID uuid.UUID, email, name string, opts TokenOptions) (string, error) {
	claims := NewAccessClaims(s.config, userID, email, name)
	if opts.DPoPThumbprint != "" {
		claims.WithDPoPBinding(opts.DPoPThumbprint)
	}
	return s.signToken(claims)
}

// GenerateAccessTokenWithOrg creates an access token with organization context.
func (s *Service) GenerateAccessTokenWithOrg(
	userID uuid.UUID,
	email, name string,
	orgID uuid.UUID,
	orgSlug, role string,
	permissions []string,
	isPlatformAdmin bool,
) (string, error) {
	claims := NewAccessClaims(s.config, userID, email, name).
		WithOrganization(orgID, orgSlug, role, permissions).
		WithPlatformAdmin(isPlatformAdmin)
	return s.signToken(claims)
}

// GenerateAccessTokenWithOrgAndOptions creates an access token with organization context and options.
func (s *Service) GenerateAccessTokenWithOrgAndOptions(
	userID uuid.UUID,
	email, name string,
	orgID uuid.UUID,
	orgSlug, role string,
	permissions []string,
	isPlatformAdmin bool,
	opts TokenOptions,
) (string, error) {
	claims := NewAccessClaims(s.config, userID, email, name).
		WithOrganization(orgID, orgSlug, role, permissions).
		WithPlatformAdmin(isPlatformAdmin)
	if opts.DPoPThumbprint != "" {
		claims.WithDPoPBinding(opts.DPoPThumbprint)
	}
	return s.signToken(claims)
}

// GenerateRefreshToken creates a new refresh token for the given user.
// The family parameter is used for refresh token rotation tracking.
func (s *Service) GenerateRefreshToken(userID uuid.UUID, family string) (string, error) {
	if family == "" {
		family = uuid.NewString()
	}
	claims := NewRefreshClaims(s.config, userID, family)
	return s.signToken(claims)
}

// TokenPair represents an access token and refresh token pair.
//
//nolint:gosec // G117: struct fields hold runtime OAuth tokens
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // Access token expiry in seconds
}

// GenerateTokenPair creates both an access token and refresh token.
func (s *Service) GenerateTokenPair(userID uuid.UUID, email, name string) (*TokenPair, error) {
	accessToken, err := s.GenerateAccessToken(userID, email, name)
	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	family := uuid.NewString()
	refreshToken, err := s.GenerateRefreshToken(userID, family)
	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.AccessTokenExpiry.Seconds()),
	}, nil
}

// GenerateTokenPairWithOptions creates a token pair with optional DPoP binding.
func (s *Service) GenerateTokenPairWithOptions(userID uuid.UUID, email, name string, opts TokenOptions) (*TokenPair, error) {
	accessToken, err := s.GenerateAccessTokenWithOptions(userID, email, name, opts)
	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	family := uuid.NewString()
	refreshToken, err := s.GenerateRefreshToken(userID, family)
	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.AccessTokenExpiry.Seconds()),
	}, nil
}

// GenerateTokenPairWithOrg creates a token pair with organization context.
func (s *Service) GenerateTokenPairWithOrg(
	userID uuid.UUID,
	email, name string,
	orgID uuid.UUID,
	orgSlug, role string,
	permissions []string,
	isPlatformAdmin bool,
) (*TokenPair, error) {
	accessToken, err := s.GenerateAccessTokenWithOrg(
		userID, email, name, orgID, orgSlug, role, permissions, isPlatformAdmin,
	)
	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	family := uuid.NewString()
	refreshToken, err := s.GenerateRefreshToken(userID, family)
	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.AccessTokenExpiry.Seconds()),
	}, nil
}

// ValidateAccessToken validates and parses an access token.
func (s *Service) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims, err := s.parseToken(tokenString)
	if err != nil {
		return nil, err
	}

	if !claims.IsAccessToken() {
		return nil, ErrWrongTokenType
	}

	return claims, nil
}

// ValidateRefreshToken validates and parses a refresh token.
func (s *Service) ValidateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := s.parseToken(tokenString)
	if err != nil {
		return nil, err
	}

	if !claims.IsRefreshToken() {
		return nil, ErrWrongTokenType
	}

	return claims, nil
}

// signToken creates a signed JWT from the given claims.
func (s *Service) signToken(claims *Claims) (string, error) {
	var method jwt.SigningMethod
	switch s.config.Algorithm {
	case "HS256":
		method = jwt.SigningMethodHS256
	case "HS384":
		method = jwt.SigningMethodHS384
	case "HS512":
		method = jwt.SigningMethodHS512
	case "RS256":
		method = jwt.SigningMethodRS256
	case "RS384":
		method = jwt.SigningMethodRS384
	case "RS512":
		method = jwt.SigningMethodRS512
	case "ES256":
		method = jwt.SigningMethodES256
	case "ES384":
		method = jwt.SigningMethodES384
	case "ES512":
		method = jwt.SigningMethodES512
	default:
		method = jwt.SigningMethodHS256
	}

	token := jwt.NewWithClaims(method, claims)

	var key any
	if s.config.PrivateKey != nil {
		key = s.config.PrivateKey
	} else {
		key = s.config.Secret
	}

	return token.SignedString(key)
}

// parseToken parses and validates a JWT token string.
func (s *Service) parseToken(tokenString string) (*Claims, error) {
	keyFunc := func(token *jwt.Token) (any, error) {
		// Verify signing method matches configuration
		switch s.config.Algorithm {
		case "HS256", "HS384", "HS512":
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return s.config.Secret, nil
		case "RS256", "RS384", "RS512":
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return s.config.PublicKey, nil
		case "ES256", "ES384", "ES512":
			if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return s.config.PublicKey, nil
		default:
			return nil, ErrInvalidAlgorithm
		}
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, keyFunc)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// Config returns the service configuration.
func (s *Service) Config() *Config {
	return s.config
}

// RefreshTokenTTL returns the refresh token expiry duration.
// This method provides compatibility with goauth/jwt.Service.
func (s *Service) RefreshTokenTTL() time.Duration {
	return s.config.RefreshTokenExpiry
}

// AccessTokenTTL returns the access token expiry duration.
// This method provides compatibility with goauth/jwt.Service.
func (s *Service) AccessTokenTTL() time.Duration {
	return s.config.AccessTokenExpiry
}

// GenerateTokenPairLegacy creates a token pair using the legacy goauth interface.
// This provides backward compatibility during migration from goauth/jwt.
// Deprecated: Use GenerateTokenPair or GenerateTokenPairWithOrg instead.
func (s *Service) GenerateTokenPairLegacy(userID uuid.UUID, email string, isPlatformAdmin bool) (*TokenPair, error) {
	// Use isPlatformAdmin to determine name fallback
	name := email // Use email as name fallback

	accessToken, err := s.GenerateAccessTokenWithOrg(
		userID, email, name,
		uuid.Nil, "", "", nil, isPlatformAdmin,
	)
	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	family := uuid.NewString()
	refreshToken, err := s.GenerateRefreshToken(userID, family)
	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.AccessTokenExpiry.Seconds()),
	}, nil
}
