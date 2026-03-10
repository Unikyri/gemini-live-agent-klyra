import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:dio/src/adapters/io_adapter.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:klyra/features/course/data/course_remote_datasource.dart';
import 'package:klyra/features/course/domain/topic_readiness.dart';

class _ReadinessFakeAdapter extends IOHttpClientAdapter {
  RequestOptions? lastRequest;
  Map<String, dynamic>? _readinessData;
  String? _summaryData;
  int _statusCode = 200;

  void setReadinessResponse(Map<String, dynamic> data, {int statusCode = 200}) {
    _readinessData = data;
    _statusCode = statusCode;
  }

  void setSummaryResponse(String summary, {int statusCode = 200}) {
    _summaryData = summary;
    _statusCode = statusCode;
  }

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    lastRequest = options;

    if (options.path.contains('/readiness') && options.method == 'GET') {
      if (_readinessData != null) {
        return ResponseBody.fromString(
          jsonEncode(_readinessData),
          _statusCode,
          headers: {Headers.contentTypeHeader: ['application/json']},
        );
      }
    }

    if (options.path.contains('/summary') && options.method == 'GET') {
      if (_summaryData != null) {
        return ResponseBody.fromString(
          jsonEncode({
            'summary': _summaryData,
            'material_ids': ['mat-1', 'mat-2'],
            'from_cache': true,
          }),
          _statusCode,
          headers: {Headers.contentTypeHeader: ['application/json']},
        );
      }
    }

    return ResponseBody.fromString(
      jsonEncode({'error': 'not found'}),
      404,
      headers: {Headers.contentTypeHeader: ['application/json']},
    );
  }
}

void main() {
  group('CourseRemoteDataSource.checkTopicReadiness', () {
    test('returns ready readiness when validated_count > 0', () async {
      final dio = Dio();
      final adapter = _ReadinessFakeAdapter();
      adapter.setReadinessResponse({
        'topic_id': 'topic-ready',
        'is_ready': true,
        'validated_count': 2,
        'total_count': 3,
        'message': 'Ready to start tutoring',
      });
      dio.httpClientAdapter = adapter;
      final ds = CourseRemoteDataSource(dio);

      final readiness = await ds.checkTopicReadiness('course-1', 'topic-ready');

      expect(readiness.isReady, isTrue);
      expect(readiness.validatedCount, 2);
      expect(readiness.totalCount, 3);
      expect(readiness.message, 'Ready to start tutoring');
      expect(adapter.lastRequest?.method, 'GET');
      expect(adapter.lastRequest?.path, contains('/readiness'));
    });

    test('returns not-ready readiness when validated_count == 0', () async {
      final dio = Dio();
      final adapter = _ReadinessFakeAdapter();
      adapter.setReadinessResponse({
        'topic_id': 'topic-empty',
        'is_ready': false,
        'validated_count': 0,
        'total_count': 1,
        'message': 'Upload and validate at least one material to start tutoring',
      });
      dio.httpClientAdapter = adapter;
      final ds = CourseRemoteDataSource(dio);

      final readiness = await ds.checkTopicReadiness('course-2', 'topic-empty');

      expect(readiness.isReady, isFalse);
      expect(readiness.validatedCount, 0);
      expect(readiness.totalCount, 1);
      expect(readiness.message, contains('Upload and validate'));
    });

    test('throws exception on 404 topic not found', () async {
      final dio = Dio();
      final adapter = _ReadinessFakeAdapter();
      // Don't set readiness response → adapter returns 404
      dio.httpClientAdapter = adapter;
      final ds = CourseRemoteDataSource(dio);

      expect(
        () => ds.checkTopicReadiness('course-1', 'nonexistent'),
        throwsA(isA<DioException>()),
      );
    });
  });

  group('CourseRemoteDataSource.fetchTopicSummary', () {
    test('returns summary markdown when cache hit', () async {
      final dio = Dio();
      final adapter = _ReadinessFakeAdapter();
      adapter.setSummaryResponse('## Topic Summary\\n\\nCached content with \$\$E=mc^2\$\$.');
      dio.httpClientAdapter = adapter;
      final ds = CourseRemoteDataSource(dio);

      final summary = await ds.fetchTopicSummary('course-1', 'topic-1');

      expect(summary, contains('## Topic Summary'));
      expect(summary, contains('\$\$E=mc^2\$\$'));
      expect(adapter.lastRequest?.method, 'GET');
      expect(adapter.lastRequest?.path, contains('/summary'));
    });

    test('returns summary with LaTeX warning tag', () async {
      final dio = Dio();
      final adapter = _ReadinessFakeAdapter();
      adapter.setSummaryResponse('## Summary\\n\\n[latex_warning] Se detecto expresion LaTeX invalida.');
      dio.httpClientAdapter = adapter;
      final ds = CourseRemoteDataSource(dio);

      final summary = await ds.fetchTopicSummary('course-1', 'topic-latex');

      expect(summary, contains('[latex_warning]'));
      expect(summary, contains('LaTeX invalida'));
    });

    test('throws exception on 404 topic not found', () async {
      final dio = Dio();
      final adapter = _ReadinessFakeAdapter();
      // Don't set summary response → adapter returns 404
      dio.httpClientAdapter = adapter;
      final ds = CourseRemoteDataSource(dio);

      expect(
        () => ds.fetchTopicSummary('course-1', 'nonexistent'),
        throwsA(isA<DioException>()),
      );
    });
  });
}
