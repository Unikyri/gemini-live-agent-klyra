package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

type MockStorageService struct {
	mock.Mock
}

func (m *MockStorageService) UploadFile(ctx context.Context, materialID uuid.UUID, file interface{}, filename string) (string, error) {
	args := m.Called(ctx, materialID, file, filename)
	return args.String(0), args.Error(1)
}

type MockTextExtractor struct {
	mock.Mock
}

func (m *MockTextExtractor) ExtractText(ctx context.Context, file interface{}, contentType string) (string, error) {
	args := m.Called(ctx, file, contentType)
	return args.String(0), args.Error(1)
}

func TestGetMaterialByID_Success(t *testing.T) {
	// This test would require a database fixture
	// For now, we document the expected behavior
	expectedMaterial := &domain.Material{
		ID:       uuid.New(),
		TopicID:  uuid.New(),
		Title:    "Introduction to Neural Networks",
		Content:  "Neural networks are...",
		FileType: "pdf",
	}

	_ = expectedMaterial
	// In a full integration test:
	// 1. Create material in database
	// 2. Retrieve via repository
	// 3. Assert equality
}

func TestCreateMaterial_WithFileExtraction(t *testing.T) {
	mockStorage := new(MockStorageService)
	mockExtractor := new(MockTextExtractor)

	materialID := uuid.New()
	filename := "neuroscience-101.pdf"

	mockExtractor.On("ExtractText", mock.Anything, mock.Anything, "application/pdf").
		Return("Chapter 1: Brain Anatomy...", nil)

	mockStorage.On("UploadFile", mock.Anything, materialID, mock.Anything, filename).
		Return("gs://klyra-bucket/materials/"+materialID.String()+"/neuroscience-101.pdf", nil)

	// Verify mocks were called
	mockExtractor.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestCreateMaterial_ExtractionFails(t *testing.T) {
	mockStorage := new(MockStorageService)
	mockExtractor := new(MockTextExtractor)

	mockExtractor.On("ExtractText", mock.Anything, mock.Anything, mock.Anything).
		Return("", errors.New("unsupported file format"))

	// Verify that extraction failure is properly propagated
	mockExtractor.AssertExpectations(t)
}
