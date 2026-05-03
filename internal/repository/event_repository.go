package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EventRepository interface {
	Create(ctx context.Context, event *Event) error
	GetByAggregateID(ctx context.Context, aggregateID string, limit, offset int) ([]Event, error)
	ListRecent(ctx context.Context, limit int) ([]Event, error)
	GetAnalytics(ctx context.Context, since time.Time) (*EventAnalytics, error)
}

type Event struct {
	ID          string                 `json:"id"`
	AggregateID string                 `json:"aggregate_id"`
	UserID      string                 `json:"user_id"`
	EventType   string                 `json:"event_type"`
	Version     int                    `json:"version"`
	Payload     map[string]interface{} `json:"payload"`
	CreatedAt   time.Time              `json:"created_at"`
}

type EventAnalytics struct {
	TotalEvents     int            `json:"total_events"`
	EventsByType    map[string]int `json:"events_by_type"`
	EventsByHour    map[string]int `json:"events_by_hour"`
	UserActivity    map[string]int `json:"user_activity"`
	RecentEvents    []Event        `json:"recent_events"`
}

type PostgresEventRepository struct {
	pool *pgxpool.Pool
}

func NewEventRepository(pool *pgxpool.Pool) EventRepository {
	return &PostgresEventRepository{pool: pool}
}

func (r *PostgresEventRepository) Create(ctx context.Context, event *Event) error {
	payloadBytes, err := json.Marshal(event.Payload)
	if err != nil {
		return err
	}

	return r.pool.QueryRow(ctx, `
		INSERT INTO subscription_events (aggregate_id, user_id, event_type, version, payload)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, created_at`,
		event.AggregateID, event.UserID, event.EventType, event.Version, payloadBytes,
	).Scan(&event.ID, &event.CreatedAt)
}

func (r *PostgresEventRepository) GetByAggregateID(ctx context.Context, aggregateID string, limit, offset int) ([]Event, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, aggregate_id::text, user_id::text, event_type, version, payload, created_at
		FROM subscription_events
		WHERE aggregate_id = $1
		ORDER BY version ASC
		LIMIT $2 OFFSET $3`, aggregateID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		var payload []byte
		if err := rows.Scan(&event.ID, &event.AggregateID, &event.UserID, &event.EventType, &event.Version, &payload, &event.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(payload, &event.Payload)
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *PostgresEventRepository) ListRecent(ctx context.Context, limit int) ([]Event, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, aggregate_id::text, user_id::text, event_type, version, payload, created_at
		FROM subscription_events
		ORDER BY created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		var payload []byte
		if err := rows.Scan(&event.ID, &event.AggregateID, &event.UserID, &event.EventType, &event.Version, &payload, &event.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(payload, &event.Payload)
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *PostgresEventRepository) GetAnalytics(ctx context.Context, since time.Time) (*EventAnalytics, error) {
	analytics := &EventAnalytics{
		EventsByType: make(map[string]int),
		EventsByHour: make(map[string]int),
		UserActivity: make(map[string]int),
	}

	// Get total events
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM subscription_events WHERE created_at >= $1`, since).Scan(&analytics.TotalEvents)
	if err != nil {
		return nil, err
	}

	// Get events by type
	rows, err := r.pool.Query(ctx, `
		SELECT event_type, COUNT(*)::int
		FROM subscription_events
		WHERE created_at >= $1
		GROUP BY event_type`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var eventType string
		var count int
		if err := rows.Scan(&eventType, &count); err != nil {
			return nil, err
		}
		analytics.EventsByType[eventType] = count
	}

	// Get events by hour
	hourRows, err := r.pool.Query(ctx, `
		SELECT DATE_TRUNC('hour', created_at)::text, COUNT(*)::int
		FROM subscription_events
		WHERE created_at >= $1
		GROUP BY DATE_TRUNC('hour', created_at)
		ORDER BY DATE_TRUNC('hour', created_at) DESC`, since)
	if err != nil {
		return nil, err
	}
	defer hourRows.Close()

	for hourRows.Next() {
		var hour string
		var count int
		if err := hourRows.Scan(&hour, &count); err != nil {
			return nil, err
		}
		analytics.EventsByHour[hour] = count
	}

	// Get user activity
	userRows, err := r.pool.Query(ctx, `
		SELECT user_id::text, COUNT(*)::int
		FROM subscription_events
		WHERE created_at >= $1
		GROUP BY user_id
		ORDER BY COUNT(*) DESC
		LIMIT 10`, since)
	if err != nil {
		return nil, err
	}
	defer userRows.Close()

	for userRows.Next() {
		var userID string
		var count int
		if err := userRows.Scan(&userID, &count); err != nil {
			return nil, err
		}
		analytics.UserActivity[userID] = count
	}

	// Get recent events
	analytics.RecentEvents, err = r.ListRecent(ctx, 50)
	if err != nil {
		return nil, err
	}

	return analytics, nil
}