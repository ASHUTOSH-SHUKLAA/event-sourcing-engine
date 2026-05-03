package appapi

import "time"

const premiumPrice = 199

type Subscription struct {
	Plan              string    `json:"plan"`
	Status            string    `json:"status"`
	Price             int       `json:"price"`
	Version           int       `json:"version"`
	StartedAt         time.Time `json:"started_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	CurrentPeriodEnd  time.Time `json:"current_period_end"`
	CancelAtPeriodEnd bool      `json:"cancel_at_period_end"`
	TotalRevenue      int       `json:"total_revenue"`
}

type LikedSong struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Duration string `json:"duration"`
	Artwork  string `json:"artwork"`
}



type PlayerState struct {
	CurrentSongID string `json:"current_song_id"`
	IsPlaying     bool   `json:"is_playing"`
	LastEventType string `json:"last_event_type"`
	UpdatedAt     string `json:"updated_at"`
}

type ProviderSong struct {
	ID          string    `json:"id"`
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

type AdminUser struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Plan      string    `json:"plan"`
	CreatedAt time.Time `json:"created_at"`
}

type AdminSong struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
	Plays  int    `json:"plays"`
	Likes  int    `json:"likes"`
}

type AdminMetrics struct {
	TotalSubscribers int `json:"total_subscribers"`
	MRR              int `json:"mrr"`
	ActiveCount      int `json:"active_count"`
	CancelledCount   int `json:"cancelled_count"`
	TotalLTV         int `json:"total_ltv"`
}



type likeSongRequest struct {
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Duration string `json:"duration"`
	Artwork  string `json:"artwork"`
}

type playerEventRequest struct {
	EventType string                 `json:"event_type"`
	SongID    string                 `json:"song_id"`
	Payload   map[string]interface{} `json:"payload"`
}

type providerUploadRequest struct {
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	Genre       string `json:"genre"`
	ReleaseYear string `json:"release_year"`
}
