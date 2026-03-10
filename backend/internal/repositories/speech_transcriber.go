package repositories

import (
	"context"
	"fmt"
	"strings"
)

// SpeechTranscriber is a lightweight transcription adapter placeholder.
// In production this should call Google Cloud Speech-to-Text.
type SpeechTranscriber struct{}

// NewSpeechTranscriber creates an audio transcription adapter.
func NewSpeechTranscriber() *SpeechTranscriber {
	return &SpeechTranscriber{}
}

// Transcribe returns transcribed text from audio bytes.
func (s *SpeechTranscriber) Transcribe(ctx context.Context, audioData []byte, mimeType string) (string, error) {
	_ = ctx
	_ = mimeType
	text := strings.TrimSpace(string(audioData))
	if text == "" {
		return "", fmt.Errorf("transcription produced empty text")
	}
	return text, nil
}
