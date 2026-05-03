package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/middleware"
	userservice "gin-quickstart/internal/user/service"
)

// UserHandler holds dependencies for user profile HTTP handlers.
type UserHandler struct {
	userSvc userservice.UserService
}

// NewUserHandler constructs a UserHandler.
func NewUserHandler(userSvc userservice.UserService) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

// updateProfileRequest is the expected JSON body for PUT /api/v1/users/me.
type updateProfileRequest struct {
	DisplayName string `json:"display_name"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

// GetProfileHandler handles GET /api/v1/users/me.
// Extracts userID from context (set by AuthMiddleware), returns UserResponse.
// Requirements: 4.1
func (h *UserHandler) GetProfileHandler(c *gin.Context) {
	userID, ok := c.Get(middleware.UserIDKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := h.userSvc.GetProfile(c.Request.Context(), userID.(string))
	if err != nil {
		if errors.Is(err, userservice.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": user.ToResponse()})
}

// UpdateProfileHandler handles PUT /api/v1/users/me.
// Parses display_name from body, updates the user, returns updated UserResponse.
// Requirements: 4.2
func (h *UserHandler) UpdateProfileHandler(c *gin.Context) {
	userID, ok := c.Get(middleware.UserIDKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required fields"})
		return
	}

	user, err := h.userSvc.UpdateProfile(c.Request.Context(), userID.(string), req.DisplayName)
	if err != nil {
		if errors.Is(err, userservice.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": user.ToResponse()})
}

// ChangePasswordHandler handles PATCH /api/v1/users/me/password.
func (h *UserHandler) ChangePasswordHandler(c *gin.Context) {
	userID, ok := c.Get(middleware.UserIDKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required fields"})
		return
	}
	if req.CurrentPassword == "" || req.NewPassword == "" || req.ConfirmPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required fields"})
		return
	}
	if req.NewPassword != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password and confirm password must match"})
		return
	}

	err := h.userSvc.ChangePassword(c.Request.Context(), userID.(string), req.CurrentPassword, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, userservice.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		case errors.Is(err, userservice.ErrInvalidPassword):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "current password is incorrect"})
		case errors.Is(err, userservice.ErrPasswordTooShort):
			c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"updated": true}})
}
