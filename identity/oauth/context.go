package oauth

import (
	"context"

	"github.com/ory/fosite"
)

type contextKey string

const accessRequestKey contextKey = "oauth_access_request"

// WithAccessRequest adds the access request to the context.
func WithAccessRequest(ctx context.Context, ar fosite.AccessRequester) context.Context {
	return context.WithValue(ctx, accessRequestKey, ar)
}

// AccessRequestFromContext extracts the access request from context.
func AccessRequestFromContext(ctx context.Context) fosite.AccessRequester {
	ar, _ := ctx.Value(accessRequestKey).(fosite.AccessRequester)
	return ar
}

// UserIDFromContext extracts the user ID (subject) from the access request in context.
func UserIDFromContext(ctx context.Context) string {
	ar := AccessRequestFromContext(ctx)
	if ar == nil {
		return ""
	}
	session := ar.GetSession()
	if session == nil {
		return ""
	}
	return session.GetSubject()
}

// ScopesFromContext extracts the granted scopes from the access request in context.
func ScopesFromContext(ctx context.Context) []string {
	ar := AccessRequestFromContext(ctx)
	if ar == nil {
		return nil
	}
	return ar.GetGrantedScopes()
}

// HasScope checks if the access request in context has a specific scope.
func HasScope(ctx context.Context, scope string) bool {
	ar := AccessRequestFromContext(ctx)
	if ar == nil {
		return false
	}
	return ar.GetGrantedScopes().Has(scope)
}
