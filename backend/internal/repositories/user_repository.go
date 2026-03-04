package repositories

import (
	"context"
	"errors"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"gorm.io/gorm"
)

// PostgresUserRepository is the concrete PostgreSQL implementation of ports.UserRepository.
// It lives in the infrastructure layer — the use cases never import this package directly.
type PostgresUserRepository struct {
	db *gorm.DB
}

// NewPostgresUserRepository creates an instance of the GORM-backed user repository.
func NewPostgresUserRepository(db *gorm.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	result := r.db.WithContext(ctx).Where("email = ? AND deleted_at IS NULL", email).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil // Returning nil, nil signals "not found" (not an error state)
	}
	return &user, result.Error
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	result := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, result.Error
}
