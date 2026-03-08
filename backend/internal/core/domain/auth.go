package domain

// AuthResult holds the tokens and user information returned after successful authentication.
// This is returned by any authentication strategy (Google, Guest, Email/Password).
type AuthResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user"`
	// Provider identifies which authentication method was used (e.g., "google", "guest").
	// This helps with analytics, debugging, and potential provider-specific logic.
	Provider string `json:"provider"`
}

// AuthCredentials represents the credentials passed to an authentication strategy.
// Different strategies expect different credential formats:
// - Google: {"id_token": "eyJhbGc..."}
// - Guest: {"email": "test@example.com", "name": "Test User"}
// - Future Email/Password: {"email": "...", "password": "..."}
//
// Using map[string]interface{} provides flexibility for different auth providers
// without forcing a rigid struct schema.
type AuthCredentials map[string]interface{}

// GetString safely extracts a string value from credentials.
// Returns empty string if key doesn't exist or value is not a string.
func (c AuthCredentials) GetString(key string) string {
	val, ok := c[key]
	if !ok {
		return ""
	}
	str, ok := val.(string)
	if !ok {
		return ""
	}
	return str
}

// GetInt safely extracts an int value from credentials.
// Returns 0 if key doesn't exist or value is not convertible to int.
func (c AuthCredentials) GetInt(key string) int {
	val, ok := c[key]
	if !ok {
		return 0
	}

	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		return 0
	}
}
