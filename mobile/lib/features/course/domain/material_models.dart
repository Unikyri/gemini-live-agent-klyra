import 'package:freezed_annotation/freezed_annotation.dart';

part 'material_models.freezed.dart';
part 'material_models.g.dart';

// Mirrors backend domain.MaterialStatus
enum MaterialStatus {
  @JsonValue('pending')
  pending,
  @JsonValue('processing')
  processing,
  @JsonValue('validated')
  validated,
  @JsonValue('rejected')
  rejected,
}

// Mirrors backend domain.MaterialFormatType
enum MaterialFormatType {
  @JsonValue('pdf')
  pdf,
  @JsonValue('txt')
  txt,
  @JsonValue('md')
  md,
  @JsonValue('audio')
  audio,
}

extension MaterialStatusX on MaterialStatus {
  bool get isProcessing => this == MaterialStatus.pending || this == MaterialStatus.processing;
  bool get isReady => this == MaterialStatus.validated;
  bool get isFailed => this == MaterialStatus.rejected;
}

@freezed
abstract class Material with _$Material {
  const factory Material({
    required String id,
    @JsonKey(name: 'topic_id') required String topicId,
    @JsonKey(name: 'format_type') required MaterialFormatType formatType,
    @JsonKey(name: 'storage_url') required String storageUrl,
    @JsonKey(name: 'extracted_text') String? extractedText,
    required MaterialStatus status,
    @JsonKey(name: 'original_name') required String originalName,
    @JsonKey(name: 'size_bytes') required int sizeBytes,
    @JsonKey(name: 'created_at') required DateTime createdAt,
    @JsonKey(name: 'updated_at') required DateTime updatedAt,
  }) = _Material;

  factory Material.fromJson(Map<String, dynamic> json) =>
      _$MaterialFromJson(json);
}
