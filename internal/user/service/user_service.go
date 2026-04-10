package service

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf8"

	"gin-quickstart/internal/user/model"
	"gin-quickstart/internal/user/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidPassword      = errors.New("invalid password")
	ErrPasswordTooShort     = errors.New("password must be at least 8 characters")
)

// UserService defines the business-logic contract for profile operations.
type UserService interface {
	GetProfile(ctx context.Context, userID string) (*model.User, error)
	UpdateProfile(ctx context.Context, userID string, displayName string) (*model.User, error)
	ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error
}

type userService struct {
	repo repository.UserRepository
}

// NewUserService constructs a UserService backed by the given repository.
func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

// GetProfile retrieves a user by ID. Requirements: 4.1
func (s *userService) GetProfile(ctx context.Context, userID string) (*model.User, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("userService.GetProfile: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// UpdateProfile updates the display name and returns the refreshed user. Requirements: 4.2
func (s *userService) UpdateProfile(ctx context.Context, userID string, displayName string) (*model.User, error) {
	if err := s.repo.UpdateDisplayName(ctx, userID, displayName); err != nil {
		return nil, fmt.Errorf("userService.UpdateProfile: %w", err)
	}
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("userService.UpdateProfile: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// ChangePassword validates the current password, then stores a new bcrypt hash.
func (s *userService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("userService.ChangePassword: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrInvalidPassword
	}
	if utf8.RuneCountInString(newPassword) < 8 {
		return ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return fmt.Errorf("userService.ChangePassword: %w", err)
	}
	if err := s.repo.UpdatePasswordHash(ctx, userID, string(hash)); err != nil {
		return fmt.Errorf("userService.ChangePassword: %w", err)
	}
	return nil
}
