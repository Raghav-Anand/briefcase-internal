package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MicahParks/keyfunc"
	"github.com/golang-jwt/jwt/v4"
)

const (
	googleJWKSURL = "https://www.googleapis.com/oauth2/v3/certs"
	issuer1       = "https://accounts.google.com"
	issuer2       = "accounts.google.com"
)

// Claims holds the verified user identity extracted from a Google ID token.
type Claims struct {
	UID     string // Google user ID (the `sub` claim — stable across sessions)
	Email   string
	Name    string
	Picture string
}

// Verifier validates Google ID tokens (OIDC JWTs) against Google's public JWKS.
type Verifier struct {
	clientID string
	once     sync.Once
	jwks     *keyfunc.JWKS
	initErr  error
}

// NewVerifier creates a token verifier for the given OAuth 2.0 client ID.
// The client ID is used to verify the `aud` claim in the token.
// JWKS keys are fetched from Google on first use and cached with automatic rotation.
func NewVerifier(clientID string) *Verifier {
	return &Verifier{clientID: clientID}
}

func (v *Verifier) init() {
	v.once.Do(func() {
		jwks, err := keyfunc.Get(googleJWKSURL, keyfunc.Options{
			RefreshInterval:   6 * time.Hour,
			RefreshRateLimit:  5 * time.Minute,
			RefreshUnknownKID: true,
		})
		v.jwks = jwks
		v.initErr = err
	})
}

// VerifyToken validates a Google ID token (JWT) and returns user claims.
// Checks signature (via JWKS), expiry, issuer (accounts.google.com), and audience.
// Used by both MCP server and API server auth middleware.
func (v *Verifier) VerifyToken(ctx context.Context, idToken string) (*Claims, error) {
	v.init()
	if v.initErr != nil {
		return nil, fmt.Errorf("JWKS init: %w", v.initErr)
	}

	var mapClaims jwt.MapClaims
	token, err := jwt.ParseWithClaims(idToken, &mapClaims, v.jwks.Keyfunc)
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	iss, _ := mapClaims["iss"].(string)
	if iss != issuer1 && iss != issuer2 {
		return nil, fmt.Errorf("invalid issuer: %s", iss)
	}

	if !mapClaims.VerifyAudience(v.clientID, true) {
		return nil, fmt.Errorf("invalid audience")
	}

	sub, _ := mapClaims["sub"].(string)
	if sub == "" {
		return nil, fmt.Errorf("missing sub claim")
	}

	claims := &Claims{UID: sub}
	if email, ok := mapClaims["email"].(string); ok {
		claims.Email = email
	}
	if name, ok := mapClaims["name"].(string); ok {
		claims.Name = name
	}
	if picture, ok := mapClaims["picture"].(string); ok {
		claims.Picture = picture
	}

	return claims, nil
}
