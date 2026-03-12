package domain

// InterpretationBlockType enumerates structured block types extracted from a material.
type InterpretationBlockType string

const (
	InterpretationBlockText            InterpretationBlockType = "text"
	InterpretationBlockEquation        InterpretationBlockType = "equation"
	InterpretationBlockFigure          InterpretationBlockType = "figure"
	InterpretationBlockAudioTranscript InterpretationBlockType = "audio_transcript"
)

// InterpretationBlock represents a single extracted unit (text, equation, figure, transcript).
type InterpretationBlock struct {
	BlockIndex        int                   `json:"block_index"`
	BlockType         InterpretationBlockType `json:"block_type"`
	Content           string                `json:"content,omitempty"`
	LaTeX             string                `json:"latex,omitempty"`
	FigureDescription string                `json:"figure_description,omitempty"`
	Confidence        *float64              `json:"confidence,omitempty"`
}

// InterpretationResult is the top-level response returned to clients.
type InterpretationResult struct {
	Summary string               `json:"summary,omitempty"`
	Blocks  []InterpretationBlock `json:"blocks"`
}

