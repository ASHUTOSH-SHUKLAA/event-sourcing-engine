package auth

import (
	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/auth/handler"
	"gin-quickstart/internal/auth/service"
	"gin-quickstart/internal/user/repository"
)

// RegisterRoutes wires all auth endpoints onto the given Gin router group.
// Requirements: 1.1, 2.1, 3.1
func RegisterRoutes(rg *gin.RouterGroup, repo repository.UserRepository) {
	tokenSvc := service.NewTokenService()
	authSvc := service.NewAuthService(repo, tokenSvc)
	h := handler.NewAuthHandler(authSvc, tokenSvc)

	auth := rg.Group("/auth")
	{
		auth.POST("/register", h.RegisterHandler)
		auth.POST("/login", h.LoginHandler)
		auth.POST("/refresh", h.RefreshHandler)
	}
}
