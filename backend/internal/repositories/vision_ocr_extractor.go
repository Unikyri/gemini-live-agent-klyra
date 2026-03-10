package repositories

import (
	"context"
	"fmt"
	"strings"
)

// VisionOCRExtractor is a lightweight OCR adapter placeholder.
// In production this should call Google Cloud Vision OCR.
type VisionOCRExtractor struct {
	minConfidence float64
}

// NewVisionOCRExtractor creates an OCR extractor with quality threshold.
func NewVisionOCRExtractor(minConfidence float64) *VisionOCRExtractor {
	if minConfidence <= 0 || minConfidence > 1 {
		minConfidence = 0.70
	}
	return &VisionOCRExtractor{minConfidence: minConfidence}
}

// ExtractText returns OCR text and confidence score.
func (v *VisionOCRExtractor) ExtractText(ctx context.Context, imageData []byte) (string, float64, error) {
	_ = ctx
	text := strings.TrimSpace(string(imageData))
	if text == "" {
		return "", 0, fmt.Errorf("ocr produced empty text")
	}
	return text, 1.0, nil
}

// MinConfidence exposes the configured threshold used by the extraction pipeline.
func (v *VisionOCRExtractor) MinConfidence() float64 {
	return v.minConfidence
}
