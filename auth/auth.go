package auth

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
)

// Claims holds the verified user identity extracted from a Firebase ID token.
type Claims struct {
	UID     string
	Email   string
	Name    string
	Picture string
}

// InitAuth initializes the Firebase Auth client using Application Default Credentials.
// On Cloud Run, ADC resolves automatically via the service account.
// Locally, set GOOGLE_APPLICATION_CREDENTIALS to a service account key file,
// or set FIREBASE_AUTH_EMULATOR_HOST to use the local emulator.
func InitAuth(ctx context.Context, projectID string) (*firebaseauth.Client, error) {
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("firebase.NewApp: %w", err)
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("app.Auth: %w", err)
	}
	return client, nil
}

// VerifyToken validates a Firebase ID token and returns the user's claims.
// Returns an error if the token is expired, malformed, or revoked.
func VerifyToken(ctx context.Context, client *firebaseauth.Client, idToken string) (*Claims, error) {
	token, err := client.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("VerifyIDToken: %w", err)
	}

	claims := &Claims{
		UID: token.UID,
	}
	if v, ok := token.Claims["email"].(string); ok {
		claims.Email = v
	}
	if v, ok := token.Claims["name"].(string); ok {
		claims.Name = v
	}
	if v, ok := token.Claims["picture"].(string); ok {
		claims.Picture = v
	}
	return claims, nil
}
