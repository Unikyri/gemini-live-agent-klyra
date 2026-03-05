// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'material_models.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_Material _$MaterialFromJson(Map<String, dynamic> json) => _Material(
  id: json['id'] as String,
  topicId: json['topic_id'] as String,
  formatType: $enumDecode(_$MaterialFormatTypeEnumMap, json['format_type']),
  storageUrl: json['storage_url'] as String,
  extractedText: json['extracted_text'] as String?,
  status: $enumDecode(_$MaterialStatusEnumMap, json['status']),
  originalName: json['original_name'] as String,
  sizeBytes: (json['size_bytes'] as num).toInt(),
  createdAt: DateTime.parse(json['created_at'] as String),
  updatedAt: DateTime.parse(json['updated_at'] as String),
);

Map<String, dynamic> _$MaterialToJson(_Material instance) => <String, dynamic>{
  'id': instance.id,
  'topic_id': instance.topicId,
  'format_type': _$MaterialFormatTypeEnumMap[instance.formatType]!,
  'storage_url': instance.storageUrl,
  'extracted_text': instance.extractedText,
  'status': _$MaterialStatusEnumMap[instance.status]!,
  'original_name': instance.originalName,
  'size_bytes': instance.sizeBytes,
  'created_at': instance.createdAt.toIso8601String(),
  'updated_at': instance.updatedAt.toIso8601String(),
};

const _$MaterialFormatTypeEnumMap = {
  MaterialFormatType.pdf: 'pdf',
  MaterialFormatType.txt: 'txt',
  MaterialFormatType.md: 'md',
  MaterialFormatType.audio: 'audio',
};

const _$MaterialStatusEnumMap = {
  MaterialStatus.pending: 'pending',
  MaterialStatus.processing: 'processing',
  MaterialStatus.validated: 'validated',
  MaterialStatus.rejected: 'rejected',
};
