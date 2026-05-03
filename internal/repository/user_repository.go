package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash, displayName string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, id string, displayName string) error
	ListAll(ctx context.Context) ([]User, error)
}

type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	PasswordHash string   `json:"password_hash"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &PostgresUserRepository{pool: pool}
}

func (r *PostgresUserRepository) Create(ctx context.Context, email, passwordHash, displayName string) (*User, error) {
	var user User
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, display_name)
		VALUES ($1, $2, $3)
		RETURNING id::text, email, password_hash, display_name, created_at, updated_at`,
		email, passwordHash, displayName).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*User, error) {
	var user User
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, email, password_hash, display_name, created_at, updated_at
		FROM users
		WHERE id = $1`, id).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, email, password_hash, display_name, created_at, updated_at
		FROM users
		WHERE email = $1`, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *PostgresUserRepository) Update(ctx context.Context, id string, displayName string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET display_name = $2, updated_at = NOW()
		WHERE id = $1`, id, displayName)
	return err
}

func (r *PostgresUserRepository) ListAll(ctx context.Context) ([]User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, email, display_name, created_at
		FROM users
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Email, &user.DisplayName, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}