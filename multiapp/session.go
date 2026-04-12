package multiapp

import (
	"time"

	"github.com/google/uuid"
	"github.com/grokify/coreforge/session/bff"
)

// AppSession extends bff.Session with app-specific fields.
// This provides app-scoped sessions for multi-app deployments.
type AppSession struct {
	*bff.Session

	// AppID is the app this session belongs to.
	AppID string

	// AppSlug is the URL-safe app identifier.
	AppSlug string
}

// SessionKeyWithApp returns a session store key that includes the app ID.
// This ensures sessions are isolated per-app in shared session stores.
func SessionKeyWithApp(appID, sessionID string) string {
	return appID + ":" + sessionID
}

// NewAppSession creates an app-scoped session from a BFF session.
func NewAppSession(session *bff.Session, appID, appSlug string) *AppSession {
	return &AppSession{
		Session: session,
		AppID:   appID,
		AppSlug: appSlug,
	}
}

// CreateAppSession creates a new app-scoped session.
func CreateAppSession(
	appID, appSlug string,
	userID uuid.UUID,
	accessToken, refreshToken string,
	accessExpiry, refreshExpiry time.Duration,
) (*AppSession, error) {
	// Create base BFF session
	session, err := bff.NewSession(
		userID,
		accessToken,
		refreshToken,
		accessExpiry,
		refreshExpiry,
	)
	if err != nil {
		return nil, err
	}

	// Add app metadata to the session
	if session.Metadata == nil {
		session.Metadata = make(map[string]string)
	}
	session.Metadata["app_id"] = appID
	session.Metadata["app_slug"] = appSlug

	return &AppSession{
		Session: session,
		AppID:   appID,
		AppSlug: appSlug,
	}, nil
}

// AppSessionMetadata returns metadata for storing app context in session.
// This can be added to bff.Session.Metadata for app tracking.
func AppSessionMetadata(appID, appSlug string) map[string]string {
	return map[string]string{
		"app_id":   appID,
		"app_slug": appSlug,
	}
}

// AppIDFromSessionMetadata extracts app ID from session metadata.
func AppIDFromSessionMetadata(session *bff.Session) string {
	if session == nil || session.Metadata == nil {
		return ""
	}
	return session.Metadata["app_id"]
}

// AppSlugFromSessionMetadata extracts app slug from session metadata.
func AppSlugFromSessionMetadata(session *bff.Session) string {
	if session == nil || session.Metadata == nil {
		return ""
	}
	return session.Metadata["app_slug"]
}
