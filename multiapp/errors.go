package multiapp

import "errors"

// Sentinel errors for the multiapp package.
var (
	// ErrNoAppContext is returned when app context is required but not present.
	ErrNoAppContext = errors.New("multiapp: no app context in request")

	// ErrFeatureNotEnabled is returned when a required feature is not enabled.
	ErrFeatureNotEnabled = errors.New("multiapp: feature not enabled for this app")

	// ErrNotAuthenticated is returned when authentication is required but not present.
	ErrNotAuthenticated = errors.New("multiapp: authentication required")

	// ErrAppNotFound is returned when the requested app is not registered.
	ErrAppNotFound = errors.New("multiapp: app not found")

	// ErrAppAlreadyRegistered is returned when trying to register a duplicate app.
	ErrAppAlreadyRegistered = errors.New("multiapp: app already registered")

	// ErrInvalidSchemaName is returned when a schema name is invalid.
	ErrInvalidSchemaName = errors.New("multiapp: invalid schema name")

	// ErrMigrationFailed is returned when database migrations fail.
	ErrMigrationFailed = errors.New("multiapp: migration failed")
)
