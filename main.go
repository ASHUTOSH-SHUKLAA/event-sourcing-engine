package main

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gin-quickstart/internal/auth"
	"gin-quickstart/internal/auth/service"
	"gin-quickstart/internal/config"
	"gin-quickstart/internal/database"
	"gin-quickstart/internal/user"
	"gin-quickstart/internal/user/repository"
)

func main() {

	//  Logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	//  Check DB URL
	dbURL := config.GetDBUrl()
	if dbURL == "" {
		logger.Fatal("DATABASE_URL not set")
	}

	//  Connect DB
	database.Connect(dbURL)

	//  Router
	router := gin.Default()

	//  CORS FIX (important)
	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:5173",
			"http://localhost:5174",
		},
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"DELETE",
			"OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Authorization",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	//  Zap logging middleware
	router.Use(func(c *gin.Context) {
		c.Next()

		logger.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
		)
	})

	router.Use(gin.Recovery())

	//  Health check
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	//  Repository
	userRepo := repository.NewUserRepository(database.Pool)

	//  Services
	tokenSvc := service.NewTokenService()

	//  API group
	v1 := router.Group("/api/v1")

	//  Auth routes
	auth.RegisterRoutes(v1, userRepo)

	//  User routes
	user.RegisterRoutes(v1, userRepo, tokenSvc)

	logger.Info("Server starting on :8080")

	err = router.Run(":8080")
	if err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}
