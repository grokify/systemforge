// Package oauth provides OAuth2 provider configuration and handlers for CoreForge.
package oauth

import (
	"strconv"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// Provider represents a supported OAuth2 provider.
type Provider string

const (
	// GitHub is the GitHub OAuth provider.
	GitHub Provider = "github"
	// Google is the Google OAuth provider.
	Google Provider = "google"
)

// ProviderConfig holds configuration for an OAuth2 provider.
type ProviderConfig struct {
	// Provider is the provider identifier.
	Provider Provider

	// ClientID is the OAuth2 client ID.
	ClientID string

	// ClientSecret is the OAuth2 client secret.
	ClientSecret string //nolint:gosec // G117: config field, not a hardcoded secret

	// RedirectURL is the OAuth2 callback URL.
	RedirectURL string

	// Scopes are the OAuth2 scopes to request.
	Scopes []string
}

// OAuth2Config returns an oauth2.Config for this provider.
func (p *ProviderConfig) OAuth2Config() *oauth2.Config {
	cfg := &oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		RedirectURL:  p.RedirectURL,
		Scopes:       p.Scopes,
	}

	switch p.Provider {
	case GitHub:
		cfg.Endpoint = github.Endpoint
		if len(cfg.Scopes) == 0 {
			cfg.Scopes = []string{"user:email", "read:user"}
		}
	case Google:
		cfg.Endpoint = google.Endpoint
		if len(cfg.Scopes) == 0 {
			cfg.Scopes = []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			}
		}
	}

	return cfg
}

// UserInfo represents user information from an OAuth provider.
//
//nolint:gosec // G117: struct fields hold runtime OAuth tokens
type UserInfo struct {
	// ID is the user's ID from the provider.
	ID string

	// Email is the user's email address.
	Email string

	// Name is the user's display name.
	Name string

	// AvatarURL is the URL to the user's avatar image.
	AvatarURL string

	// Provider is the OAuth provider.
	Provider Provider

	// AccessToken is the OAuth access token.
	AccessToken string

	// RefreshToken is the OAuth refresh token (if provided).
	RefreshToken string
}

// GitHubUser represents a GitHub user profile.
type GitHubUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// ToUserInfo converts a GitHub user to UserInfo.
func (u *GitHubUser) ToUserInfo(accessToken, refreshToken string) *UserInfo {
	name := u.Name
	if name == "" {
		name = u.Login
	}
	return &UserInfo{
		ID:           formatInt(u.ID),
		Email:        u.Email,
		Name:         name,
		AvatarURL:    u.AvatarURL,
		Provider:     GitHub,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
}

// GoogleUser represents a Google user profile.
type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

// ToUserInfo converts a Google user to UserInfo.
func (u *GoogleUser) ToUserInfo(accessToken, refreshToken string) *UserInfo {
	return &UserInfo{
		ID:           u.ID,
		Email:        u.Email,
		Name:         u.Name,
		AvatarURL:    u.Picture,
		Provider:     Google,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
}

// formatInt converts an int to a string.
func formatInt(n int) string {
	return strconv.Itoa(n)
}
