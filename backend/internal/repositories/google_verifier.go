package repositories

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/api/idtoken"
)

// GoogleVerifier implements ports.GoogleTokenVerifier.
// It validates Google ID Tokens using Google's public key infrastructure.
type GoogleVerifier struct {
	// clientID is the OAuth2 Client ID registered in Google Cloud Console.
	// The ID Token must be issued for this exact audience.
	clientID string
}

// NewGoogleVerifier creates a GoogleVerifier for the given Google OAuth2 Client ID.
func NewGoogleVerifier(clientID string) *GoogleVerifier {
	return &GoogleVerifier{clientID: clientID}
}

// Verify validates the given Google ID Token and extracts user info from its claims.
// SECURITY: We validate the `aud` (audience) to ensure the token was issued
// specifically for our app, preventing token interception attacks.
func (v *GoogleVerifier) Verify(ctx context.Context, idToken string) (email, name, picture string, err error) {
	payload, err := idtoken.Validate(ctx, idToken, v.clientID)
	if err != nil {
		log.Printf("[GoogleVerifier] Validate failed — clientID=%s — error: %v", v.clientID, err)
		return "", "", "", fmt.Errorf("google token validation failed: %w", err)
	}

	emailVal, _ := payload.Claims["email"].(string)
	nameVal, _ := payload.Claims["name"].(string)
	pictureVal, _ := payload.Claims["picture"].(string)

	if emailVal == "" {
		return "", "", "", fmt.Errorf("google token missing email claim")
	}

	return emailVal, nameVal, pictureVal, nil
}
