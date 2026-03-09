import 'dart:io' as io show Platform;
import 'package:flutter/foundation.dart';

/// Platform-aware URL resolution for static image assets.
/// 
/// Handles platform-specific host resolution to ensure images load correctly
/// across all platforms, particularly for Android emulator which requires
/// 10.0.2.2 instead of localhost to reach the host machine.
class PlatformImageUrl {
  /// Resolves a URL to be platform-appropriate.
  /// 
  /// On Android emulator: Replaces localhost/127.0.0.1 with 10.0.2.2
  /// On other platforms: Returns URL unchanged
  /// 
  /// Example:
  /// ```dart
  /// final url = 'http://localhost:8080/static/avatars/123/avatar.png';
  /// final resolved = PlatformImageUrl.resolve(url);
  /// // On Android: 'http://10.0.2.2:8080/static/avatars/123/avatar.png'
  /// // On Web/iOS: 'http://localhost:8080/static/avatars/123/avatar.png'
  /// ```
  static String resolve(String originalUrl) {
    if (originalUrl.isEmpty) {
      return originalUrl;
    }

    // Web and iOS can use localhost directly
    if (kIsWeb) {
      return originalUrl;
    }

    try {
      // iOS, macOS, Windows, Linux use localhost
      if (io.Platform.isIOS || 
          io.Platform.isMacOS || 
          io.Platform.isWindows || 
          io.Platform.isLinux) {
        return originalUrl;
      }

      // Android emulator needs 10.0.2.2 instead of localhost
      if (io.Platform.isAndroid) {
        return originalUrl
            .replaceAll('localhost', '10.0.2.2')
            .replaceAll('127.0.0.1', '10.0.2.2');
      }
    } catch (e) {
      // If platform detection fails, return original (defensive)
      debugPrint('[PlatformImageUrl] Platform detection failed: $e');
      return originalUrl;
    }

    return originalUrl;
  }
}
