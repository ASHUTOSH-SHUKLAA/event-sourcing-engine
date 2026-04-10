package appapi

import (
	"github.com/gin-gonic/gin"

	authservice "gin-quickstart/internal/auth/service"
	"gin-quickstart/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, tokenSvc authservice.TokenService) {
	h := NewHandler()

	app := rg.Group("")
	app.Use(middleware.AuthMiddleware(tokenSvc))
	{
		app.GET("/liked-songs", h.GetLikedSongs)
		app.POST("/liked-songs/:songId", h.LikeSong)
		app.DELETE("/liked-songs/:songId", h.UnlikeSong)

		app.GET("/playlists", h.GetPlaylists)
		app.GET("/playlists/:id", h.GetPlaylistByID)
		app.POST("/playlists", h.CreatePlaylist)

		app.GET("/player/state", h.GetPlayerState)
		app.POST("/player/events", h.PostPlayerEvent)

		app.GET("/admin/users", h.GetAdminUsers)
		app.GET("/admin/songs", h.GetAdminSongs)
		app.GET("/admin/subscriptions", h.GetAdminSubscriptions)
		app.GET("/admin/subscriptions/:aggregateId/events", h.GetAdminSubscriptionEvents)
		app.GET("/admin/metrics", h.GetAdminMetrics)
		app.POST("/admin/subscriptions/:aggregateId/upgrade", h.AdminUpgradeSubscription)
		app.POST("/admin/subscriptions/:aggregateId/downgrade", h.AdminDowngradeSubscription)
		app.GET("/admin/health", h.GetAdminHealth)

		app.POST("/provider/upload", h.ProviderUploadSong)
		app.GET("/provider/songs", h.ProviderSongs)
	}
}
