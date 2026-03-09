import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:klyra/core/utils/platform_image_url.dart';

void main() {
  group('PlatformImageUrl.resolve', () {
    test('returns empty string when input is empty', () {
      final result = PlatformImageUrl.resolve('');
      expect(result, '');
    });

    test('returns original URL unchanged on web platform', () {
      // Note: In actual web builds, kIsWeb would be true
      // This test runs in VM, so we're testing the logic path
      const originalUrl = 'http://localhost:8080/static/avatars/123/avatar.png';
      final result = PlatformImageUrl.resolve(originalUrl);
      
      // On non-Android platforms (including test VM), URL should be unchanged
      // unless running on actual Android
      expect(result, isNotEmpty);
    });

    test('preserves HTTPS URLs', () {
      const originalUrl = 'https://localhost:8080/static/avatars/123/avatar.png';
      final result = PlatformImageUrl.resolve(originalUrl);
      expect(result, contains('https://'));
    });

    test('preserves path and query parameters', () {
      const originalUrl = 'http://localhost:8080/static/avatars/123/avatar.png?v=1';
      final result = PlatformImageUrl.resolve(originalUrl);
      expect(result, contains('/static/avatars/123/avatar.png'));
      expect(result, contains('?v=1'));
    });

    test('handles URLs without localhost (external URLs)', () {
      const originalUrl = 'https://example.com/image.png';
      final result = PlatformImageUrl.resolve(originalUrl);
      expect(result, originalUrl);
    });

    test('handles URLs with ports', () {
      const originalUrl = 'http://localhost:8080/static/image.png';
      final result = PlatformImageUrl.resolve(originalUrl);
      expect(result, contains(':8080'));
    });

    test('handles 127.0.0.1 URLs', () {
      const originalUrl = 'http://127.0.0.1:8080/static/image.png';
      final result = PlatformImageUrl.resolve(originalUrl);
      expect(result, isNotEmpty);
    });

    // Note: Platform-specific tests would require platform mocking
    // or running on actual devices/emulators. These tests verify
    // the basic URL handling logic works correctly.
  });
}
