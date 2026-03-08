package repositories

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTServiceImpl implements token generation and validation.
type JWTServiceImpl struct {
	acccessSecret  string
	refreshSecret  string
	accessExpiry   time.Duration
	refreshExpiry  time.Duration
}

// NewJWTService creates a new JWT service with configured secrets and expiry times.
func NewJWTService(accessSecret, refreshSecret string, accessExpiry, refreshExpiry time.Duration) *JWTServiceImpl {
	return &JWTServiceImpl{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
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
	accessToken, err = accessJWT.SignedString([]byte(s.acccessSecret))
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

// ValidateAccessToken validates and extracts user ID from access token
func (s *JWTServiceImpl) ValidateAccessToken(token string) (userID string, err error) {
	claims := &jwtClaims{}
	_, err = jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.acccessSecret), nil
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
