package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/auth/service"
)

// AuthHandler holds the dependencies for auth HTTP handlers.
type AuthHandler struct {
	authSvc  service.AuthService
	tokenSvc service.TokenService
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(authSvc service.AuthService, tokenSvc service.TokenService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, tokenSvc: tokenSvc}
}

// registerRequest is the expected JSON body for POST /api/v1/auth/register.
type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginRequest is the expected JSON body for POST /api/v1/auth/login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterHandler handles POST /api/v1/auth/register.
// Requirements: 1.1, 1.2, 1.3, 1.4, 6.1, 6.3
func (h *AuthHandler) RegisterHandler(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required fields"})
		return
	}
	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required fields"})
		return
	}

	user, err := h.authSvc.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailTaken):
			c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
		case errors.Is(err, service.ErrInvalidEmail):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email format"})
		case errors.Is(err, service.ErrPasswordTooShort):
			c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": user.ToResponse()})
}

// LoginHandler handles POST /api/v1/auth/login.
// Requirements: 2.1, 2.2, 2.3, 6.1, 6.3
func (h *AuthHandler) LoginHandler(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required fields"})
		return
	}
	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required fields"})
		return
	}

	pair, err := h.authSvc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCreds) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Set refresh token as HTTP-only cookie (not accessible to JS).
	// Requirement: 5.2 (frontend token storage strategy)
	c.SetCookie(
		"refresh_token",
		pair.RefreshToken,
		int((7 * 24 * time.Hour).Seconds()),
		"/",
		"",
		false, // secure — set to true in production behind HTTPS
		true,  // httpOnly
	)

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"access_token": pair.AccessToken}})
}

// RefreshHandler handles POST /api/v1/auth/refresh.
// Requirements: 3.1, 3.2, 6.1
func (h *AuthHandler) RefreshHandler(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	claims, err := h.tokenSvc.ValidateRefreshToken(refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	newAccessToken, err := h.tokenSvc.GenerateAccessToken(claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"access_token": newAccessToken}})
}
