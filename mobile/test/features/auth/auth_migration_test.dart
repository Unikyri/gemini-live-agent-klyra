import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:dio/src/adapters/io_adapter.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:klyra/features/auth/data/auth_remote_datasource.dart';

class _AuthAdapter extends IOHttpClientAdapter {
  _AuthAdapter({
    this.unifiedStatusCode = 200,
    this.legacyStatusCode = 200,
  });

  final int unifiedStatusCode;
  final int legacyStatusCode;
  final List<String> calledPaths = [];
  final Map<String, dynamic> requestBodyByPath = {};

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    calledPaths.add(options.path);
    requestBodyByPath[options.path] = options.data;

    if (options.path == '/auth/login') {
      if (unifiedStatusCode == 404) {
        return ResponseBody.fromString('{}', 404);
      }
      return ResponseBody.fromString(
        jsonEncode({
          'access_token': 'test_access_token',
          'refresh_token': 'test_refresh_token',
          'user': {
            'id': 'user_123',
            'email': 'guest@example.com',
            'name': 'Guest User',
            'profile_image_url': 'https://example.com/pic.jpg',
          },
          'provider': 'guest',
        }),
        unifiedStatusCode,
        headers: {Headers.contentTypeHeader: ['application/json']},
      );
    }

    if (options.path == '/auth/google-mock') {
      return ResponseBody.fromString(
        jsonEncode({
          'access_token': 'legacy_token',
          'refresh_token': 'legacy_refresh',
          'user': {
            'id': 'user_456',
            'email': 'guest@example.com',
            'name': 'Guest User',
            'profile_image_url': 'https://example.com/pic.jpg',
          },
        }),
        legacyStatusCode,
        headers: {Headers.contentTypeHeader: ['application/json']},
      );
    }

    return ResponseBody.fromString('{}', 404);
  }
}

void main() {
  group('AuthRemoteDataSource - Phase 4 Migration Tests', () {
    late Dio dio;
    late _AuthAdapter adapter;
    late AuthRemoteDataSource datasource;

    test('_signInWithProvider: Guest login uses unified endpoint (/auth/login)', () async {
      dio = Dio();
      adapter = _AuthAdapter(unifiedStatusCode: 200);
      dio.httpClientAdapter = adapter;
      datasource = AuthRemoteDataSource(dio, null);

      final result = await datasource.signInAsGuest();

      expect(adapter.calledPaths, ['/auth/login']);
      expect(
        adapter.requestBodyByPath['/auth/login'],
        {'provider': 'guest', 'email': 'guest@example.com', 'name': 'Guest User'},
      );

      expect(result.accessToken, 'test_access_token');
      expect(result.refreshToken, 'test_refresh_token');
      expect(result.user?.email, 'guest@example.com');
    });

    test('_signInWithProvider: Falls back to legacy endpoint on 404', () async {
      dio = Dio();
      adapter = _AuthAdapter(unifiedStatusCode: 404, legacyStatusCode: 200);
      dio.httpClientAdapter = adapter;
      datasource = AuthRemoteDataSource(dio, null);

      final result = await datasource.signInAsGuest();

      expect(adapter.calledPaths, ['/auth/login', '/auth/google-mock']);
      expect(
        adapter.requestBodyByPath['/auth/google-mock'],
        {'email': 'guest@example.com', 'name': 'Guest User'},
      );

      expect(result.accessToken, 'legacy_token');
    });

    test('signInAsGuest: Public API unchanged (backward compatible)', () async {
      dio = Dio();
      adapter = _AuthAdapter(unifiedStatusCode: 200);
      dio.httpClientAdapter = adapter;
      datasource = AuthRemoteDataSource(dio, null);

      final result = await datasource.signInAsGuest(
        email: 'custom@example.com',
        name: 'Custom User',
      );

      expect(adapter.calledPaths, ['/auth/login']);
      expect(
        adapter.requestBodyByPath['/auth/login'],
        {'provider': 'guest', 'email': 'custom@example.com', 'name': 'Custom User'},
      );
      expect(result.user?.email, 'guest@example.com');
    });
  });
}
