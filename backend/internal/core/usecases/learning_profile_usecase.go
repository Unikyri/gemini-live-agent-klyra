package usecases

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Unikyri/gemini-live-agent-klyra/backend/internal/core/ports"
)

// LearningProfileUseCase maintains an "invisible" learning profile (memory bank).
// NOTE: In this MVP phase, the profile update uses lightweight heuristics unless
// an LLM-backed summarizer is wired later.
type LearningProfileUseCase struct {
	userRepo ports.UserRepository
}

func NewLearningProfileUseCase(userRepo ports.UserRepository) *LearningProfileUseCase {
	return &LearningProfileUseCase{userRepo: userRepo}
}

func (uc *LearningProfileUseCase) GetProfile(ctx context.Context, userID string) (map[string]interface{}, error) {
	u, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, nil
	}
	if u.LearningProfile == nil {
		return map[string]interface{}{}, nil
	}
	return u.LearningProfile, nil
}

// UpdateProfile merges recentMessages into the existing profile and persists it.
// recentMessages are expected to be the last N user/tutor utterances.
func (uc *LearningProfileUseCase) UpdateProfile(ctx context.Context, userID string, recentMessages []string) (map[string]interface{}, error) {
	u, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, nil
	}

	current := u.LearningProfile
	if current == nil {
		current = map[string]interface{}{}
	}

	updated := cloneMap(current)

	// --- Counters ---
	updated["total_sessions"] = asInt(updated["total_sessions"]) + 1
	updated["total_session_minutes"] = asInt(updated["total_session_minutes"]) + estimateMinutes(recentMessages)
	updated["last_updated_at"] = time.Now().UTC().Format(time.RFC3339)

	// --- Simple signals ---
	// difficult_topics: append unique keywords found in recent messages
	keywords := extractKeywords(recentMessages)
	if len(keywords) > 0 {
		existing := asStringSlice(updated["difficult_topics"])
		updated["difficult_topics"] = mergeUnique(existing, keywords, 12)
	}

	// style_scores: nudge based on language cues
	style := asMap(updated["style_scores"])
	if style == nil {
		style = map[string]interface{}{}
	}
	incrementStyleScores(style, recentMessages)
	updated["style_scores"] = style

	if err := uc.userRepo.UpdateLearningProfile(ctx, userID, updated); err != nil {
		return nil, err
	}
	return updated, nil
}

func cloneMap(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func asInt(v interface{}) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	default:
		return 0
	}
}

func asStringSlice(v interface{}) []string {
	raw, ok := v.([]interface{})
	if ok {
		out := make([]string, 0, len(raw))
		for _, x := range raw {
			if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	}
	if ss, ok := v.([]string); ok {
		return ss
	}
	return []string{}
}

func mergeUnique(existing, add []string, max int) []string {
	seen := make(map[string]struct{}, len(existing))
	out := make([]string, 0, max)
	for _, s := range existing {
		k := strings.ToLower(strings.TrimSpace(s))
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, s)
		if len(out) >= max {
			return out
		}
	}
	for _, s := range add {
		k := strings.ToLower(strings.TrimSpace(s))
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, s)
		if len(out) >= max {
			return out
		}
	}
	return out
}

func estimateMinutes(msgs []string) int {
	totalChars := 0
	for _, m := range msgs {
		totalChars += len(m)
	}
	// Heuristic: ~900 chars/min for speech+text combined.
	minutes := int(math.Ceil(float64(totalChars) / 900.0))
	if minutes < 1 {
		minutes = 1
	}
	if minutes > 60 {
		minutes = 60
	}
	return minutes
}

func extractKeywords(msgs []string) []string {
	text := strings.ToLower(strings.Join(msgs, " "))
	// Very small heuristic set: pick out common STEM markers.
	candidates := []string{
		"derivada", "integral", "límite", "limite", "matriz", "vector",
		"probabilidad", "estadística", "estadistica", "física", "fisica",
		"química", "quimica", "algoritmo", "ecuación", "ecuacion",
	}
	var out []string
	for _, c := range candidates {
		if strings.Contains(text, c) {
			out = append(out, c)
		}
	}
	return out
}

func asMap(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return nil
}

func incrementStyleScores(style map[string]interface{}, msgs []string) {
	joined := strings.ToLower(strings.Join(msgs, " "))
	bump := func(key string, delta float64) {
		cur := 0.0
		switch t := style[key].(type) {
		case float64:
			cur = t
		case int:
			cur = float64(t)
		}
		style[key] = math.Min(1.0, cur+delta)
	}
	if strings.Contains(joined, "ejemplo") || strings.Contains(joined, "ejemplos") {
		bump("examples", 0.1)
	}
	if strings.Contains(joined, "paso a paso") {
		bump("step_by_step", 0.12)
	}
	if strings.Contains(joined, "resumen") {
		bump("summary", 0.08)
	}
}

func (uc *LearningProfileUseCase) DebugString(ctx context.Context, userID string) (string, error) {
	p, err := uc.GetProfile(ctx, userID)
	if err != nil {
		return "", err
	}
	if p == nil {
		return "", nil
	}
	return fmt.Sprintf("%v", p), nil
}

