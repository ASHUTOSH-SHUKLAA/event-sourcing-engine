package appapi

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"gin-quickstart/internal/middleware"
	"gin-quickstart/internal/repository"
	"gin-quickstart/internal/service"
	authservice "gin-quickstart/internal/auth/service"
)

func RegisterRoutes(rg *gin.RouterGroup, tokenSvc authservice.TokenService, pool *pgxpool.Pool,
	eventQueueSvc service.EventQueueService, subRepo repository.SubscriptionRepository,
	eventRepo repository.EventRepository, musicRepo repository.MusicRepository) {

	h := NewHandler(tokenSvc, eventQueueSvc, subRepo, eventRepo, musicRepo)

	app := rg.Group("")
	app.Use(middleware.AuthMiddleware(tokenSvc))
	{
		app.GET("/liked-songs", h.GetLikedSongs)
		app.POST("/liked-songs/:songId", h.LikeSong)
		app.DELETE("/liked-songs/:songId", h.UnlikeSong)



		app.GET("/player/state", h.GetPlayerState)
		app.POST("/player/events", h.PostPlayerEvent)
		app.GET("/player/play-counts", h.GetPlayCounts)

		app.GET("/subscriptions/current", h.GetCurrentSubscription)
		app.GET("/subscriptions/plans", h.GetPlans)
		app.POST("/subscriptions/upgrade", h.UpgradeSubscription)
		app.POST("/subscriptions/downgrade", h.DowngradeSubscription)
		app.POST("/subscriptions/pause", h.PauseSubscription)
		app.POST("/subscriptions/resume", h.ResumeSubscription)
		app.GET("/subscriptions/events", h.GetSubscriptionEvents)

		admin := app.Group("/admin")
		admin.Use(middleware.AdminOnly())
		{
			admin.GET("/users", h.GetAdminUsers)
			admin.GET("/users/:id/events", h.GetAdminUserEvents)
			admin.GET("/songs", h.GetAdminSongs)
			admin.GET("/subscriptions", h.GetAdminSubscriptions)
			admin.GET("/subscriptions/:aggregateId/events", h.GetAdminSubscriptionEvents)
			admin.GET("/metrics", h.GetAdminMetrics)
			admin.POST("/subscriptions/:aggregateId/upgrade", h.AdminUpgradeSubscription)
			admin.POST("/subscriptions/:aggregateId/downgrade", h.AdminDowngradeSubscription)
			admin.GET("/analytics", h.GetAdminAnalytics)
			admin.GET("/health", h.GetAdminHealth)
		}


	}
}
