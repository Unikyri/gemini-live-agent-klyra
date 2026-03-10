package repositories

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// TextExtractorImpl extracts text from various file formats.
type TextExtractorImpl struct{}

// PlainTextExtractor is the ports-compatible extractor used by use cases.
type PlainTextExtractor struct {
	ocrExtractor     *VisionOCRExtractor
	audioExtractor   *SpeechTranscriber
	minOCRConfidence float64
}

// NewTextExtractor creates a new text extractor instance.
func NewTextExtractor() *TextExtractorImpl {
	return &TextExtractorImpl{}
}

// NewPlainTextExtractor creates the extractor implementation expected by use cases.
func NewPlainTextExtractor() *PlainTextExtractor {
	ocr := NewVisionOCRExtractor(0.70)
	return &PlainTextExtractor{
		ocrExtractor:     ocr,
		audioExtractor:   NewSpeechTranscriber(),
		minOCRConfidence: ocr.MinConfidence(),
	}
}

// Extract implements ports.TextExtractor.
func (t *PlainTextExtractor) Extract(ctx context.Context, data []byte, formatType domain.MaterialFormatType) (string, error) {
	_ = ctx
	switch formatType {
	case domain.MaterialFormatTXT, domain.MaterialFormatMD:
		return strings.TrimSpace(string(data)), nil
	case domain.MaterialFormatPDF:
		return "[PDF extraction pending parser integration]", nil
	case domain.MaterialFormatPNG, domain.MaterialFormatJPG, domain.MaterialFormatJPEG, domain.MaterialFormatWEBP:
		text, confidence, err := t.ExtractFromImage(ctx, data)
		if err != nil {
			return "", err
		}
		if confidence < t.minOCRConfidence {
			return "", fmt.Errorf("ocr confidence below threshold: %.2f", confidence)
		}
		return strings.TrimSpace(text), nil
	case domain.MaterialFormatAudio:
		transcript, err := t.ExtractFromAudio(ctx, data, "audio/mpeg")
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(transcript), nil
	default:
		return "", fmt.Errorf("unsupported material format: %s", formatType)
	}
}

// ExtractFromImage implements ports.TextExtractor.
func (t *PlainTextExtractor) ExtractFromImage(ctx context.Context, imageData []byte) (string, float64, error) {
	if t.ocrExtractor == nil {
		return "", 0, fmt.Errorf("ocr extractor not configured")
	}
	return t.ocrExtractor.ExtractText(ctx, imageData)
}

// ExtractFromAudio implements ports.TextExtractor.
func (t *PlainTextExtractor) ExtractFromAudio(ctx context.Context, audioData []byte, mimeType string) (string, error) {
	if t.audioExtractor == nil {
		return "", fmt.Errorf("audio extractor not configured")
	}
	return t.audioExtractor.Transcribe(ctx, audioData, mimeType)
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
