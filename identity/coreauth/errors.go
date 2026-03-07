package coreauth

import "errors"

// Configuration errors.
var (
	// ErrMissingIssuer is returned when the issuer is not configured.
	ErrMissingIssuer = errors.New("coreauth: issuer is required")

	// ErrKeyGenerationFailed is returned when key generation fails.
	ErrKeyGenerationFailed = errors.New("coreauth: failed to generate signing key")

	// ErrStorageInitFailed is returned when storage initialization fails.
	ErrStorageInitFailed = errors.New("coreauth: failed to initialize storage")
)

// Client errors.
var (
	// ErrClientNotFound is returned when a client is not found.
	ErrClientNotFound = errors.New("coreauth: client not found")

	// ErrClientExists is returned when trying to create a client that already exists.
	ErrClientExists = errors.New("coreauth: client already exists")

	// ErrInvalidClientType is returned when the client type is invalid.
	ErrInvalidClientType = errors.New("coreauth: invalid client type")
)

// Token errors.
var (
	// ErrTokenNotFound is returned when a token is not found.
	ErrTokenNotFound = errors.New("coreauth: token not found")

	// ErrTokenExpired is returned when a token has expired.
	ErrTokenExpired = errors.New("coreauth: token expired")

	// ErrTokenRevoked is returned when a token has been revoked.
	ErrTokenRevoked = errors.New("coreauth: token revoked")

	// ErrInvalidToken is returned when a token is invalid.
	ErrInvalidToken = errors.New("coreauth: invalid token")
)

// Authorization errors.
var (
	// ErrAuthCodeNotFound is returned when an authorization code is not found.
	ErrAuthCodeNotFound = errors.New("coreauth: authorization code not found")

	// ErrAuthCodeExpired is returned when an authorization code has expired.
	ErrAuthCodeExpired = errors.New("coreauth: authorization code expired")

	// ErrAuthCodeUsed is returned when an authorization code has already been used.
	ErrAuthCodeUsed = errors.New("coreauth: authorization code already used")

	// ErrPKCEVerificationFailed is returned when PKCE verification fails.
	ErrPKCEVerificationFailed = errors.New("coreauth: PKCE verification failed")
)

// Federation errors.
var (
	// ErrFederationNotConfigured is returned when federation is not configured.
	ErrFederationNotConfigured = errors.New("coreauth: federation not configured")

	// ErrFederationConnectionFailed is returned when connection to CoreControl fails.
	ErrFederationConnectionFailed = errors.New("coreauth: failed to connect to CoreControl")

	// ErrInvalidGlobalToken is returned when a global identity token is invalid.
	ErrInvalidGlobalToken = errors.New("coreauth: invalid global identity token")
)

// User errors.
var (
	// ErrUserNotFound is returned when a user is not found.
	ErrUserNotFound = errors.New("coreauth: user not found")

	// ErrUserExists is returned when trying to create a user that already exists.
	ErrUserExists = errors.New("coreauth: user already exists")
)
