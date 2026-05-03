package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"gin-quickstart/internal/appapi"
	"gin-quickstart/internal/auth"
	authservice "gin-quickstart/internal/auth/service"
	"gin-quickstart/internal/catalog"
	"gin-quickstart/internal/config"
	"gin-quickstart/internal/database"
	"gin-quickstart/internal/repository"
	queueservice "gin-quickstart/internal/service"
	userrepo "gin-quickstart/internal/user/repository"
	"gin-quickstart/internal/user"
)

func main() {
	// 🔹 Logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	// 🔹 Check DB URL
	dbURL := config.GetDBUrl()
	if dbURL == "" {
		logger.Fatal("DATABASE_URL not set")
	}

	// 🔹 Connect DB
	database.Connect(dbURL)

	// 🔹 Initialize Redis client
	redisAddr := config.GetRedisURL()
	var redisOpt *redis.Options
	if strings.HasPrefix(redisAddr, "redis://") || strings.HasPrefix(redisAddr, "rediss://") {
		var err error
		redisOpt, err = redis.ParseURL(redisAddr)
		if err != nil {
			logger.Warn("Failed to parse Redis URL, using defaults", zap.Error(err))
			redisOpt = &redis.Options{Addr: "localhost:6379"}
		}
	} else {
		redisOpt = &redis.Options{
			Addr:     redisAddr,
			Password: "",
			DB:       0,
		}
	}

	redisClient := redis.NewClient(redisOpt)
	defer redisClient.Close()

	// Test Redis connection
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		logger.Warn("Redis connection failed, continuing without Redis", zap.Error(err))
		redisClient = nil
	}

	// 🔹 Router
	router := gin.Default()

	// 🔹 CORS (AWS friendly)
	router.Use(cors.New(cors.Config{
		AllowOrigins: config.GetCorsAllowedOrigins(),
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 🔹 Zap logging middleware
	router.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", time.Since(start)),
		)
	})

	router.Use(gin.Recovery())

	// 🔹 AWS Health check (MUST be at root or configured in EB)
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "event-store-backend"})
	})

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// 🔹 Initialize repositories
	userRepo2 := userrepo.NewUserRepository(database.Pool)
	subRepo := repository.NewSubscriptionRepository(database.Pool)
	eventRepo := repository.NewEventRepository(database.Pool)
	musicRepo := repository.NewMusicRepository(database.Pool)

	// 🔹 Initialize services
	tokenSvc := authservice.NewTokenService()

	// 🔹 Initialize event queue service
	eventQueueSvc := queueservice.NewEventQueueService(redisClient, eventRepo, subRepo, musicRepo)

	// 🔹 Start background services
	ctx := context.Background()
	go eventQueueSvc.ProcessEvents(ctx)

	// 🔹 API group
	v1 := router.Group("/api/v1")

	// 🔹 Auth routes
	auth.RegisterRoutes(v1, userRepo2)

	// 🔹 User routes
	user.RegisterRoutes(v1, userRepo2, tokenSvc)

	// 🔹 Catalog routes
	catalog.RegisterRoutes(v1)

	// 🔹 App API (Admin/Provider/Domain)
	appapi.RegisterRoutes(v1, tokenSvc, database.Pool, eventQueueSvc, subRepo, eventRepo, musicRepo)

	// 🔹 Port (AWS uses PORT env)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Common for Go on EB
	}

	logger.Info("Server starting", zap.String("port", port))
	
	// 🔹 Start server
	if err := router.Run(":" + port); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}
