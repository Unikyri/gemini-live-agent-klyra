package repositories

import (
	"context"
	"fmt"
	"mime/multipart"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// MaterialRepository handles persistence for educational materials and their metadata.
type MaterialRepository struct {
	db             *gorm.DB
	storageService StorageService
	textExtractor  TextExtractor
}

// NewMaterialRepository creates a new material repository instance.
func NewMaterialRepository(
	db *gorm.DB,
	storageService StorageService,
	textExtractor TextExtractor,
) *MaterialRepository {
	return &MaterialRepository{
		db:             db,
		storageService: storageService,
		textExtractor:  textExtractor,
	}
}

// NewPostgresMaterialRepository is a compatibility constructor used by main wiring.
func NewPostgresMaterialRepository(db *gorm.DB) *MaterialRepository {
	return &MaterialRepository{db: db}
}

// Create implements ports.MaterialRepository.
func (r *MaterialRepository) Create(ctx context.Context, material *domain.Material) error {
	result := r.db.WithContext(ctx).Create(material)
	if result.Error != nil {
		return fmt.Errorf("failed to create material: %w", result.Error)
	}
	return nil
}

// FindByID implements ports.MaterialRepository.
func (r *MaterialRepository) FindByID(ctx context.Context, id string) (*domain.Material, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid material id: %w", err)
	}
	var material domain.Material
	result := r.db.WithContext(ctx).Where("id = ?", parsedID).First(&material)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find material: %w", result.Error)
	}
	return &material, nil
}

// FindByTopic implements ports.MaterialRepository.
func (r *MaterialRepository) FindByTopic(ctx context.Context, topicID string) ([]domain.Material, error) {
	parsedTopicID, err := uuid.Parse(topicID)
	if err != nil {
		return nil, fmt.Errorf("invalid topic id: %w", err)
	}
	var materials []domain.Material
	result := r.db.WithContext(ctx).Where("topic_id = ?", parsedTopicID).Order("created_at DESC").Find(&materials)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find materials by topic: %w", result.Error)
	}
	return materials, nil
}

// UpdateStatus implements ports.MaterialRepository.
func (r *MaterialRepository) UpdateStatus(ctx context.Context, materialID string, status domain.MaterialStatus, extractedText string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if extractedText != "" {
		updates["extracted_text"] = extractedText
	}
	result := r.db.WithContext(ctx).Model(&domain.Material{}).Where("id = ?", materialID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update material status: %w", result.Error)
	}
	return nil
}

// CreateMaterial persists a new material record and its content (PDF/text).
func (r *MaterialRepository) CreateMaterial(ctx context.Context, material *domain.Material, fileHeader *multipart.FileHeader) error {
	if material.ID == uuid.Nil {
		material.ID = uuid.New()
	}

	material.CreatedAt = time.Now()

	// Open file from multipart header
	if fileHeader != nil {
		file, err := fileHeader.Open()
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		// Extract text from file
		textContent, err := r.textExtractor.ExtractText(ctx, file, fileHeader.Header.Get("Content-Type"))
		if err != nil {
			return fmt.Errorf("failed to extract text: %w", err)
		}
		material.ExtractedText = textContent

		// Upload file to storage
		storageURL, err := r.storageService.UploadFile(ctx, material.ID, file, fileHeader.Filename)
		if err != nil {
			return fmt.Errorf("failed to upload file: %w", err)
		}
		material.StorageURL = storageURL
	}

	// Persist to database
	result := r.db.WithContext(ctx).Create(material)
	if result.Error != nil {
		return fmt.Errorf("failed to create material: %w", result.Error)
	}

	return nil
}

// GetMaterialByID retrieves a material with all its chunks.
func (r *MaterialRepository) GetMaterialByID(materialID uuid.UUID) (*domain.Material, error) {
	var material domain.Material
	result := r.db.
		Preload("Chunks").
		Where("id = ?", materialID).
		First(&material)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get material: %w", result.Error)
	}

	return &material, nil
}

// GetMaterialsByTopic retrieves all materials for a specific topic.
func (r *MaterialRepository) GetMaterialsByTopic(topicID uuid.UUID) ([]*domain.Material, error) {
	var materials []*domain.Material
	result := r.db.
		Where("topic_id = ?", topicID).
		Order("created_at DESC").
		Find(&materials)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get materials for topic: %w", result.Error)
	}

	return materials, nil
}

// GetMaterialsByCourse retrieves all materials in a course via topics.
func (r *MaterialRepository) GetMaterialsByCourse(courseID uuid.UUID) ([]*domain.Material, error) {
	var materials []*domain.Material
	result := r.db.
		Joins("JOIN topics ON topics.id = materials.topic_id").
		Where("topics.course_id = ?", courseID).
		Order("materials.created_at DESC").
		Find(&materials)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get materials for course: %w", result.Error)
	}

	return materials, nil
}

// UpdateMaterial updates material metadata (title, description, etc)
func (r *MaterialRepository) UpdateMaterial(material *domain.Material) error {
	result := r.db.Model(material).Updates(material)
	if result.Error != nil {
		return fmt.Errorf("failed to update material: %w", result.Error)
	}
	return nil
}

// DeleteMaterial soft-deletes a material (cascade to chunks via FK).
func (r *MaterialRepository) DeleteMaterial(materialID uuid.UUID) error {
	result := r.db.Delete(&domain.Material{}, "id = ?", materialID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete material: %w", result.Error)
	}
	return nil
}

// CountMaterialsByTopic returns the number of materials in a topic.
func (r *MaterialRepository) CountMaterialsByTopic(topicID uuid.UUID) (int64, error) {
	var count int64
	result := r.db.Model(&domain.Material{}).Where("topic_id = ?", topicID).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to count materials: %w", result.Error)
	}
	return count, nil
}
