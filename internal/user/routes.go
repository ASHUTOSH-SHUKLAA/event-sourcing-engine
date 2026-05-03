package user

import (
	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/auth/service"
	"gin-quickstart/internal/middleware"
	"gin-quickstart/internal/user/handler"
	"gin-quickstart/internal/user/repository"
	userservice "gin-quickstart/internal/user/service"
)

// RegisterRoutes wires all user profile endpoints onto the given Gin router group,
// protected by the JWT auth middleware.
// Requirements: 4.1, 4.2, 4.3
func RegisterRoutes(rg *gin.RouterGroup, repo repository.UserRepository, tokenSvc service.TokenService) {
	userSvc := userservice.NewUserService(repo)
	h := handler.NewUserHandler(userSvc)

	users := rg.Group("/users")
	users.Use(middleware.AuthMiddleware(tokenSvc))
	{
		users.GET("/me", h.GetProfileHandler)
		users.PUT("/me", h.UpdateProfileHandler)
		users.PATCH("/me/password", h.ChangePasswordHandler)
	}
}
