package repositories

import (
	"context"
	"fmt"
	"os"
	"strings"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// ResolveGoogleClientOptions returns client options based on credentials configuration.
//
// Precedence:
//  1) GOOGLE_APPLICATION_CREDENTIALS_JSON (inline JSON) if present
//  2) GOOGLE_APPLICATION_CREDENTIALS:
//     - if starts with '{' treat as inline JSON
//     - else treat as file path
//  3) none -> ADC (Cloud Run, local gcloud auth, etc.)
func ResolveGoogleClientOptions(scopes ...string) ([]option.ClientOption, error) {
	if jsonVal := strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")); jsonVal != "" {
		return credsJSONOptions(jsonVal, scopes...)
	}

	credsVal := strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	if credsVal == "" {
		return nil, nil
	}
	if strings.HasPrefix(credsVal, "{") {
		return credsJSONOptions(credsVal, scopes...)
	}

	return []option.ClientOption{option.WithCredentialsFile(credsVal)}, nil
}

func credsJSONOptions(jsonVal string, scopes ...string) ([]option.ClientOption, error) {
	creds, err := google.CredentialsFromJSON(context.Background(), []byte(jsonVal), scopes...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials json: %w", err)
	}
	return []option.ClientOption{option.WithCredentials(creds)}, nil
}

