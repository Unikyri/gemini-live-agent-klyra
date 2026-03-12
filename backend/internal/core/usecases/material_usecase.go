package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/ports"
)

// MaterialUseCase holds the business rules for the Material Upload module.
// It depends only on ports (interfaces), never on concrete implementations.
type MaterialUseCase struct {
	materialRepo ports.MaterialRepository
	topicRepo    ports.TopicRepository
	courseRepo   ports.CourseRepository
	storage      ports.StorageService
	extractor    ports.TextExtractor
	correctionRepo ports.CorrectionRepository
	ragUseCase   *RAGUseCase
}

// ErrMaterialForbidden is returned when a user tries to access another user's course materials.
var ErrMaterialForbidden = errors.New("forbidden material access")
var ErrCorrectionNotFound = errors.New("correction not found")

// NewMaterialUseCase creates a new MaterialUseCase with injected dependencies.
func NewMaterialUseCase(
	materialRepo ports.MaterialRepository,
	topicRepo ports.TopicRepository,
	courseRepo ports.CourseRepository,
	storage ports.StorageService,
	extractor ports.TextExtractor,
	correctionRepo ports.CorrectionRepository,
	ragUseCase *RAGUseCase,
) *MaterialUseCase {
	return &MaterialUseCase{
		materialRepo: materialRepo,
		topicRepo:    topicRepo,
		courseRepo:   courseRepo,
		storage:      storage,
		extractor:    extractor,
		correctionRepo: correctionRepo,
		ragUseCase:   ragUseCase,
	}
}

func (uc *MaterialUseCase) validateCourseTopicOwnership(ctx context.Context, courseID, topicID, userID string) error {
	course, err := uc.courseRepo.FindByID(ctx, courseID)
	if err != nil {
		return err
	}
	if course == nil {
		return nil
	}
	if course.UserID.String() != userID {
		return ErrMaterialForbidden
	}
	topics, err := uc.topicRepo.FindByCourse(ctx, courseID)
	if err != nil {
		return err
	}
	for _, t := range topics {
		if t.ID.String() == topicID {
			return nil
		}
	}
	// 404 anti-enumeration
	return nil
}

func (uc *MaterialUseCase) GetMaterialInterpretation(ctx context.Context, courseID, topicID, materialID, userID string) (*domain.InterpretationResult, error) {
	// ownership checks
	course, err := uc.courseRepo.FindByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if course == nil {
		return nil, nil
	}
	if course.UserID.String() != userID {
		return nil, ErrMaterialForbidden
	}
	topics, err := uc.topicRepo.FindByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	topicOK := false
	for _, t := range topics {
		if t.ID.String() == topicID {
			topicOK = true
			break
		}
	}
	if !topicOK {
		return nil, nil
	}

	material, err := uc.materialRepo.FindByID(ctx, materialID)
	if err != nil {
		return nil, err
	}
	if material == nil {
		return nil, nil
	}
	if material.TopicID.String() != topicID {
		return nil, nil
	}
	if len(material.InterpretationJSON) == 0 {
		return nil, nil
	}
	var result domain.InterpretationResult
	if err := json.Unmarshal(material.InterpretationJSON, &result); err != nil {
		return nil, fmt.Errorf("invalid interpretation_json: %w", err)
	}
	return &result, nil
}

type CreateCorrectionInput struct {
	UserID        string
	CourseID      string
	TopicID       string
	MaterialID    string
	BlockIndex    int
	OriginalText  string
	CorrectedText string
}

func (uc *MaterialUseCase) CreateCorrection(ctx context.Context, in CreateCorrectionInput) (*domain.MaterialCorrection, error) {
	course, err := uc.courseRepo.FindByID(ctx, in.CourseID)
	if err != nil {
		return nil, err
	}
	if course == nil {
		return nil, nil
	}
	if course.UserID.String() != in.UserID {
		return nil, ErrMaterialForbidden
	}

	material, err := uc.materialRepo.FindByID(ctx, in.MaterialID)
	if err != nil {
		return nil, err
	}
	if material == nil || material.TopicID.String() != in.TopicID {
		return nil, nil
	}
	// Require interpretation exists before corrections.
	if len(material.InterpretationJSON) == 0 {
		return nil, nil
	}

	corr := &domain.MaterialCorrection{
		MaterialID:    material.ID,
		BlockIndex:    in.BlockIndex,
		OriginalText:  in.OriginalText,
		CorrectedText: in.CorrectedText,
	}
	if uc.correctionRepo == nil {
		return nil, fmt.Errorf("correction repository not configured")
	}
	if err := uc.correctionRepo.Create(ctx, corr); err != nil {
		return nil, err
	}
	return corr, nil
}

func (uc *MaterialUseCase) ListCorrections(ctx context.Context, courseID, topicID, materialID, userID string) ([]domain.MaterialCorrection, error) {
	course, err := uc.courseRepo.FindByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if course == nil {
		return nil, nil
	}
	if course.UserID.String() != userID {
		return nil, ErrMaterialForbidden
	}
	material, err := uc.materialRepo.FindByID(ctx, materialID)
	if err != nil {
		return nil, err
	}
	if material == nil || material.TopicID.String() != topicID {
		return nil, nil
	}
	if uc.correctionRepo == nil {
		return []domain.MaterialCorrection{}, nil
	}
	return uc.correctionRepo.FindByMaterial(ctx, materialID)
}

func (uc *MaterialUseCase) DeleteCorrection(ctx context.Context, courseID, topicID, materialID, correctionID, userID string) error {
	course, err := uc.courseRepo.FindByID(ctx, courseID)
	if err != nil {
		return err
	}
	if course == nil {
		return ErrCorrectionNotFound
	}
	if course.UserID.String() != userID {
		return ErrMaterialForbidden
	}
	material, err := uc.materialRepo.FindByID(ctx, materialID)
	if err != nil {
		return err
	}
	if material == nil || material.TopicID.String() != topicID {
		return ErrCorrectionNotFound
	}
	if uc.correctionRepo == nil {
		return ErrCorrectionNotFound
	}
	if err := uc.correctionRepo.Delete(ctx, correctionID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCorrectionNotFound
		}
		return err
	}
	return nil
}

// UploadMaterialInput holds all data required to upload a material.
type UploadMaterialInput struct {
	UserID     string                    // from JWT — used for ownership validation
	CourseID   string                    // path param :course_id
	TopicID    string                    // path param :topic_id
	FileName   string                    // original filename from the upload
	FileData   []byte                    // raw file bytes
	FormatType domain.MaterialFormatType // pdf | txt | md
	SizeBytes  int64                     // pre-validated size
}

// UploadMaterial validates ownership, uploads the file to Cloud Storage, persist the record,
// and asynchronously extracts text from supported formats.
func (uc *MaterialUseCase) UploadMaterial(ctx context.Context, input UploadMaterialInput) (*domain.Material, error) {
	// Step 1: Validate that the course belongs to the requesting user.
	course, err := uc.courseRepo.FindByID(ctx, input.CourseID)
	if err != nil {
		return nil, err
	}
	if course == nil {
		return nil, nil
	}
	if course.UserID.String() != input.UserID {
		return nil, ErrMaterialForbidden
	}

	// Step 2: Validate that the topic belongs to the given course.
	topics, err := uc.topicRepo.FindByCourse(ctx, input.CourseID)
	if err != nil {
		return nil, err
	}
	topicExists := false
	for _, t := range topics {
		if t.ID.String() == input.TopicID {
			topicExists = true
			break
		}
	}
	if !topicExists {
		return nil, nil // 404 anti-enumeration
	}

	// Step 3: Upload the raw file to Cloud Storage.
	topicUUID, err := uuid.Parse(input.TopicID)
	if err != nil {
		return nil, fmt.Errorf("invalid topic ID: %w", err)
	}
	objectName := fmt.Sprintf("materials/%s/%s/%s", input.CourseID, input.TopicID, uuid.New().String())
	storageURL, err := uc.storage.UploadFile(ctx, "", objectName, input.FileData, mimeForFormat(input.FormatType))
	if err != nil {
		return nil, fmt.Errorf("failed to upload material to storage: %w", err)
	}

	// Step 4: Persist the material record with status "pending".
	material := &domain.Material{
		TopicID:      topicUUID,
		FormatType:   input.FormatType,
		StorageURL:   storageURL,
		Status:       domain.MaterialStatusPending,
		OriginalName: input.FileName,
		SizeBytes:    input.SizeBytes,
	}
	if err := uc.materialRepo.Create(ctx, material); err != nil {
		return nil, err
	}

	// Step 5: Kick off async text extraction for supported types.
	// This updates the status to "validated" or "rejected" when done.
	// NOTE: For Sprint 3+, replace with Cloud Tasks/Pub-Sub for reliability.
	go uc.extractTextAsync(material.ID.String(), input.FileData, input.FormatType)

	return material, nil
}

// GetMaterialsByTopic returns all materials for a topic, validating course ownership.
func (uc *MaterialUseCase) GetMaterialsByTopic(ctx context.Context, courseID, topicID, userID string) ([]domain.Material, error) {
	course, err := uc.courseRepo.FindByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if course == nil {
		return nil, nil
	}
	if course.UserID.String() != userID {
		return nil, ErrMaterialForbidden
	}
	return uc.materialRepo.FindByTopic(ctx, topicID)
}

// extractTextAsync runs in the background, extracts text and updates the material record.
func (uc *MaterialUseCase) extractTextAsync(materialID string, data []byte, format domain.MaterialFormatType) {
	ctx := context.Background()
	log.Printf("[Material] Starting text extraction for material %s (format: %s)", materialID, format)

	if err := uc.materialRepo.UpdateStatus(ctx, materialID, domain.MaterialStatusProcessing, ""); err != nil {
		log.Printf("[Material] Failed to set status=processing for %s: %v", materialID, err)
	}

	text, err := uc.extractor.Extract(ctx, data, format)
	if err != nil {
		log.Printf("[Material] Extraction failed for material %s: %v", materialID, err)
		_ = uc.materialRepo.UpdateStatus(ctx, materialID, domain.MaterialStatusRejected, "[rejected] extraction failed")
		return
	}

	if text == "" {
		log.Printf("[Material] Extraction produced empty content for material %s", materialID)
		_ = uc.materialRepo.UpdateStatus(ctx, materialID, domain.MaterialStatusRejected, "[rejected] empty extracted content")
		return
	}

	_ = uc.materialRepo.UpdateStatus(ctx, materialID, domain.MaterialStatusValidated, text)
	if uc.ragUseCase != nil {
		if err := uc.ragUseCase.ProcessMaterialChunks(ctx, materialID); err != nil {
			log.Printf("[Material] RAG chunking failed for material %s: %v", materialID, err)
		}
	}
	log.Printf("[Material] Extraction complete for material %s — %d chars extracted", materialID, len(text))
}

// mimeForFormat returns the correct MIME type for a given MaterialFormatType.
func mimeForFormat(f domain.MaterialFormatType) string {
	switch f {
	case domain.MaterialFormatPDF:
		return "application/pdf"
	case domain.MaterialFormatMD:
		return "text/markdown"
	case domain.MaterialFormatPNG:
		return "image/png"
	case domain.MaterialFormatJPG, domain.MaterialFormatJPEG:
		return "image/jpeg"
	case domain.MaterialFormatWEBP:
		return "image/webp"
	case domain.MaterialFormatAudio:
		return "audio/mpeg"
	default:
		return "text/plain"
	}
}
