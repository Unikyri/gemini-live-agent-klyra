import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mockito/mockito.dart';

import 'package:klyra/features/auth/data/auth_remote_datasource.dart';
import 'package:klyra/features/auth/domain/auth_models.dart';

class MockDio extends Mock implements Dio {}

void main() {
  group('AuthRemoteDataSource - Phase 4 Migration Tests', () {
    late MockDio mockDio;
    late AuthRemoteDataSource datasource;

    setUp(() {
      mockDio = MockDio();
      datasource = AuthRemoteDataSource(mockDio, null); // null googleSignIn since we're not testing OAuth flow
    });

    test('_signInWithProvider: Guest login uses unified endpoint (/auth/login)', () async {
      // Arrange
      final mockResponse = Response(
        data: {
          'access_token': 'test_access_token',
          'refresh_token': 'test_refresh_token',
          'user': {
            'id': 'user_123',
            'email': 'guest@example.com',
            'name': 'Guest User',
            'profile_image_url': 'https://example.com/pic.jpg',
          },
          'provider': 'guest',
        },
        statusCode: 200,
        requestOptions: RequestOptions(path: '/auth/login'),
      );

      when(mockDio.post(
        '/auth/login',
        data: anyNamed('data'),
      )).thenAnswer((_) async => mockResponse);

      // Act
      final result = await datasource.signInAsGuest();

      // Assert
      verify(mockDio.post(
        '/auth/login',
        data: {
          'provider': 'guest',
          'email': 'guest@example.com',
          'name': 'Guest User',
        },
      )).called(1);

      expect(result.accessToken, 'test_access_token');
      expect(result.refreshToken, 'test_refresh_token');
      expect(result.user?.email, 'guest@example.com');
    });

    test('_signInWithProvider: Falls back to legacy endpoint on 404', () async {
      // Arrange
      final legacyResponse = Response(
        data: {
          'access_token': 'legacy_token',
          'refresh_token': 'legacy_refresh',
          'user': {
            'id': 'user_456',
            'email': 'guest@example.com',
            'name': 'Guest User',
            'profile_image_url': 'https://example.com/pic.jpg',
          },
        },
        statusCode: 200,
        requestOptions: RequestOptions(path: '/auth/google-mock'),
      );

      // First call (unified endpoint) throws 404
      when(mockDio.post(
        '/auth/login',
        data: anyNamed('data'),
      )).thenThrow(
        DioException(
          requestOptions: RequestOptions(path: '/auth/login'),
          response: Response(
            statusCode: 404,
            requestOptions: RequestOptions(path: '/auth/login'),
          ),
        ),
      );

      // Fallback call (legacy endpoint) succeeds
      when(mockDio.post(
        '/auth/google-mock',
        data: anyNamed('data'),
      )).thenAnswer((_) async => legacyResponse);

      // Act
      final result = await datasource.signInAsGuest();

      // Assert
      // Should have tried unified endpoint first, then fallen back to legacy
      verify(mockDio.post('/auth/login', data: anyNamed('data'))).called(1);
      verify(mockDio.post(
        '/auth/google-mock',
        data: {
          'email': 'guest@example.com',
          'name': 'Guest User',
        },
      )).called(1);

      expect(result.accessToken, 'legacy_token');
    });

    test('signInAsGuest: Public API unchanged (backward compatible)', () async {
      // Verify that the public API still accepts the same parameters
      final mockResponse = Response(
        data: {
          'access_token': 'token',
          'refresh_token': 'refresh',
          'user': {
            'id': 'user',
            'email': 'custom@example.com',
            'name': 'Custom User',
            'profile_image_url': 'https://example.com/pic.jpg',
          },
        },
        statusCode: 200,
        requestOptions: RequestOptions(path: '/auth/login'),
      );

      when(mockDio.post(anyNamed('url'), data: anyNamed('data')))
          .thenAnswer((_) async => mockResponse);

      // Act & Assert - should not throw
      final result = await datasource.signInAsGuest(
        email: 'custom@example.com',
        name: 'Custom User',
      );

      expect(result.user?.email, 'custom@example.com');
      expect(result.user?.name, 'Custom User');
    });
  });
}
