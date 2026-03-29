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
