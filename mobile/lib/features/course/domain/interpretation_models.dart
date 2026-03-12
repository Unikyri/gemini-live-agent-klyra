enum InterpretationBlockType {
  text,
  equation,
  figure,
  audioTranscript,
}

InterpretationBlockType _blockTypeFromJson(String v) {
  switch (v) {
    case 'text':
      return InterpretationBlockType.text;
    case 'equation':
      return InterpretationBlockType.equation;
    case 'figure':
      return InterpretationBlockType.figure;
    case 'audio_transcript':
      return InterpretationBlockType.audioTranscript;
    default:
      return InterpretationBlockType.text;
  }
}

class InterpretationBlock {
  final int blockIndex;
  final InterpretationBlockType blockType;
  final String? content;
  final String? latex;
  final String? figureDescription;
  final double? confidence;

  const InterpretationBlock({
    required this.blockIndex,
    required this.blockType,
    this.content,
    this.latex,
    this.figureDescription,
    this.confidence,
  });

  factory InterpretationBlock.fromJson(Map<String, dynamic> json) {
    return InterpretationBlock(
      blockIndex: (json['block_index'] as num?)?.toInt() ?? 0,
      blockType: _blockTypeFromJson((json['block_type'] as String?) ?? 'text'),
      content: json['content'] as String?,
      latex: json['latex'] as String?,
      figureDescription: json['figure_description'] as String?,
      confidence: (json['confidence'] as num?)?.toDouble(),
    );
  }
}

class InterpretationResult {
  final String? summary;
  final List<InterpretationBlock> blocks;

  const InterpretationResult({this.summary, required this.blocks});

  factory InterpretationResult.fromJson(Map<String, dynamic> json) {
    final raw = (json['blocks'] as List?) ?? const [];
    return InterpretationResult(
      summary: json['summary'] as String?,
      blocks: raw
          .whereType<Map<String, dynamic>>()
          .map(InterpretationBlock.fromJson)
          .toList(),
    );
  }
}

class MaterialCorrection {
  final String id;
  final String materialId;
  final int blockIndex;
  final String originalText;
  final String correctedText;
  final DateTime createdAt;

  const MaterialCorrection({
    required this.id,
    required this.materialId,
    required this.blockIndex,
    required this.originalText,
    required this.correctedText,
    required this.createdAt,
  });

  factory MaterialCorrection.fromJson(Map<String, dynamic> json) {
    return MaterialCorrection(
      id: json['id'] as String,
      materialId: json['material_id'] as String,
      blockIndex: (json['block_index'] as num?)?.toInt() ?? 0,
      originalText: (json['original_text'] as String?) ?? '',
      correctedText: (json['corrected_text'] as String?) ?? '',
      createdAt: DateTime.tryParse((json['created_at'] as String?) ?? '') ??
          DateTime.fromMillisecondsSinceEpoch(0),
    );
  }
}

