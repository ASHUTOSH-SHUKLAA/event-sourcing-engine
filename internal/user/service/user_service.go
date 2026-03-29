package service

import (
	"context"
	"errors"
	"fmt"

	"gin-quickstart/internal/user/model"
	"gin-quickstart/internal/user/repository"
)

var ErrUserNotFound = errors.New("user not found")

// UserService defines the business-logic contract for profile operations.
type UserService interface {
	GetProfile(ctx context.Context, userID string) (*model.User, error)
	UpdateProfile(ctx context.Context, userID string, displayName string) (*model.User, error)
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
