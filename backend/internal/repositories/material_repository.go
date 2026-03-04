package repositories

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// PostgresMaterialRepository is the GORM-backed persistence implementation for Materials.
// Lives in the infrastructure layer — use cases only see the MaterialRepository port.
type PostgresMaterialRepository struct {
	db *gorm.DB
}

// NewPostgresMaterialRepository creates an instance of the GORM material repository.
func NewPostgresMaterialRepository(db *gorm.DB) *PostgresMaterialRepository {
	return &PostgresMaterialRepository{db: db}
}

func (r *PostgresMaterialRepository) Create(ctx context.Context, material *domain.Material) error {
	return r.db.WithContext(ctx).Create(material).Error
}

func (r *PostgresMaterialRepository) FindByID(ctx context.Context, id string) (*domain.Material, error) {
	var m domain.Material
	result := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &m, result.Error
}

func (r *PostgresMaterialRepository) FindByTopic(ctx context.Context, topicID string) ([]domain.Material, error) {
	var materials []domain.Material
	result := r.db.WithContext(ctx).
		Where("topic_id = ? AND deleted_at IS NULL", topicID).
		Order("created_at ASC").
		Find(&materials)
	return materials, result.Error
}

// UpdateStatus updates the processing status and optional extracted text for a material.
func (r *PostgresMaterialRepository) UpdateStatus(ctx context.Context, materialID string, status domain.MaterialStatus, extractedText string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if extractedText != "" {
		updates["extracted_text"] = extractedText
	}
	return r.db.WithContext(ctx).
		Model(&domain.Material{}).
		Where("id = ?", materialID).
		Updates(updates).Error
}
