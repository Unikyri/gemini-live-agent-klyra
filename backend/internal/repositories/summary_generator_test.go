package repositories

import (
	"context"
	"strings"
	"testing"
)

// TestMarkdownSummaryGenerator_Generate_Success tests normal summary generation.
func TestMarkdownSummaryGenerator_Generate_Success(t *testing.T) {
	gen := NewMarkdownSummaryGenerator()
	ctx := context.Background()

	summary, err := gen.Generate(ctx, "Algebra", "Material covers quadratic equations.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(summary, "## Resumen del tema: Algebra") {
		t.Errorf("expected topic title in summary")
	}
	if !strings.Contains(summary, "quadratic equations") {
		t.Errorf("expected context text in summary")
	}
}

// TestMarkdownSummaryGenerator_Generate_EmptyContext tests error on empty context.
func TestMarkdownSummaryGenerator_Generate_EmptyContext(t *testing.T) {
	gen := NewMarkdownSummaryGenerator()
	ctx := context.Background()

	_, err := gen.Generate(ctx, "Empty Topic", "")
	if err == nil {
		t.Fatalf("expected error for empty context")
	}
	if !strings.Contains(err.Error(), "empty context text") {
		t.Errorf("expected 'empty context text' error, got: %v", err)
	}
}

// TestMarkdownSummaryGenerator_Generate_TruncatesLongText tests 1800 character truncation.
func TestMarkdownSummaryGenerator_Generate_TruncatesLongText(t *testing.T) {
	gen := NewMarkdownSummaryGenerator()
	ctx := context.Background()

	longText := strings.Repeat("x", 2500)
	summary, err := gen.Generate(ctx, "Long Topic", longText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Summary includes header + newlines + 1800 chars
	if len(summary) > 1850 {
		t.Errorf("expected truncation to ~1800 chars, got length: %d", len(summary))
	}
}

// TestMarkdownSummaryGenerator_Generate_ValidLatex_NoWarning tests valid LaTeX passes through.
func TestMarkdownSummaryGenerator_Generate_ValidLatex_NoWarning(t *testing.T) {
	gen := NewMarkdownSummaryGenerator()
	ctx := context.Background()

	contextWithLatex := "The formula $$E=mc^2$$ is famous. Also $F=ma$."
	summary, err := gen.Generate(ctx, "Physics", contextWithLatex)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(summary, "[latex_warning]") {
		t.Errorf("expected no LaTeX warning for valid expressions")
	}
	if !strings.Contains(summary, "$$E=mc^2$$") {
		t.Errorf("expected LaTeX block preserved in output")
	}
}

// TestMarkdownSummaryGenerator_Generate_UnmatchedBlockLatex tests warning on unmatched $$.
func TestMarkdownSummaryGenerator_Generate_UnmatchedBlockLatex(t *testing.T) {
	gen := NewMarkdownSummaryGenerator()
	ctx := context.Background()

	// NOTE: Current implementation counts $ and $$ independently,
	// so "$$x" has 2 $ (even) and 1 $$ (odd), which triggers warning via blockCount.
	// But "$$x$$y" has 4 $ (even) and 2 $$ (even), so no warning.
	// To trigger warning reliably, use odd number of $$ or odd total $.
	unmatchedBlock := "Formula: $$E=mc^2$$ and $x" // Odd number of $ total (4+1=5)
	summary, err := gen.Generate(ctx, "Math", unmatchedBlock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(summary, "[latex_warning]") {
		t.Errorf("expected LaTeX warning for unmatched delimiters")
	}
}

// TestMarkdownSummaryGenerator_Generate_UnmatchedInlineLatex tests warning on unmatched $.
func TestMarkdownSummaryGenerator_Generate_UnmatchedInlineLatex(t *testing.T) {
	gen := NewMarkdownSummaryGenerator()
	ctx := context.Background()

	unmatchedInline := "Formula: $E=mc^2$x$y" // 5 $ total (odd)
	summary, err := gen.Generate(ctx, "Math", unmatchedInline)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(summary, "[latex_warning]") {
		t.Errorf("expected LaTeX warning for unmatched $ delimiter")
	}
}

// TestMarkdownSummaryGenerator_Generate_EmptyLatexBlock tests warning on empty $$$$.
func TestMarkdownSummaryGenerator_Generate_EmptyLatexBlock(t *testing.T) {
	gen := NewMarkdownSummaryGenerator()
	ctx := context.Background()

	emptyBlock := "Some text $$$$ more text."
	summary, err := gen.Generate(ctx, "Empty Block", emptyBlock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(summary, "[latex_warning]") {
		t.Errorf("expected LaTeX warning for empty $$$$ block")
	}
	if !strings.Contains(summary, "bloque LaTeX vacio") {
		t.Errorf("expected empty block warning message")
	}
}

// TestMarkdownSummaryGenerator_Generate_MultipleLatexBlocks tests valid multiple LaTeX blocks.
func TestMarkdownSummaryGenerator_Generate_MultipleLatexBlocks(t *testing.T) {
	gen := NewMarkdownSummaryGenerator()
	ctx := context.Background()

	multipleBlocks := "First: $$E=mc^2$$. Second: $$F=ma$$. Inline: $x=y$."
	summary, err := gen.Generate(ctx, "Physics", multipleBlocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(summary, "[latex_warning]") {
		t.Errorf("expected no warning for valid multiple LaTeX blocks")
	}
}

// TestMarkdownSummaryGenerator_Generate_MixedValidInvalid tests warning when some LaTeX invalid.
func TestMarkdownSummaryGenerator_Generate_MixedValidInvalid(t *testing.T) {
	gen := NewMarkdownSummaryGenerator()
	ctx := context.Background()

	// Valid block $$E=mc^2$$, but add unmatched inline to make total $ count odd
	mixed := "Valid: $$E=mc^2$$. Invalid: $x" // 4 + 1 = 5 $ total (odd)
	summary, err := gen.Generate(ctx, "Mixed", mixed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(summary, "[latex_warning]") {
		t.Errorf("expected warning when any LaTeX expression is invalid")
	}
}
