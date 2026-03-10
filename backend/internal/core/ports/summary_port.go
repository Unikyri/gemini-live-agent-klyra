package ports

import "context"

// SummaryGenerator builds a markdown summary from consolidated topic text.
type SummaryGenerator interface {
	Generate(ctx context.Context, topicTitle string, contextText string) (string, error)
}
