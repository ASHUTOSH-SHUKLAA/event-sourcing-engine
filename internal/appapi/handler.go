package appapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	authservice "gin-quickstart/internal/auth/service"
	"gin-quickstart/internal/repository"
	"gin-quickstart/internal/service"
)

type Handler struct {
	tokenSvc      authservice.TokenService
	eventQueueSvc service.EventQueueService
	subRepo       repository.SubscriptionRepository
	eventRepo     repository.EventRepository
	musicRepo     repository.MusicRepository
}

func NewHandler(tokenSvc authservice.TokenService, eventQueueSvc service.EventQueueService,
	subRepo repository.SubscriptionRepository, eventRepo repository.EventRepository,
	musicRepo repository.MusicRepository) *Handler {

	return &Handler{
		tokenSvc:      tokenSvc,
		eventQueueSvc: eventQueueSvc,
		subRepo:       subRepo,
		eventRepo:     eventRepo,
		musicRepo:     musicRepo,
	}
}

func respondError(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{"error": err.Error()})
}

func (h *Handler) GetCurrentSubscription(c *gin.Context) {
	sub, err := h.subRepo.GetByUserID(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sub})
}

func (h *Handler) GetPlans(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []gin.H{
		{"id": "free", "name": "Free", "price": 0},
		{"id": "premium", "name": "Premium", "price": premiumPrice},
	}})
}

func (h *Handler) UpgradeSubscription(c *gin.Context) {
	h.changeSubscription(c, "premium", "PlanUpgraded")
}

func (h *Handler) DowngradeSubscription(c *gin.Context) {
	h.changeSubscription(c, "free", "PlanDowngraded")
}

func (h *Handler) PauseSubscription(c *gin.Context) {
	userID := c.GetString("userID")
	event := &service.QueuedEvent{
		UserID:    userID,
		EventType: "SubscriptionPaused",
		Payload:   map[string]interface{}{"reason": "user_initiated"},
	}
	if err := h.eventQueueSvc.Enqueue(c.Request.Context(), event); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	updated, err := h.subRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *Handler) ResumeSubscription(c *gin.Context) {
	userID := c.GetString("userID")
	event := &service.QueuedEvent{
		UserID:    userID,
		EventType: "SubscriptionResumed",
		Payload:   map[string]interface{}{"reason": "user_initiated"},
	}
	if err := h.eventQueueSvc.Enqueue(c.Request.Context(), event); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	updated, err := h.subRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *Handler) changeSubscription(c *gin.Context, nextPlan, eventType string) {
	userID := c.GetString("userID")
	current, err := h.subRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	if current.Plan == nextPlan {
		c.JSON(http.StatusOK, gin.H{"data": current})
		return
	}

	event := &service.QueuedEvent{
		UserID:    userID,
		EventType: eventType,
		Payload: map[string]interface{}{
			"from":     current.Plan,
			"to":       nextPlan,
			"price":    map[string]int{"premium": premiumPrice, "free": 0}[nextPlan],
			"currency": "INR",
		},
	}

	if err := h.eventQueueSvc.Enqueue(c.Request.Context(), event); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	// Return updated subscription
	updated, err := h.subRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *Handler) GetSubscriptionEvents(c *gin.Context) {
	userID := c.GetString("userID")
	if err := h.subRepo.Create(c.Request.Context(), userID); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	limit, offset := getPagination(c)
	events, err := h.eventRepo.GetByAggregateID(c.Request.Context(), userID, limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": events})
}

func (h *Handler) GetLikedSongs(c *gin.Context) {
	items, err := h.musicRepo.ListLikedSongs(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": items}})
}

func (h *Handler) LikeSong(c *gin.Context) {
	songID := c.Param("songId")
	if songID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "songId is required"})
		return
	}
	var req likeSongRequest
	_ = c.ShouldBindJSON(&req)

	err := h.eventQueueSvc.Enqueue(c.Request.Context(), &service.QueuedEvent{
		UserID:    c.GetString("userID"),
		EventType: "SongLiked",
		SongID:    songID,
		Payload: map[string]interface{}{
			"title":    req.Title,
			"artist":   req.Artist,
			"album":    req.Album,
			"duration": req.Duration,
			"artwork":  req.Artwork,
		},
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"liked": true}})
}

func (h *Handler) UnlikeSong(c *gin.Context) {
	err := h.eventQueueSvc.Enqueue(c.Request.Context(), &service.QueuedEvent{
		UserID:    c.GetString("userID"),
		EventType: "SongUnliked",
		SongID:    c.Param("songId"),
		Payload:   map[string]interface{}{},
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"liked": false}})
}



func (h *Handler) GetPlayerState(c *gin.Context) {
	state, err := h.musicRepo.GetPlayerState(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": state})
}

func (h *Handler) PostPlayerEvent(c *gin.Context) {
	var req playerEventRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.EventType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	payload := req.Payload
	if payload == nil {
		payload = map[string]interface{}{"song_id": req.SongID}
	} else {
		payload["song_id"] = req.SongID
	}

	event := &service.QueuedEvent{
		UserID:    c.GetString("userID"),
		EventType: req.EventType,
		SongID:    req.SongID,
		Payload:   payload,
	}

	if err := h.eventQueueSvc.Enqueue(c.Request.Context(), event); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"event_type": req.EventType, "song_id": req.SongID}})
}

func (h *Handler) GetPlayCounts(c *gin.Context) {
	counts, err := h.musicRepo.ListPlayCounts(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": counts})
}

func getPagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit
	return limit, offset
}

func (h *Handler) GetAdminUsers(c *gin.Context) {
	limit, offset := getPagination(c)
	users, err := h.subRepo.ListAll(c.Request.Context(), limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	// Convert to admin user format
	adminUsers := make([]AdminUser, len(users))
	for i, user := range users {
		adminUsers[i] = AdminUser{
			ID:        user.UserID,
			Email:     user.Email,
			Name:      user.Name,
			Plan:      user.Plan,
			CreatedAt: user.StartedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": adminUsers}})
}

func (h *Handler) GetAdminUserEvents(c *gin.Context) {
	userID := c.Param("id")
	limit, offset := getPagination(c)

	events, err := h.eventRepo.GetByAggregateID(c.Request.Context(), userID, limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": events})
}

func (h *Handler) GetAdminSongs(c *gin.Context) {
	limit, offset := getPagination(c)
	items, err := h.musicRepo.ListAdminSongs(c.Request.Context(), limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": items}})
}

func (h *Handler) GetAdminSubscriptions(c *gin.Context) {
	limit, offset := getPagination(c)
	items, err := h.subRepo.ListAll(c.Request.Context(), limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": items}})
}

func (h *Handler) GetAdminSubscriptionEvents(c *gin.Context) {
	limit, offset := getPagination(c)
	events, err := h.eventRepo.GetByAggregateID(c.Request.Context(), c.Param("aggregateId"), limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": events})
}

func (h *Handler) GetAdminMetrics(c *gin.Context) {
	metrics, err := h.subRepo.GetMetrics(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": metrics})
}

func (h *Handler) GetAdminHealth(c *gin.Context) {
	queueDepth := h.eventQueueSvc.GetQueueLength()
	processedCount := h.eventQueueSvc.GetProcessedCount()

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"status":           "healthy",
		"queue":            "redis",
		"api":              "running",
		"event_rate":       queueDepth,
		"consumer_lag":     queueDepth,
		"processed_events": processedCount,
		"failed_events":    []interface{}{},
	}})
}

func (h *Handler) GetAdminAnalytics(c *gin.Context) {
	analytics, err := h.eventRepo.GetAnalytics(c.Request.Context(), time.Now().Add(-24*time.Hour))
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	musicAnalytics, err := h.musicRepo.GetAnalytics(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"events": analytics,
		"music":  musicAnalytics,
	}})
}

func (h *Handler) AdminUpgradeSubscription(c *gin.Context) {
	h.adminChangeSubscription(c, "premium", "PlanUpgraded")
}

func (h *Handler) AdminDowngradeSubscription(c *gin.Context) {
	h.adminChangeSubscription(c, "free", "PlanDowngraded")
}

func (h *Handler) adminChangeSubscription(c *gin.Context, nextPlan, eventType string) {
	userID := c.Param("aggregateId")
	current, err := h.subRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	if current.Plan == nextPlan {
		c.JSON(http.StatusOK, gin.H{"data": current})
		return
	}

	if err := h.eventQueueSvc.Enqueue(c.Request.Context(), &service.QueuedEvent{
		UserID:    userID,
		EventType: eventType,
		Payload: map[string]interface{}{
			"from":     current.Plan,
			"to":       nextPlan,
			"price":    map[string]int{"premium": premiumPrice, "free": 0}[nextPlan],
			"currency": "INR",
		},
	}); err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	// Return updated subscription
	updated, err := h.subRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

