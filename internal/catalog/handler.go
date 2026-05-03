package catalog

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetBanners(c *gin.Context) {
	items, err := h.svc.GetBanners(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": items}})
}

func (h *Handler) GetSongs(c *gin.Context) {
	tracks, err := h.svc.GetSongs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": tracks}})
}

func (h *Handler) GetPlaylists(c *gin.Context) {
	items, err := h.svc.GetPlaylists(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": items}})
}

func (h *Handler) SearchTracks(c *gin.Context) {
	tracks, err := h.svc.SearchTracks(c.Request.Context(), c.Query("q"))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"tracks": tracks}})
}
