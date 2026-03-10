package repositories

import (
	"context"
	"strings"
	"testing"
)

// TestVisionOCRExtractor_ExtractText_Success tests successful OCR extraction.
func TestVisionOCRExtractor_ExtractText_Success(t *testing.T) {
	extractor := NewVisionOCRExtractor(0.70)
	ctx := context.Background()

	imageData := []byte("Extracted text from image")
	text, confidence, err := extractor.ExtractText(ctx, imageData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Extracted text from image" {
		t.Errorf("expected extracted text, got: %s", text)
	}
	if confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got: %f", confidence)
	}
}

// TestVisionOCRExtractor_ExtractText_EmptyData tests error on empty image data.
func TestVisionOCRExtractor_ExtractText_EmptyData(t *testing.T) {
	extractor := NewVisionOCRExtractor(0.70)
	ctx := context.Background()

	_, _, err := extractor.ExtractText(ctx, []byte(""))
	if err == nil {
		t.Fatalf("expected error for empty image data")
	}
	if !strings.Contains(err.Error(), "ocr produced empty text") {
		t.Errorf("expected OCR empty text error, got: %v", err)
	}
}

// TestVisionOCRExtractor_ExtractText_WhitespaceOnly tests error on whitespace-only data.
func TestVisionOCRExtractor_ExtractText_WhitespaceOnly(t *testing.T) {
	extractor := NewVisionOCRExtractor(0.70)
	ctx := context.Background()

	_, _, err := extractor.ExtractText(ctx, []byte("   \n\t   "))
	if err == nil {
		t.Fatalf("expected error for whitespace-only data")
	}
	if !strings.Contains(err.Error(), "ocr produced empty text") {
		t.Errorf("expected OCR empty text error, got: %v", err)
	}
}

// TestVisionOCRExtractor_MinConfidence_DefaultValue tests default min confidence.
func TestVisionOCRExtractor_MinConfidence_DefaultValue(t *testing.T) {
	extractor := NewVisionOCRExtractor(0.0)
	if extractor.MinConfidence() != 0.70 {
		t.Errorf("expected default min confidence 0.70, got: %f", extractor.MinConfidence())
	}
}

// TestVisionOCRExtractor_MinConfidence_InvalidValue tests boundary clamping.
func TestVisionOCRExtractor_MinConfidence_InvalidValue(t *testing.T) {
	extractorNeg := NewVisionOCRExtractor(-0.5)
	if extractorNeg.MinConfidence() != 0.70 {
		t.Errorf("expected default on negative confidence")
	}

	extractorHigh := NewVisionOCRExtractor(1.5)
	if extractorHigh.MinConfidence() != 0.70 {
		t.Errorf("expected default on >1.0 confidence")
	}
}

// TestSpeechTranscriber_Transcribe_Success tests successful audio transcription.
func TestSpeechTranscriber_Transcribe_Success(t *testing.T) {
	transcriber := NewSpeechTranscriber()
	ctx := context.Background()

	audioData := []byte("Transcribed audio content")
	text, err := transcriber.Transcribe(ctx, audioData, "audio/webm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Transcribed audio content" {
		t.Errorf("expected transcribed text, got: %s", text)
	}
}

// TestSpeechTranscriber_Transcribe_EmptyData tests error on empty audio data.
func TestSpeechTranscriber_Transcribe_EmptyData(t *testing.T) {
	transcriber := NewSpeechTranscriber()
	ctx := context.Background()

	_, err := transcriber.Transcribe(ctx, []byte(""), "audio/webm")
	if err == nil {
		t.Fatalf("expected error for empty audio data")
	}
	if !strings.Contains(err.Error(), "transcription produced empty text") {
		t.Errorf("expected transcription empty text error, got: %v", err)
	}
}

// TestSpeechTranscriber_Transcribe_WhitespaceOnly tests error on whitespace-only transcription.
func TestSpeechTranscriber_Transcribe_WhitespaceOnly(t *testing.T) {
	transcriber := NewSpeechTranscriber()
	ctx := context.Background()

	_, err := transcriber.Transcribe(ctx, []byte("  \n  "), "audio/mp3")
	if err == nil {
		t.Fatalf("expected error for whitespace-only transcription")
	}
	if !strings.Contains(err.Error(), "transcription produced empty text") {
		t.Errorf("expected transcription empty text error, got: %v", err)
	}
}

// TestSpeechTranscriber_Transcribe_DifferentMimeTypes tests transcription with various mime types.
func TestSpeechTranscriber_Transcribe_DifferentMimeTypes(t *testing.T) {
	transcriber := NewSpeechTranscriber()
	ctx := context.Background()

	mimeTypes := []string{"audio/webm", "audio/mp3", "audio/wav", "audio/ogg"}
	audioData := []byte("Content for all formats")

	for _, mimeType := range mimeTypes {
		text, err := transcriber.Transcribe(ctx, audioData, mimeType)
		if err != nil {
			t.Errorf("unexpected error for mime type %s: %v", mimeType, err)
		}
		if text == "" {
			t.Errorf("expected transcribed text for mime type %s", mimeType)
		}
	}
}
