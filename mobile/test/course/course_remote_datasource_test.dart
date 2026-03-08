import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:dio/src/adapters/io_adapter.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:klyra/features/course/data/course_remote_datasource.dart';

class _CourseFakeAdapter extends IOHttpClientAdapter {
  RequestOptions? lastRequest;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    lastRequest = options;

    if (options.path.endsWith('/topics') && options.method == 'POST') {
      final payload = jsonEncode({
        'id': 'topic-1',
        'course_id': 'course-1',
        'title': 'Newton Laws',
        'order_index': 0,
        'consolidated_context': null,
        'created_at': DateTime.utc(2026, 1, 1).toIso8601String(),
        'updated_at': DateTime.utc(2026, 1, 1).toIso8601String(),
      });
      return ResponseBody.fromString(
        payload,
        201,
        headers: {Headers.contentTypeHeader: ['application/json']},
      );
    }

    return ResponseBody.fromString(
      jsonEncode({'error': 'not found'}),
      404,
      headers: {Headers.contentTypeHeader: ['application/json']},
    );
  }
}

void main() {
  group('CourseRemoteDataSource.addTopic', () {
    test('sends POST /courses/:courseId/topics with title and parses topic', () async {
      final dio = Dio();
      final adapter = _CourseFakeAdapter();
      dio.httpClientAdapter = adapter;
      final ds = CourseRemoteDataSource(dio);

      final topic = await ds.addTopic('course-1', 'Newton Laws');

      expect(topic.id, 'topic-1');
      expect(topic.courseId, 'course-1');
      expect(topic.title, 'Newton Laws');
      expect(adapter.lastRequest?.method, 'POST');
      expect(adapter.lastRequest?.path, '/courses/course-1/topics');
      expect(adapter.lastRequest?.data, isA<Map<String, dynamic>>());
      expect((adapter.lastRequest?.data as Map<String, dynamic>)['title'], 'Newton Laws');
    });
  });
}
