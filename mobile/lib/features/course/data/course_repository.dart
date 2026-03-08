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

  Future<Course> createCourse(CreateCourseRequest request) => _remote.createCourse(request);

  Future<Topic> addTopic(String courseId, String title) => _remote.addTopic(courseId, title);
}
