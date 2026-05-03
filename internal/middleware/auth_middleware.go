package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/auth/service"
)

const (
	UserIDKey   = "userID"
	UserRoleKey = "userRole"
)

// AuthMiddleware validates the Authorization: Bearer <token> header and injects
// the userID into the Gin context. Returns 401 for missing, malformed, or expired tokens.
// Requirements: 4.3, 4.4
func AuthMiddleware(tokenSvc service.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		claims, err := tokenSvc.ValidateAccessToken(parts[1])
		if err != nil {
			if errors.Is(err, service.ErrTokenExpired) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Set(UserRoleKey, claims.Role)
		c.Next()
	}
}

// AdminOnly blocks any request where the user role is not 'admin'.
// Requirement: 6.3
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get(UserRoleKey)
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden: admin access required"})
			return
		}
		c.Next()
	}
}
