package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// UserRepository handles persistence for user accounts and authentication.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository instance.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// NewPostgresUserRepository is a compatibility constructor used by main wiring.
func NewPostgresUserRepository(db *gorm.DB) *UserRepository {
	return NewUserRepository(db)
}

// Create implements ports.UserRepository.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	_ = ctx
	return r.CreateUser(user)
}

// FindByEmail implements ports.UserRepository.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	_ = ctx
	return r.GetUserByEmail(email)
}

// FindByID implements ports.UserRepository.
func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	_ = ctx
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}
	return r.GetUserByID(parsedID)
}

// UpdateLearningProfile updates the JSONB learning_profile for a user.
func (r *UserRepository) UpdateLearningProfile(ctx context.Context, id string, profile map[string]interface{}) error {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}
	res := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ? AND deleted_at IS NULL", parsedID).
		Update("learning_profile", profile)
	if res.Error != nil {
		return fmt.Errorf("failed to update learning profile: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// CreateUser persists a new user from OAuth provider data.
func (r *UserRepository) CreateUser(user *domain.User) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	result := r.db.Create(user)
	if result.Error != nil {
		return fmt.Errorf("failed to create user: %w", result.Error)
	}

	return nil
}

// GetUserByID retrieves a user by their UUID.
func (r *UserRepository) GetUserByID(id uuid.UUID) (*domain.User, error) {
	var user domain.User
	result := r.db.Where("id = ?", id).First(&user)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", result.Error)
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by their email (unique constraint).
func (r *UserRepository) GetUserByEmail(email string) (*domain.User, error) {
	var user domain.User
	result := r.db.Where("email = ?", email).First(&user)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email: %w", result.Error)
	}

	return &user, nil
}

// GetUserByGoogleID retrieves a user by their Google OAuth ID.
func (r *UserRepository) GetUserByGoogleID(googleID string) (*domain.User, error) {
	var user domain.User
	result := r.db.Where("google_id = ?", googleID).First(&user)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by google_id: %w", result.Error)
	}

	return &user, nil
}

// UpdateUser updates user profile information.
func (r *UserRepository) UpdateUser(user *domain.User) error {
	user.UpdatedAt = time.Now()
	result := r.db.Model(user).Updates(user)
	if result.Error != nil {
		return fmt.Errorf("failed to update user: %w", result.Error)
	}
	return nil
}

// DeleteUser soft-deletes a user (cascade to courses, materials, chunks via FK).
func (r *UserRepository) DeleteUser(userID uuid.UUID) error {
	result := r.db.Delete(&domain.User{}, "id = ?", userID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}
	return nil
}

// GetAllUsers retrieves all users (for admin purposes).
func (r *UserRepository) GetAllUsers() ([]*domain.User, error) {
	var users []*domain.User
	result := r.db.Find(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get all users: %w", result.Error)
	}
	return users, nil
}

// CountUsers returns the total number of registered users.
func (r *UserRepository) CountUsers() (int64, error) {
	var count int64
	result := r.db.Model(&domain.User{}).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to count users: %w", result.Error)
	}
	return count, nil
}
