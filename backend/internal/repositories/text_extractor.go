package repositories

import (
	"context"
	"fmt"
	"strings"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/domain"
)

// PlainTextExtractor implements the TextExtractor port for text-based formats (TXT, MD).
// PDF extraction is NOT implemented in Sprint 2 — PDF files are stored but not parsed.
// PDF full extraction (via PDFium or Gemini Doc API) is planned for Sprint 3.
type PlainTextExtractor struct{}

// NewPlainTextExtractor creates a PlainTextExtractor.
func NewPlainTextExtractor() *PlainTextExtractor {
	return &PlainTextExtractor{}
}

// Extract returns the plain text content from raw bytes.
// For TXT and MD: returns the content as-is (UTF-8 string).
// For PDF: returns ErrUnsupportedFormat — caller sets status to "rejected" for PDF in this sprint.
// For audio: returns ErrUnsupportedFormat — speech-to-text planned for Sprint 4.
func (e *PlainTextExtractor) Extract(_ context.Context, data []byte, formatType domain.MaterialFormatType) (string, error) {
	switch formatType {
	case domain.MaterialFormatTXT, domain.MaterialFormatMD:
		// SECURITY: trim to 500KB of extracted text to avoid storing massive blobs.
		// Full content stays in GCS; only the first 500KB is indexed for RAG.
		const maxExtractedChars = 500_000
		text := string(data)
		text = strings.TrimSpace(text)
		if len(text) > maxExtractedChars {
			text = text[:maxExtractedChars]
		}
		return text, nil

	case domain.MaterialFormatPDF:
		// PDF parsing will be implemented in Sprint 3 using the Gemini Document AI API.
		// For now, the file is stored in GCS but text is not extracted.
		return "", fmt.Errorf("PDF text extraction not yet supported — file stored in Cloud Storage")

	case domain.MaterialFormatAudio:
		// Speech-to-text transcription planned for Sprint 4 via Gemini speech API.
		return "", fmt.Errorf("audio transcription not yet supported — file stored in Cloud Storage")

	default:
		return "", fmt.Errorf("unsupported format: %s", formatType)
	}
}
