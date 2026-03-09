import 'dart:io' as io show Platform;
import 'package:flutter/foundation.dart';

class EnvInfo {
  // Android emulator uses 10.0.2.2 to reach the host machine's localhost.
  // iOS simulator, desktop, and web use 127.0.0.1 / localhost directly.
  static String get backendBaseUrl {
    // Web always uses localhost (no Platform.io in web)
    if (kIsWeb) {
      return 'http://localhost:8080/api/v1';
    }

    // Mobile / Desktop - detect platform
    try {
      if (io.Platform.isAndroid) {
        final url = 'http://10.0.2.2:8080/api/v1';
        debugPrint('[EnvInfo] Android detected - using 10.0.2.2: $url');
        return url;
      } else if (io.Platform.isIOS) {
        final url = 'http://localhost:8080/api/v1';
        debugPrint('[EnvInfo] iOS detected - using localhost: $url');
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