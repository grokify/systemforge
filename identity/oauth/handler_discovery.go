package oauth

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// OpenIDConfiguration represents the OpenID Connect discovery document.
type OpenIDConfiguration struct {
	Issuer                            string   `json:"issuer" doc:"OAuth 2.0 Issuer Identifier"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint" doc:"URL of the authorization endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint" doc:"URL of the token endpoint"`
	IntrospectionEndpoint             string   `json:"introspection_endpoint" doc:"URL of the introspection endpoint"`
	RevocationEndpoint                string   `json:"revocation_endpoint" doc:"URL of the revocation endpoint"`
	JWKSURI                           string   `json:"jwks_uri" doc:"URL of the JSON Web Key Set document"`
	ResponseTypesSupported            []string `json:"response_types_supported" doc:"List of supported response types"`
	GrantTypesSupported               []string `json:"grant_types_supported" doc:"List of supported grant types"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported" doc:"List of supported token endpoint authentication methods"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported" doc:"List of supported PKCE code challenge methods"`
}

// JWKS represents a JSON Web Key Set.
type JWKS struct {
	Keys []JWK `json:"keys" doc:"Array of JSON Web Keys"`
}

// JWK represents a JSON Web Key.
type JWK struct {
	Kty string `json:"kty" doc:"Key type (e.g., RSA, EC)"`
	Use string `json:"use,omitempty" doc:"Key use (sig for signature, enc for encryption)"`
	Kid string `json:"kid,omitempty" doc:"Key ID"`
	Alg string `json:"alg,omitempty" doc:"Algorithm"`
	N   string `json:"n,omitempty" doc:"RSA modulus"`
	E   string `json:"e,omitempty" doc:"RSA exponent"`
}

// OpenIDConfigOutput is the response for the OpenID Configuration endpoint.
type OpenIDConfigOutput struct {
	Body OpenIDConfiguration
}

// JWKSOutput is the response for the JWKS endpoint.
type JWKSOutput struct {
	Body JWKS
}

// registerDiscoveryEndpoints registers OAuth/OIDC discovery endpoints with Huma.
func (a *API) registerDiscoveryEndpoints() {
	// OpenID Configuration
	huma.Register(a.huma, huma.Operation{
		OperationID:   "getOpenIDConfiguration",
		Method:        http.MethodGet,
		Path:          "/.well-known/openid-configuration",
		Summary:       "Get OpenID Connect configuration",
		Description:   "Returns the OpenID Connect discovery document",
		Tags:          []string{"Discovery"},
		DefaultStatus: http.StatusOK,
	}, a.getOpenIDConfiguration)

	// JWKS
	huma.Register(a.huma, huma.Operation{
		OperationID:   "getJWKS",
		Method:        http.MethodGet,
		Path:          "/.well-known/jwks.json",
		Summary:       "Get JSON Web Key Set",
		Description:   "Returns the public keys for token verification",
		Tags:          []string{"Discovery"},
		DefaultStatus: http.StatusOK,
	}, a.getJWKS)
}

// getOpenIDConfiguration returns the OpenID Connect discovery document.
func (a *API) getOpenIDConfiguration(ctx context.Context, input *struct{}) (*OpenIDConfigOutput, error) {
	issuer := a.provider.config.Issuer

	config := OpenIDConfiguration{
		Issuer:                issuer,
		AuthorizationEndpoint: issuer + "/oauth/authorize",
		TokenEndpoint:         issuer + "/oauth/token",
		IntrospectionEndpoint: issuer + "/oauth/introspect",
		RevocationEndpoint:    issuer + "/oauth/revoke",
		JWKSURI:               issuer + "/.well-known/jwks.json",
		ResponseTypesSupported: []string{
			"code",
			"token",
		},
		GrantTypesSupported: []string{
			"authorization_code",
			"refresh_token",
			"client_credentials",
		},
		TokenEndpointAuthMethodsSupported: []string{
			"client_secret_basic",
			"client_secret_post",
			"none",
		},
		CodeChallengeMethodsSupported: []string{
			"S256",
		},
	}

	return &OpenIDConfigOutput{Body: config}, nil
}

// getJWKS returns the JSON Web Key Set for token verification.
func (a *API) getJWKS(ctx context.Context, input *struct{}) (*JWKSOutput, error) {
	// Return empty JWKS for now - in production, this would return actual public keys
	// TODO: Add actual JWK generation from the provider's RSA key
	return &JWKSOutput{Body: JWKS{Keys: []JWK{}}}, nil
}
