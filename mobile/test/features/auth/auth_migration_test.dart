import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:dio/src/adapters/io_adapter.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:klyra/features/auth/data/auth_remote_datasource.dart';

class _AuthFakeAdapter extends IOHttpClientAdapter {
  RequestOptions? lastRequest;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    lastRequest = options;

    if (options.path == '/auth/login' && options.method == 'POST') {
      final payload = jsonEncode({
        'access_token': 'test_access_token',
        'refresh_token': 'test_refresh_token',
        'user': {
          'id': 'user_123',
          'email': 'guest@example.com',
          'name': 'Guest User',
          'profile_image_url': 'https://example.com/pic.jpg',
        },
        'provider': 'guest',
      });
      return ResponseBody.fromString(
        payload,
        200,
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
  group('AuthRemoteDataSource migration', () {
    test('signInAsGuest uses unified endpoint /auth/login', () async {
      final dio = Dio();
      final adapter = _AuthFakeAdapter();
      dio.httpClientAdapter = adapter;

      final datasource = AuthRemoteDataSource(dio, null);

      final result = await datasource.signInAsGuest();

      expect(result.accessToken, 'test_access_token');
      expect(result.refreshToken, 'test_refresh_token');
      expect(result.user?.email, 'guest@example.com');
      expect(adapter.lastRequest?.path, '/auth/login');
      expect(adapter.lastRequest?.method, 'POST');
      expect(adapter.lastRequest?.data, isA<Map<String, dynamic>>());

      final payload = adapter.lastRequest?.data as Map<String, dynamic>;
      expect(payload['provider'], 'guest');
      expect(payload['email'], 'guest@example.com');
      expect(payload['name'], 'Guest User');
    });

    test('signInAsGuest keeps public API for custom email and name', () async {
      final dio = Dio();
      final adapter = _AuthFakeAdapter();
      dio.httpClientAdapter = adapter;

      final datasource = AuthRemoteDataSource(dio, null);

      final result = await datasource.signInAsGuest(
        email: 'custom@example.com',
        name: 'Custom User',
      );

      expect(result.accessToken, 'test_access_token');
      expect(adapter.lastRequest?.path, '/auth/login');

      final payload = adapter.lastRequest?.data as Map<String, dynamic>;
      expect(payload['provider'], 'guest');
      expect(payload['email'], 'custom@example.com');
      expect(payload['name'], 'Custom User');
    });
  });
}
