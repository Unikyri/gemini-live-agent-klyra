import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:klyra/features/course/data/course_remote_datasource.dart';
import 'package:klyra/features/course/domain/course_models.dart';

part 'course_repository.g.dart';

@riverpod
CourseRepository courseRepository(Ref ref) {
  final remote = ref.watch(courseRemoteDataSourceProvider);
  return CourseRepository(remote);
}

class CourseRepository {
  final CourseRemoteDataSource _remote;

  CourseRepository(this._remote);

  Future<List<Course>> getCourses() => _remote.getCourses();

  Future<Course> getCourse(String courseId) => _remote.getCourse(courseId);

  Future<Map<String, dynamic>> fetchCourseContext(String courseId, {String? query}) =>
      _remote.fetchCourseContext(courseId, query: query);

  Future<Course> updateCourse(String courseId, {String? name, String? educationLevel}) =>
      _remote.updateCourse(courseId, name: name, educationLevel: educationLevel);

  Future<void> deleteCourse(String courseId) => _remote.deleteCourse(courseId);

  Future<Topic> updateTopic(String courseId, String topicId, {String? title}) =>
      _remote.updateTopic(courseId, topicId, title: title);

  Future<void> deleteTopic(String courseId, String topicId) =>
      _remote.deleteTopic(courseId, topicId);

  Future<Course> createCourse(CreateCourseRequest request) => _remote.createCourse(request);

  Future<Topic> addTopic(String courseId, String title) => _remote.addTopic(courseId, title);
}
