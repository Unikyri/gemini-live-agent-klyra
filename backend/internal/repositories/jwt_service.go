package repositories

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// JWTServiceImpl implements token generation and validation.
type JWTServiceImpl struct {
	accessSecret  string
	refreshSecret string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// NewJWTService creates a new JWT service with configured secrets and expiry times.
func NewJWTService(accessSecret, refreshSecret string) *JWTServiceImpl {
	return &JWTServiceImpl{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
		accessExpiry:  15 * time.Minute,
		refreshExpiry: 7 * 24 * time.Hour,
	}
}

// jwtClaims custom claims structure for JWT tokens
type jwtClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateTokens creates access (15min) and refresh (7d) tokens
func (s *JWTServiceImpl) GenerateTokens(userID string) (accessToken, refreshToken string, err error) {
	// Access token (short-lived)
	accessClaims := jwtClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	accessJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accessJWT.SignedString([]byte(s.accessSecret))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	// Refresh token (long-lived)
	refreshClaims := jwtClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	refreshJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = refreshJWT.SignedString([]byte(s.refreshSecret))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// ValidateAccessTokenUserID validates and extracts user ID from access token (legacy helper).
func (s *JWTServiceImpl) ValidateAccessTokenUserID(token string) (userID string, err error) {
	claims := &jwtClaims{}
	_, err = jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.accessSecret), nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse access token: %w", err)
	}

	return claims.UserID, nil
}

// ValidateRefreshToken validates and extracts user ID from refresh token
func (s *JWTServiceImpl) ValidateRefreshToken(token string) (userID string, err error) {
	claims := &jwtClaims{}
	_, err = jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.refreshSecret), nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse refresh token: %w", err)
	}

	return claims.UserID, nil
}

// RefreshAccessToken generates a new access token from a valid refresh token
func (s *JWTServiceImpl) RefreshAccessToken(refreshToken string) (newAccessToken string, err error) {
	userID, err := s.ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	newAccessToken, _, err = s.GenerateTokens(userID)
	return newAccessToken, err
}

// GenerateAccessToken implements ports.TokenService.
func (s *JWTServiceImpl) GenerateAccessToken(user *domain.User) (string, error) {
	if user == nil {
		return "", fmt.Errorf("user is nil")
	}
	claims := jwt.MapClaims{
		"sub":   user.ID.String(),
		"email": user.Email,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(s.accessExpiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.accessSecret))
}

// GenerateRefreshToken implements ports.TokenService.
func (s *JWTServiceImpl) GenerateRefreshToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(s.refreshExpiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.refreshSecret))
}

// ValidateAccessToken implements ports.TokenService.
func (s *JWTServiceImpl) ValidateAccessToken(tokenString string) (map[string]interface{}, error) {
	parsed, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.accessSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	out := make(map[string]interface{}, len(claims))
	for k, v := range claims {
		out[k] = v
	}
	return out, nil
}
