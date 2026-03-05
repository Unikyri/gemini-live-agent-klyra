import 'package:freezed_annotation/freezed_annotation.dart';

part 'course_models.freezed.dart';
part 'course_models.g.dart';

@freezed
class Course with _$Course {
  const factory Course({
    required String id,
    @JsonKey(name: 'user_id') required String userId,
    required String name,
    @JsonKey(name: 'education_level') required String educationLevel,
    @JsonKey(name: 'avatar_model_url') String? avatarModelUrl,
    @JsonKey(name: 'avatar_status') required String avatarStatus,
    @JsonKey(name: 'created_at') required DateTime createdAt,
    @JsonKey(name: 'updated_at') required DateTime updatedAt,
    @JsonKey(name: 'topics') @Default([]) List<Topic> topics,
  }) = _Course;

  factory Course.fromJson(Map<String, dynamic> json) => _$CourseFromJson(json);
}

@freezed
class CreateCourseRequest with _$CreateCourseRequest {
  const factory CreateCourseRequest({
    required String name,
    @JsonKey(name: 'education_level') required String educationLevel,
  }) = _CreateCourseRequest;

  factory CreateCourseRequest.fromJson(Map<String, dynamic> json) =>
      _$CreateCourseRequestFromJson(json);
}

@freezed
class Topic with _$Topic {
  const factory Topic({
    required String id,
    @JsonKey(name: 'course_id') required String courseId,
    required String title,
    @JsonKey(name: 'order_index') @Default(0) int orderIndex,
    @JsonKey(name: 'consolidated_context') String? consolidatedContext,
    @JsonKey(name: 'created_at') required DateTime createdAt,
    @JsonKey(name: 'updated_at') required DateTime updatedAt,
  }) = _Topic;

  factory Topic.fromJson(Map<String, dynamic> json) => _$TopicFromJson(json);
}

@freezed
class CreateTopicRequest with _$CreateTopicRequest {
  const factory CreateTopicRequest({
    required String title,
    @JsonKey(name: 'order_index') @Default(0) int orderIndex,
  }) = _CreateTopicRequest;

  factory CreateTopicRequest.fromJson(Map<String, dynamic> json) =>
      _$CreateTopicRequestFromJson(json);
}
