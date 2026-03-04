import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:klyra/features/course/data/course_repository.dart';
import 'package:klyra/features/course/domain/course_models.dart';

part 'course_controller.g.dart';

@riverpod
class CourseController extends _$CourseController {
  @override
  FutureOr<List<Course>> build() async {
    return _fetchCourses();
  }

  Future<List<Course>> _fetchCourses() async {
    final repo = ref.watch(courseRepositoryProvider);
    return repo.getCourses();
  }

  Future<void> createCourse(String name, String educationLevel) async {
    state = const AsyncValue.loading();
    state = await AsyncValue.guard(() async {
      final repo = ref.read(courseRepositoryProvider);
      await repo.createCourse(CreateCourseRequest(name: name, educationLevel: educationLevel));
      // Re-fetch courses after creating a new one
      return _fetchCourses();
    });
  }
}
