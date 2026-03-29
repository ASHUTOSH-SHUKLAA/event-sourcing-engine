package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gin-quickstart/internal/user/model"
)

// UserRepository defines the data access contract for User records.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id string) (*model.User, error)
	UpdateDisplayName(ctx context.Context, id string, displayName string) error
}

// pgxUserRepository is the PostgreSQL implementation backed by a pgxpool.
type pgxUserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository constructs a UserRepository backed by the given pgx pool.
func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &pgxUserRepository{pool: pool}
}

// Create inserts a new user row. The caller is responsible for setting ID and
// PasswordHash before calling this method.
func (r *pgxUserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, display_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.DisplayName,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("repository.Create: %w", err)
	}
	return nil
}

// FindByEmail retrieves a user by email address.
// Returns nil, nil when no row is found so callers can distinguish "not found"
// from a real database error.
func (r *pgxUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, display_name, created_at, updated_at
		FROM users
		WHERE email = $1`

	user := &model.User{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("repository.FindByEmail: %w", err)
	}
	return user, nil
}

// FindByID retrieves a user by UUID.
// Returns nil, nil when no row is found.
func (r *pgxUserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, display_name, created_at, updated_at
		FROM users
		WHERE id = $1`

	user := &model.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("repository.FindByID: %w", err)
	}
	return user, nil
}

// UpdateDisplayName sets a new display_name for the given user ID and bumps updated_at.
func (r *pgxUserRepository) UpdateDisplayName(ctx context.Context, id string, displayName string) error {
	query := `
		UPDATE users
		SET display_name = $1, updated_at = NOW()
		WHERE id = $2`

	tag, err := r.pool.Exec(ctx, query, displayName, id)
	if err != nil {
		return fmt.Errorf("repository.UpdateDisplayName: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("repository.UpdateDisplayName: user %s not found", id)
	}
	return nil
}
