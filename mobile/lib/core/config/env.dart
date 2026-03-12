import 'dart:io' as io show Platform;
import 'package:flutter/foundation.dart';

class EnvInfo {
  static const String googleWebClientId = String.fromEnvironment(
    'GOOGLE_WEB_CLIENT_ID',
    defaultValue:
        '782011204480-0eejl4shc1f9n360mln5secbeng6k5gb.apps.googleusercontent.com',
  );

  static const String _apiBaseUrlOverride = String.fromEnvironment('API_BASE_URL');

  // Android emulator uses 10.0.2.2 to reach the host machine's localhost.
  // iOS simulator, desktop, and web use 127.0.0.1 / localhost directly.
  static String get backendBaseUrl {
    if (_apiBaseUrlOverride.isNotEmpty) {
      return _apiBaseUrlOverride;
    }

    // Web always uses localhost (no Platform.io in web)
    if (kIsWeb) {
      return 'http://localhost:8080/api/v1';
    }

    // Mobile / Desktop - detect platform
    try {
      if (io.Platform.isAndroid) {
        final url = 'http://192.168.1.109:8080/api/v1';
        debugPrint('[EnvInfo] Android detected - using LAN: $url');
        return url;
      } else if (io.Platform.isIOS) {
        final url = 'http://192.168.1.109:8080/api/v1';
        debugPrint('[EnvInfo] iOS detected - using LAN: $url');
        return url;
      } else {
        final url = 'http://localhost:8080/api/v1';
        debugPrint('[EnvInfo] Desktop platform detected - using localhost: $url');
        return url;
      }
    } catch (e) {
      // If Platform access fails, fallback to localhost
      debugPrint('[EnvInfo] Platform detection failed: $e - falling back to localhost');
      return 'http://localhost:8080/api/v1';
    }
  }
}