import 'package:dio/dio.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:klyra/core/network/dio_client.dart';
import 'package:klyra/features/course/domain/course_models.dart';
import 'package:klyra/features/course/domain/topic_readiness.dart';

part 'course_remote_datasource.g.dart';

@riverpod
CourseRemoteDataSource courseRemoteDataSource(Ref ref) {
  final dio = ref.watch(dioClientProvider);
  return CourseRemoteDataSource(dio);
}

class CourseRemoteDataSource {
  final Dio _dio;

  CourseRemoteDataSource(this._dio);

  Future<List<Course>> getCourses() async {
    final response = await _dio.get('/courses');
    if (response.statusCode == 200) {
      final List<dynamic> data = response.data['courses'] ?? [];
      return data.map((json) => Course.fromJson(json)).toList();
    } else {
      throw Exception('Failed to load courses');
    }
  }

  Future<Course> createCourse(CreateCourseRequest request) async {
    // Convert CreateCourseRequest to FormData (backend expects multipart/form-data)
    final formData = FormData.fromMap({
      'name': request.name,
      'education_level': request.educationLevel,
      // reference_image is optional; omitted for now
    });

    final response = await _dio.post('/courses', data: formData);
    if (response.statusCode == 201) {
      return Course.fromJson(response.data);
    } else {
      throw Exception('Failed to create course: ${response.statusCode}');
    }
  }

  Future<Topic> addTopic(String courseId, String title) async {
    final response = await _dio.post(
      '/courses/$courseId/topics',
      data: {'title': title},
    );
    if (response.statusCode == 201) {
      return Topic.fromJson(response.data);
    } else {
      throw Exception('Failed to add topic: ${response.statusCode}');
    }
  }

  Future<TopicReadiness> checkTopicReadiness(
    String courseId,
    String topicId,
  ) async {
    final response = await _dio.get(
      '/courses/$courseId/topics/$topicId/readiness',
    );
    if (response.statusCode == 200) {
      return TopicReadiness.fromJson(Map<String, dynamic>.from(response.data));
    }
    throw Exception('Failed to check topic readiness: ${response.statusCode}');
  }

  Future<String> fetchTopicSummary(String courseId, String topicId) async {
    final response = await _dio.get(
      '/courses/$courseId/topics/$topicId/summary',
    );
    if (response.statusCode == 200) {
      return (response.data['summary'] as String?) ?? '';
    }
    throw Exception('Failed to fetch topic summary: ${response.statusCode}');
  }
}
