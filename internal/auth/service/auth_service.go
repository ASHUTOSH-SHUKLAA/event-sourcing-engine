package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/mail"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"gin-quickstart/internal/user/model"
	"gin-quickstart/internal/user/repository"
)

// newUUID generates a random UUID v4 string using crypto/rand.
func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Sentinel errors returned by the auth service.
var (
	ErrEmailTaken       = errors.New("email already exists")
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrInvalidCreds     = errors.New("invalid credentials")
	ErrUserNotFound     = errors.New("user not found")
)

// TokenGenerator is a dependency interface so the auth service can issue tokens
// without being coupled to a concrete JWT implementation (wired in task 4).
type TokenGenerator interface {
	GenerateAccessToken(userID string) (string, error)
	GenerateRefreshToken(userID string) (string, error)
}

// AuthService defines the business-logic contract for registration and login.
type AuthService interface {
	Register(ctx context.Context, email, password string) (*model.User, error)
	Login(ctx context.Context, email, password string) (*model.TokenPair, error)
}

// authService is the concrete implementation of AuthService.
type authService struct {
	repo      repository.UserRepository
	tokenGen  TokenGenerator
}

// NewAuthService constructs an AuthService with the given repository and token generator.
func NewAuthService(repo repository.UserRepository, tokenGen TokenGenerator) AuthService {
	return &authService{repo: repo, tokenGen: tokenGen}
}

// Register validates inputs, hashes the password, and persists a new user.
// Requirements: 1.1, 1.2, 1.3, 1.4, 1.5
func (s *authService) Register(ctx context.Context, email, password string) (*model.User, error) {
	// Requirement 1.4 — validate email format
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, ErrInvalidEmail
	}

	// Requirement 1.3 — validate password length
	if utf8.RuneCountInString(password) < 8 {
		return nil, ErrPasswordTooShort
	}

	// Requirement 1.2 — check for duplicate email
	existing, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("auth.Register: lookup failed: %w", err)
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	// Requirement 1.5 — hash password with bcrypt cost 12; never store plaintext
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, fmt.Errorf("auth.Register: bcrypt failed: %w", err)
	}

	user := &model.User{
		ID:           newUUID(),
		Email:        email,
		PasswordHash: string(hash),
	}

	// Requirement 1.1 — persist the new user
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("auth.Register: persist failed: %w", err)
	}

	return user, nil
}

// Login verifies credentials and returns a TokenPair on success.
// Requirements: 2.1, 2.2, 2.3
func (s *authService) Login(ctx context.Context, email, password string) (*model.TokenPair, error) {
	// Requirement 2.3 — unknown email returns 401 (same message as wrong password)
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("auth.Login: lookup failed: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCreds
	}

	// Requirement 2.2 — wrong password returns 401
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCreds
	}

	// Requirement 2.1 — issue token pair
	accessToken, err := s.tokenGen.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("auth.Login: access token generation failed: %w", err)
	}

	refreshToken, err := s.tokenGen.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("auth.Login: refresh token generation failed: %w", err)
	}

	return &model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
