package appapi

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	mu sync.RWMutex

	likedSongs         map[string]map[string]LikedSong
	playlists          map[string][]Playlist
	player             map[string]PlayerState
	provider           map[string][]ProviderSong
	adminSubscriptions map[string]AdminSubscription
	adminEvents        map[string][]AdminSubscriptionEvent
}

type SongRef struct {
	ID string `json:"id"`
}

type LikedSong struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Duration string `json:"duration"`
	Artwork  string `json:"artwork"`
}

type Playlist struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	SongIDs []string  `json:"song_ids"`
	Created time.Time `json:"created_at"`
}

type PlayerState struct {
	CurrentSongID string `json:"current_song_id"`
	IsPlaying     bool   `json:"is_playing"`
	LastEventType string `json:"last_event_type"`
	UpdatedAt     string `json:"updated_at"`
}

type ProviderSong struct {
	Title       string    `json:"title"`
	Artist      string    `json:"artist"`
	Album       string    `json:"album"`
	Genre       string    `json:"genre"`
	ReleaseYear string    `json:"release_year"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

type AdminSubscription struct {
	AggregateID string    `json:"aggregate_id"`
	UserEmail   string    `json:"user_email"`
	Plan        string    `json:"plan"`
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"started_at"`
}

type AdminSubscriptionEvent struct {
	ID          string         `json:"id"`
	AggregateID string         `json:"aggregate_id"`
	EventType   string         `json:"event_type"`
	Version     int            `json:"version"`
	Payload     map[string]any `json:"payload"`
	CreatedAt   time.Time      `json:"created_at"`
}

type createPlaylistRequest struct {
	Name    string   `json:"name"`
	SongIDs []string `json:"song_ids"`
}

type likeSongRequest struct {
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Duration string `json:"duration"`
	Artwork  string `json:"artwork"`
}

type playerEventRequest struct {
	EventType string `json:"event_type"`
	SongID    string `json:"song_id"`
}

type providerUploadRequest struct {
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	Genre       string `json:"genre"`
	ReleaseYear string `json:"release_year"`
}

func seedAdminSubscriptions() (map[string]AdminSubscription, map[string][]AdminSubscriptionEvent) {
	now := time.Now()
	subscriptions := []AdminSubscription{
		{AggregateID: "sub-alice", UserEmail: "alice@example.com", Plan: "premium", Status: "active", StartedAt: now.AddDate(0, -2, 0)},
		{AggregateID: "sub-bob", UserEmail: "bob@example.com", Plan: "free", Status: "active", StartedAt: now.AddDate(0, -1, -15)},
		{AggregateID: "sub-carol", UserEmail: "carol@example.com", Plan: "premium", Status: "active", StartedAt: now.AddDate(0, -1, 0)},
		{AggregateID: "sub-dave", UserEmail: "dave@example.com", Plan: "free", Status: "cancelled", StartedAt: now.AddDate(0, 0, -20)},
		{AggregateID: "sub-eve", UserEmail: "eve@example.com", Plan: "free", Status: "active", StartedAt: now.AddDate(0, 0, -10)},
	}

	subMap := make(map[string]AdminSubscription, len(subscriptions))
	eventMap := make(map[string][]AdminSubscriptionEvent, len(subscriptions))
	for _, sub := range subscriptions {
		subMap[sub.AggregateID] = sub
		eventMap[sub.AggregateID] = []AdminSubscriptionEvent{
			{
				ID:          sub.AggregateID + "-created",
				AggregateID: sub.AggregateID,
				EventType:   "SubscriptionCreated",
				Version:     1,
				Payload: map[string]any{
					"plan":  sub.Plan,
					"price": map[string]int{"premium": 199, "free": 0}[sub.Plan],
				},
				CreatedAt: sub.StartedAt,
			},
		}
	}

	return subMap, eventMap
}

func NewHandler() *Handler {
	adminSubscriptions, adminEvents := seedAdminSubscriptions()
	return &Handler{
		likedSongs:         map[string]map[string]LikedSong{},
		playlists:          map[string][]Playlist{},
		player:             map[string]PlayerState{},
		provider:           map[string][]ProviderSong{},
		adminSubscriptions: adminSubscriptions,
		adminEvents:        adminEvents,
	}
}

func (h *Handler) GetLikedSongs(c *gin.Context) {
	userID := c.GetString("userID")
	h.mu.RLock()
	defer h.mu.RUnlock()

	store := h.likedSongs[userID]
	items := make([]LikedSong, 0, len(store))
	for _, item := range store {
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": items}})
}

func (h *Handler) LikeSong(c *gin.Context) {
	userID := c.GetString("userID")
	songID := c.Param("songId")
	if songID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "songId is required"})
		return
	}
	var req likeSongRequest
	_ = c.ShouldBindJSON(&req)

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.likedSongs[userID] == nil {
		h.likedSongs[userID] = map[string]LikedSong{}
	}
	h.likedSongs[userID][songID] = LikedSong{
		ID:       songID,
		Title:    req.Title,
		Artist:   req.Artist,
		Album:    req.Album,
		Duration: req.Duration,
		Artwork:  req.Artwork,
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"liked": true}})
}

func (h *Handler) UnlikeSong(c *gin.Context) {
	userID := c.GetString("userID")
	songID := c.Param("songId")

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.likedSongs[userID] != nil {
		delete(h.likedSongs[userID], songID)
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"liked": false}})
}

func (h *Handler) GetPlaylists(c *gin.Context) {
	userID := c.GetString("userID")
	h.mu.RLock()
	defer h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": h.playlists[userID]}})
}

func (h *Handler) GetPlaylistByID(c *gin.Context) {
	userID := c.GetString("userID")
	id := c.Param("id")
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, playlist := range h.playlists[userID] {
		if playlist.ID == id {
			c.JSON(http.StatusOK, gin.H{"data": playlist})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
}

func (h *Handler) CreatePlaylist(c *gin.Context) {
	userID := c.GetString("userID")
	var req createPlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	item := Playlist{
		ID:      time.Now().Format("20060102150405.000000"),
		Name:    req.Name,
		SongIDs: req.SongIDs,
		Created: time.Now(),
	}

	h.mu.Lock()
	h.playlists[userID] = append(h.playlists[userID], item)
	h.mu.Unlock()

	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *Handler) GetPlayerState(c *gin.Context) {
	userID := c.GetString("userID")
	h.mu.RLock()
	state, ok := h.player[userID]
	h.mu.RUnlock()
	if !ok {
		state = PlayerState{
			CurrentSongID: "",
			IsPlaying:     false,
			LastEventType: "",
			UpdatedAt:     "",
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": state})
}

func (h *Handler) PostPlayerEvent(c *gin.Context) {
	userID := c.GetString("userID")
	var req playerEventRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.EventType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	isPlaying := req.EventType == "SongPlayed" || req.EventType == "PLAY_TRACK"
	state := PlayerState{
		CurrentSongID: req.SongID,
		IsPlaying:     isPlaying,
		LastEventType: req.EventType,
		UpdatedAt:     time.Now().Format(time.RFC3339),
	}

	h.mu.Lock()
	h.player[userID] = state
	h.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{"data": state})
}

func (h *Handler) GetAdminUsers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": []any{}}})
}

func (h *Handler) GetAdminSongs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": []any{}}})
}

func (h *Handler) GetAdminSubscriptions(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	items := make([]AdminSubscription, 0, len(h.adminSubscriptions))
	for _, sub := range h.adminSubscriptions {
		items = append(items, sub)
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": items}})
}

func (h *Handler) GetAdminSubscriptionEvents(c *gin.Context) {
	aggregateID := c.Param("aggregateId")

	h.mu.RLock()
	defer h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{"data": h.adminEvents[aggregateID]})
}

func (h *Handler) GetAdminMetrics(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := len(h.adminSubscriptions)
	active := 0
	cancelled := 0
	mrr := 0
	for _, sub := range h.adminSubscriptions {
		if sub.Status == "active" {
			active++
		}
		if sub.Status == "cancelled" {
			cancelled++
		}
		if sub.Status == "active" && sub.Plan == "premium" {
			mrr += 199
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"total_subscribers": total,
		"mrr":               mrr,
		"active_count":      active,
		"cancelled_count":   cancelled,
	}})
}

func (h *Handler) GetAdminHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"status":        "healthy",
			"kafka":         "unknown",
			"api":           "running",
			"event_rate":    0,
			"consumer_lag":  0,
			"failed_events": []any{},
		},
	})
}

func (h *Handler) AdminUpgradeSubscription(c *gin.Context) {
	h.mutateAdminSubscription(c, "premium", "PlanUpgraded")
}

func (h *Handler) AdminDowngradeSubscription(c *gin.Context) {
	h.mutateAdminSubscription(c, "free", "PlanDowngraded")
}

func (h *Handler) mutateAdminSubscription(c *gin.Context, nextPlan, eventType string) {
	aggregateID := c.Param("aggregateId")

	h.mu.Lock()
	defer h.mu.Unlock()

	sub, ok := h.adminSubscriptions[aggregateID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}
	if sub.Plan == nextPlan {
		c.JSON(http.StatusOK, gin.H{"data": sub})
		return
	}

	prevPlan := sub.Plan
	sub.Plan = nextPlan
	sub.Status = "active"
	h.adminSubscriptions[aggregateID] = sub

	events := h.adminEvents[aggregateID]
	event := AdminSubscriptionEvent{
		ID:          aggregateID + "-" + time.Now().Format("20060102150405.000000"),
		AggregateID: aggregateID,
		EventType:   eventType,
		Version:     len(events) + 1,
		Payload: map[string]any{
			"from":     prevPlan,
			"to":       nextPlan,
			"price":    map[string]int{"premium": 199, "free": 0}[nextPlan],
			"currency": "INR",
		},
		CreatedAt: time.Now(),
	}
	h.adminEvents[aggregateID] = append(events, event)

	c.JSON(http.StatusOK, gin.H{"data": sub})
}

func (h *Handler) ProviderUploadSong(c *gin.Context) {
	userID := c.GetString("userID")
	var req providerUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Title == "" || req.Artist == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	item := ProviderSong{
		Title:       req.Title,
		Artist:      req.Artist,
		Album:       req.Album,
		Genre:       req.Genre,
		ReleaseYear: req.ReleaseYear,
		UploadedAt:  time.Now(),
	}

	h.mu.Lock()
	h.provider[userID] = append(h.provider[userID], item)
	h.mu.Unlock()

	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *Handler) ProviderSongs(c *gin.Context) {
	userID := c.GetString("userID")
	h.mu.RLock()
	defer h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": h.provider[userID]}})
}
