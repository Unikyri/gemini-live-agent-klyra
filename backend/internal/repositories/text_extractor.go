package repositories

import (
	"context"
	"fmt"
	"io"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// TextExtractorImpl extracts text from various file formats.
type TextExtractorImpl struct{}

// PlainTextExtractor is the ports-compatible extractor used by use cases.
type PlainTextExtractor struct{}

// NewTextExtractor creates a new text extractor instance.
func NewTextExtractor() *TextExtractorImpl {
	return &TextExtractorImpl{}
}

// NewPlainTextExtractor creates the extractor implementation expected by use cases.
func NewPlainTextExtractor() *PlainTextExtractor {
	return &PlainTextExtractor{}
}

// Extract implements ports.TextExtractor.
func (t *PlainTextExtractor) Extract(ctx context.Context, data []byte, formatType domain.MaterialFormatType) (string, error) {
	_ = ctx
	switch formatType {
	case domain.MaterialFormatTXT, domain.MaterialFormatMD:
		return string(data), nil
	case domain.MaterialFormatPDF, domain.MaterialFormatAudio:
		// Placeholder for future parser/transcriber integration.
		return "", nil
	default:
		return "", fmt.Errorf("unsupported material format: %s", formatType)
	}
}

// ExtractText extracts text from files based on content type.
// Supported: text/plain, application/pdf, application/vnd.openxmlformats-officedocument.wordprocessingml.document
func (t *TextExtractorImpl) ExtractText(ctx context.Context, file interface{}, contentType string) (string, error) {
	switch contentType {
	case "text/plain":
		return t.extractPlainText(file)
	case "application/pdf":
		return t.extractPDFText(file)
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return t.extractDocxText(file)
	default:
		return "", fmt.Errorf("unsupported content type: %s", contentType)
	}
}

func (t *TextExtractorImpl) extractPlainText(file interface{}) (string, error) {
	readCloser, ok := file.(io.ReadCloser)
	if !ok {
		return "", fmt.Errorf("invalid file type")
	}
	defer readCloser.Close()

	bytes, err := io.ReadAll(readCloser)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(bytes), nil
}

func (t *TextExtractorImpl) extractPDFText(file interface{}) (string, error) {
	// TODO: Integrate pdfcpu or similar PDF parsing library
	// For MVP, return placeholder
	return "[PDF text extraction not yet implemented]", nil
}

func (t *TextExtractorImpl) extractDocxText(file interface{}) (string, error) {
	// TODO: Integrate unidoc or similar DOCX parsing library
	// For MVP, return placeholder
	return "[DOCX text extraction not yet implemented]", nil
}
