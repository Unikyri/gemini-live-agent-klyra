package repositories

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// JWTService implements ports.TokenService using the golang-jwt library.
type JWTService struct {
	accessSecret  string
	refreshSecret string
	accessTTL     time.Duration // 15 minutes
	refreshTTL    time.Duration // 7 days
}

// NewJWTService creates a JWTService with the provided secrets and TTLs.
func NewJWTService(accessSecret, refreshSecret string) *JWTService {
	return &JWTService{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
		accessTTL:     15 * time.Minute,
		refreshTTL:    7 * 24 * time.Hour,
	}
}

// GenerateAccessToken creates a short-lived JWT with user claims.
// SECURITY: Do NOT include sensitive data (passwords, PII) in the JWT payload.
func (s *JWTService) GenerateAccessToken(user *domain.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":   user.ID.String(),
		"email": user.Email,
		"name":  user.Name,
		"exp":   time.Now().Add(s.accessTTL).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.accessSecret))
}

// GenerateRefreshToken creates a signed, long-lived token for secure rotation.
// The `jti` (JWT ID) claim enables per-token revocation in future iterations.
func (s *JWTService) GenerateRefreshToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID,
		"type": "refresh",
		"jti":  uuid.New().String(),
		"exp":  time.Now().Add(s.refreshTTL).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.refreshSecret))
}

// ValidateAccessToken parses and verifies a JWT, returning the claims map.
func (s *JWTService) ValidateAccessToken(tokenString string) (map[string]interface{}, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.accessSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}
