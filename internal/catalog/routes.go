package catalog

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup) {
	svc := NewService()
	h := NewHandler(svc)

	group := rg.Group("/catalog")
	{
		group.GET("/songs", h.GetSongs)
		group.GET("/playlists", h.GetPlaylists)
		group.GET("/banners", h.GetBanners)
		group.GET("/search", h.SearchTracks)
	}
}
