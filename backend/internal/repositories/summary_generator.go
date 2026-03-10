package repositories

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// MarkdownSummaryGenerator creates a compact markdown summary from topic context.
type MarkdownSummaryGenerator struct{}

// NewMarkdownSummaryGenerator creates a new summary generator implementation.
func NewMarkdownSummaryGenerator() *MarkdownSummaryGenerator {
	return &MarkdownSummaryGenerator{}
}

// Generate builds markdown while tagging malformed latex blocks for transparent UI fallback.
func (g *MarkdownSummaryGenerator) Generate(ctx context.Context, topicTitle string, contextText string) (string, error) {
	_ = ctx
	clean := strings.TrimSpace(contextText)
	if clean == "" {
		return "", fmt.Errorf("empty context text")
	}

	if len(clean) > 1800 {
		clean = clean[:1800]
	}

	tagged := tagInvalidLatex(clean)
	return fmt.Sprintf("## Resumen del tema: %s\n\n%s", topicTitle, tagged), nil
}

func tagInvalidLatex(text string) string {
	blockCount := strings.Count(text, "$$")
	inlineCount := strings.Count(text, "$")
	if blockCount%2 == 1 || inlineCount%2 == 1 {
		return text + "\n\n[latex_warning] Se detecto expresion LaTeX invalida en el resumen."
	}

	invalidBlock := regexp.MustCompile(`\$\$\s*\$\$`)
	if invalidBlock.MatchString(text) {
		return text + "\n\n[latex_warning] Se detecto bloque LaTeX vacio."
	}
	return text
}
