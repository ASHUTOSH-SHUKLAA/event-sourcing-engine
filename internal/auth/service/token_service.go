package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"gin-quickstart/internal/config"
)

// Claims is the JWT payload used for both access and refresh tokens.
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// Sentinel errors for token validation failures.
var (
	ErrTokenExpired  = errors.New("token expired")
	ErrTokenInvalid  = errors.New("invalid token")
	ErrMissingSecret = errors.New("JWT secret is not configured")
)

// TokenService issues and validates JWT access and refresh tokens.
// Requirements: 2.4, 2.5, 3.1, 3.2, 3.3
type TokenService interface {
	GenerateAccessToken(userID string) (string, error)
	GenerateRefreshToken(userID string) (string, error)
	ValidateAccessToken(token string) (*Claims, error)
	ValidateRefreshToken(token string) (*Claims, error)
}

type tokenService struct{}

// NewTokenService returns a TokenService that reads secrets from environment config.
func NewTokenService() TokenService {
	return &tokenService{}
}

// GenerateAccessToken signs a JWT with JWT_SECRET and a 15-minute expiry.
// Requirement: 2.4
func (t *tokenService) GenerateAccessToken(userID string) (string, error) {
	secret := config.GetJWTSecret()
	if secret == "" {
		return "", ErrMissingSecret
	}
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken signs a JWT with JWT_REFRESH_SECRET and a 7-day expiry.
// Requirement: 2.5
func (t *tokenService) GenerateRefreshToken(userID string) (string, error) {
	secret := config.GetJWTRefreshSecret()
	if secret == "" {
		return "", ErrMissingSecret
	}
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateAccessToken parses and validates a JWT access token against JWT_SECRET.
// Requirements: 3.2, 3.3
func (t *tokenService) ValidateAccessToken(tokenStr string) (*Claims, error) {
	return parseToken(tokenStr, config.GetJWTSecret())
}

// ValidateRefreshToken parses and validates a JWT refresh token against JWT_REFRESH_SECRET.
// Requirements: 3.1, 3.3
func (t *tokenService) ValidateRefreshToken(tokenStr string) (*Claims, error) {
	return parseToken(tokenStr, config.GetJWTRefreshSecret())
}

// parseToken is the shared validation logic for both token types.
func parseToken(tokenStr, secret string) (*Claims, error) {
	if secret == "" {
		return nil, ErrMissingSecret
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	if !token.Valid {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}
