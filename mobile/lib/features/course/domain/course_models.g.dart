// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'course_models.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_Course _$CourseFromJson(Map<String, dynamic> json) => _Course(
  id: json['id'] as String,
  userId: json['user_id'] as String,
  name: json['name'] as String,
  educationLevel: json['education_level'] as String,
  avatarModelUrl: json['avatar_model_url'] as String?,
  avatarStatus: json['avatar_status'] as String,
  createdAt: DateTime.parse(json['created_at'] as String),
  updatedAt: DateTime.parse(json['updated_at'] as String),
  topics:
      (json['topics'] as List<dynamic>?)
          ?.map((e) => Topic.fromJson(e as Map<String, dynamic>))
          .toList() ??
      const [],
);

Map<String, dynamic> _$CourseToJson(_Course instance) => <String, dynamic>{
  'id': instance.id,
  'user_id': instance.userId,
  'name': instance.name,
  'education_level': instance.educationLevel,
  'avatar_model_url': instance.avatarModelUrl,
  'avatar_status': instance.avatarStatus,
  'created_at': instance.createdAt.toIso8601String(),
  'updated_at': instance.updatedAt.toIso8601String(),
  'topics': instance.topics,
};

_CreateCourseRequest _$CreateCourseRequestFromJson(Map<String, dynamic> json) =>
    _CreateCourseRequest(
      name: json['name'] as String,
      educationLevel: json['education_level'] as String,
    );

Map<String, dynamic> _$CreateCourseRequestToJson(
  _CreateCourseRequest instance,
) => <String, dynamic>{
  'name': instance.name,
  'education_level': instance.educationLevel,
};

_Topic _$TopicFromJson(Map<String, dynamic> json) => _Topic(
  id: json['id'] as String,
  courseId: json['course_id'] as String,
  title: json['title'] as String,
  orderIndex: (json['order_index'] as num?)?.toInt() ?? 0,
  consolidatedContext: json['consolidated_context'] as String?,
  createdAt: DateTime.parse(json['created_at'] as String),
  updatedAt: DateTime.parse(json['updated_at'] as String),
);

Map<String, dynamic> _$TopicToJson(_Topic instance) => <String, dynamic>{
  'id': instance.id,
  'course_id': instance.courseId,
  'title': instance.title,
  'order_index': instance.orderIndex,
  'consolidated_context': instance.consolidatedContext,
  'created_at': instance.createdAt.toIso8601String(),
  'updated_at': instance.updatedAt.toIso8601String(),
};

_CreateTopicRequest _$CreateTopicRequestFromJson(Map<String, dynamic> json) =>
    _CreateTopicRequest(
      title: json['title'] as String,
      orderIndex: (json['order_index'] as num?)?.toInt() ?? 0,
    );

Map<String, dynamic> _$CreateTopicRequestToJson(_CreateTopicRequest instance) =>
    <String, dynamic>{
      'title': instance.title,
      'order_index': instance.orderIndex,
    };
