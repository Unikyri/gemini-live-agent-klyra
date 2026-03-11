import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:dio/src/adapters/io_adapter.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:klyra/core/network/dio_client.dart';
import 'package:klyra/features/course/data/course_remote_datasource.dart';
import 'package:klyra/features/course/data/course_repository.dart';
import 'package:klyra/features/course/domain/course_models.dart';
import 'package:klyra/features/tutor/data/gemini_live_service.dart';
import 'package:klyra/features/tutor/presentation/tutor_session_controller.dart';
import 'package:riverpod/riverpod.dart';

class _FakeContextAdapter extends IOHttpClientAdapter {
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    if (options.path == '/courses/c1/topics/t1/context') {
      return ResponseBody.fromString(
        jsonEncode({
          'context': '',
          'has_materials': false,
          'message':
              'No hay materiales para este tema. El tutor usará su conocimiento base.',
        }),
        200,
        headers: {Headers.contentTypeHeader: ['application/json']},
      );
    }
    return ResponseBody.fromString('{}', 404);
  }
}

class _FakeGeminiLiveService extends GeminiLiveService {
  _FakeGeminiLiveService() : super('');

  String? lastContextUpdate;

  @override
  void sendContextUpdate(String contextText) {
    lastContextUpdate = contextText;
  }

  @override
  Future<void> disconnect() async {}

  @override
  void dispose() {}
}

class _FakeCourseRepository extends CourseRepository {
  _FakeCourseRepository(this.course) : super(CourseRemoteDataSource(Dio()));

  final Course course;

  @override
  Future<Course> getCourse(String courseId) async => course;
}

void main() {
  test('loadTopicContext en zero-material arma contexto minimo y marca hasCurrentTopicMaterials=false', () async {
    final dio = Dio();
    dio.httpClientAdapter = _FakeContextAdapter();

    final now = DateTime.utc(2026, 1, 1);
    final fakeCourse = Course(
      id: 'c1',
      userId: 'u1',
      name: 'Matematicas',
      educationLevel: 'secundaria',
      avatarStatus: 'ready',
      createdAt: now,
      updatedAt: now,
      topics: [
        Topic(
          id: 't1',
          courseId: 'c1',
          title: 'Algebra basica',
          createdAt: now,
          updatedAt: now,
        ),
      ],
    );

    final fakeGemini = _FakeGeminiLiveService();

    final container = ProviderContainer(
      overrides: [
        dioClientProvider.overrideWithValue(dio),
        courseRepositoryProvider.overrideWithValue(
          _FakeCourseRepository(fakeCourse),
        ),
        geminiLiveServiceFactoryProvider.overrideWithValue(
          (apiKey) => fakeGemini,
        ),
      ],
    );
    addTearDown(container.dispose);
    final subscription = container.listen(
      tutorSessionControllerProvider,
      (previous, next) {},
      fireImmediately: true,
    );
    addTearDown(subscription.close);

    await container
        .read(tutorSessionControllerProvider.notifier)
        .loadTopicContext('c1', 't1');

    final state = container.read(tutorSessionControllerProvider);
    expect(state.currentTopicId, 't1');
    expect(state.hasCurrentTopicMaterials, isFalse);
    expect(
      fakeGemini.lastContextUpdate,
      contains('El estudiante quiere hablar del tema: "Algebra basica"'),
    );
  });
}
