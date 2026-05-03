package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SubscriptionRepository interface {
	Create(ctx context.Context, userID string) error
	GetByUserID(ctx context.Context, userID string) (*Subscription, error)
	Update(ctx context.Context, userID string, plan string, status string, price int, periodEnd *time.Time, cancelAtEnd bool) error
	PauseSubscription(ctx context.Context, userID string, daysRemaining int) error
	ResumeSubscription(ctx context.Context, userID string, newPeriodEnd time.Time) error
	ListAll(ctx context.Context, limit, offset int) ([]Subscription, error)
	GetMetrics(ctx context.Context) (*SubscriptionMetrics, error)
	IncrementVersion(ctx context.Context, userID string) (int, error)
}

type Subscription struct {
	UserID            string     `json:"user_id"`
	Plan              string     `json:"plan"`
	Status            string     `json:"status"`
	Price             int        `json:"price"`
	Version           int        `json:"version"`
	StartedAt         time.Time  `json:"started_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	CurrentPeriodEnd  time.Time  `json:"current_period_end"`
	CancelAtPeriodEnd bool       `json:"cancel_at_period_end"`
	TotalRevenue      int        `json:"total_revenue"`
	PausedAt          *time.Time `json:"paused_at,omitempty"`
	PauseDaysRemaining *int      `json:"pause_days_remaining,omitempty"`
	Name              string     `json:"name"`
	Email             string     `json:"email"`
}

type SubscriptionMetrics struct {
	TotalSubscribers int `json:"total_subscribers"`
	ActiveCount      int `json:"active_count"`
	CancelledCount   int `json:"cancelled_count"`
	MRR              int `json:"mrr"`
	TotalLTV         int `json:"total_ltv"`
}

type PostgresSubscriptionRepository struct {
	pool *pgxpool.Pool
}

func NewSubscriptionRepository(pool *pgxpool.Pool) SubscriptionRepository {
	return &PostgresSubscriptionRepository{pool: pool}
}

func (r *PostgresSubscriptionRepository) Create(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO subscriptions (user_id, plan, status, price, version)
		VALUES ($1, 'free', 'active', 0, 1)
		ON CONFLICT (user_id) DO NOTHING`, userID)
	return err
}

func (r *PostgresSubscriptionRepository) GetByUserID(ctx context.Context, userID string) (*Subscription, error) {
	if err := r.Create(ctx, userID); err != nil {
		return nil, err
	}

	var sub Subscription
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, plan, status, price, version, started_at, updated_at,
		       COALESCE(current_period_end, now() + interval '30 days'),
		       COALESCE(cancel_at_period_end, false),
		       COALESCE(total_revenue_cents, 0),
		       paused_at,
		       pause_days_remaining
		FROM subscriptions
		WHERE user_id = $1`, userID).Scan(
		&sub.UserID, &sub.Plan, &sub.Status, &sub.Price, &sub.Version, &sub.StartedAt, &sub.UpdatedAt,
		&sub.CurrentPeriodEnd, &sub.CancelAtPeriodEnd, &sub.TotalRevenue,
		&sub.PausedAt, &sub.PauseDaysRemaining,
	)
	if err != nil {
		return nil, err
	}

	// Logical Expiry: If the subscription has expired and was marked for cancellation,
	// transparently downgrade them to free.
	// Skip this check if the subscription is paused — time is frozen when paused.
	if sub.Plan == "premium" && sub.CancelAtPeriodEnd && sub.Status != "paused" && time.Now().After(sub.CurrentPeriodEnd) {
		err = r.Update(ctx, userID, "free", "active", 0, nil, false)
		if err != nil {
			return nil, err
		}
		// Refresh the object
		return r.GetByUserID(ctx, userID)
	}

	return &sub, nil
}

func (r *PostgresSubscriptionRepository) Update(ctx context.Context, userID string, plan string, status string, price int, periodEnd *time.Time, cancelAtEnd bool) error {
	var err error
	if periodEnd != nil {
		_, err = r.pool.Exec(ctx, `
			UPDATE subscriptions
			SET plan = $2, status = $3, price = $4, version = version + 1, updated_at = NOW(),
				current_period_end = $5, cancel_at_period_end = $6,
				total_revenue_cents = total_revenue_cents + $4
			WHERE user_id = $1`, userID, plan, status, price, *periodEnd, cancelAtEnd)
	} else {
		_, err = r.pool.Exec(ctx, `
			UPDATE subscriptions
			SET plan = $2, status = $3, price = $4, version = version + 1, updated_at = NOW(),
				cancel_at_period_end = $5
			WHERE user_id = $1`, userID, plan, status, price, cancelAtEnd)
	}
	return err
}

func (r *PostgresSubscriptionRepository) PauseSubscription(ctx context.Context, userID string, daysRemaining int) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE subscriptions
		SET status = 'paused',
		    paused_at = NOW(),
		    pause_days_remaining = $2,
		    version = version + 1,
		    updated_at = NOW()
		WHERE user_id = $1 AND plan = 'premium' AND status = 'active'`, userID, daysRemaining)
	return err
}

func (r *PostgresSubscriptionRepository) ResumeSubscription(ctx context.Context, userID string, newPeriodEnd time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE subscriptions
		SET status = 'active',
		    current_period_end = $2,
		    paused_at = NULL,
		    pause_days_remaining = NULL,
		    version = version + 1,
		    updated_at = NOW()
		WHERE user_id = $1 AND status = 'paused'`, userID, newPeriodEnd)
	return err
}

func (r *PostgresSubscriptionRepository) ListAll(ctx context.Context, limit, offset int) ([]Subscription, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.user_id, s.plan, s.status, s.price, s.version, s.started_at, s.updated_at,
		       COALESCE(s.current_period_end, now()), COALESCE(s.cancel_at_period_end, false),
		       COALESCE(s.total_revenue_cents, 0), s.paused_at, s.pause_days_remaining,
		       u.display_name, u.email
		FROM subscriptions s
		JOIN users u ON u.id = s.user_id
		ORDER BY s.updated_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(
			&sub.UserID, &sub.Plan, &sub.Status, &sub.Price, &sub.Version, &sub.StartedAt, &sub.UpdatedAt,
			&sub.CurrentPeriodEnd, &sub.CancelAtPeriodEnd, &sub.TotalRevenue,
			&sub.PausedAt, &sub.PauseDaysRemaining, &sub.Name, &sub.Email,
		); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

func (r *PostgresSubscriptionRepository) GetMetrics(ctx context.Context) (*SubscriptionMetrics, error) {
	var metrics SubscriptionMetrics
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int,
			COUNT(*) FILTER (WHERE status = 'active')::int,
			COUNT(*) FILTER (WHERE plan = 'free' AND total_revenue_cents > 0)::int,
			COALESCE(SUM(price) FILTER (WHERE status = 'active' AND plan = 'premium'), 0)::int,
			COALESCE(SUM(total_revenue_cents), 0)::int
		FROM subscriptions`).Scan(&metrics.TotalSubscribers, &metrics.ActiveCount, &metrics.CancelledCount, &metrics.MRR, &metrics.TotalLTV)
	if err != nil {
		return nil, err
	}
	return &metrics, nil
}

func (r *PostgresSubscriptionRepository) IncrementVersion(ctx context.Context, userID string) (int, error) {
	var newVersion int
	err := r.pool.QueryRow(ctx, `
		UPDATE subscriptions 
		SET version = version + 1, updated_at = NOW() 
		WHERE user_id = $1
		RETURNING version`, userID).Scan(&newVersion)
	return newVersion, err
}
