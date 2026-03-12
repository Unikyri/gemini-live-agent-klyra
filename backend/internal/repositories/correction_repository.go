package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// CorrectionRepository persists MaterialCorrection records in Postgres.
type CorrectionRepository struct {
	db *gorm.DB
}

func NewPostgresCorrectionRepository(db *gorm.DB) *CorrectionRepository {
	return &CorrectionRepository{db: db}
}

func (r *CorrectionRepository) Create(ctx context.Context, correction *domain.MaterialCorrection) error {
	if correction == nil {
		return fmt.Errorf("correction is nil")
	}
	if correction.MaterialID == uuid.Nil {
		return fmt.Errorf("material_id is required")
	}
	if correction.BlockIndex < 0 {
		return fmt.Errorf("block_index must be >= 0")
	}
	if correction.OriginalText == "" || correction.CorrectedText == "" {
		return fmt.Errorf("original_text and corrected_text are required")
	}

	now := time.Now()
	if correction.ID == uuid.Nil {
		correction.ID = uuid.New()
	}
	if correction.CreatedAt.IsZero() {
		correction.CreatedAt = now
	}
	correction.UpdatedAt = now

	// UPSERT by (material_id, block_index)
	// Keep the latest corrected_text.
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing domain.MaterialCorrection
		err := tx.
			Where("material_id = ? AND block_index = ?", correction.MaterialID, correction.BlockIndex).
			First(&existing).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("find existing correction: %w", err)
		}
		if err == gorm.ErrRecordNotFound {
			if err := tx.Create(correction).Error; err != nil {
				return fmt.Errorf("create correction: %w", err)
			}
			return nil
		}

		updates := map[string]interface{}{
			"chunk_id":        correction.ChunkID,
			"original_text":   correction.OriginalText,
			"corrected_text":  correction.CorrectedText,
			"updated_at":      now,
		}
		if err := tx.Model(&domain.MaterialCorrection{}).
			Where("id = ?", existing.ID).
			Updates(updates).Error; err != nil {
			return fmt.Errorf("update correction: %w", err)
		}
		// Return the latest persisted record (preserve ID and timestamps).
		correction.ID = existing.ID
		correction.CreatedAt = existing.CreatedAt
		return nil
	})
}

func (r *CorrectionRepository) FindByMaterial(ctx context.Context, materialID string) ([]domain.MaterialCorrection, error) {
	id, err := uuid.Parse(materialID)
	if err != nil {
		return nil, fmt.Errorf("invalid material_id: %w", err)
	}
	var items []domain.MaterialCorrection
	if err := r.db.WithContext(ctx).
		Where("material_id = ?", id).
		Order("block_index ASC").
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("find corrections by material: %w", err)
	}
	return items, nil
}

func (r *CorrectionRepository) FindByChunkIDs(ctx context.Context, chunkIDs []string) ([]domain.MaterialCorrection, error) {
	if len(chunkIDs) == 0 {
		return []domain.MaterialCorrection{}, nil
	}
	ids := make([]uuid.UUID, 0, len(chunkIDs))
	for _, s := range chunkIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("invalid chunk_id %q: %w", s, err)
		}
		ids = append(ids, id)
	}
	var items []domain.MaterialCorrection
	if err := r.db.WithContext(ctx).
		Where("chunk_id IN ?", ids).
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("find corrections by chunk ids: %w", err)
	}
	return items, nil
}

func (r *CorrectionRepository) Delete(ctx context.Context, correctionID string) error {
	id, err := uuid.Parse(correctionID)
	if err != nil {
		return fmt.Errorf("invalid correction_id: %w", err)
	}
	res := r.db.WithContext(ctx).Delete(&domain.MaterialCorrection{}, "id = ?", id)
	if res.Error != nil {
		return fmt.Errorf("delete correction: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

