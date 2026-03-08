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

    // Mobile / Desktop
    try {
      if (io.Platform.isAndroid) {
        return 'http://10.0.2.2:8080/api/v1';
      }
    } catch (e) {
      // If Platform access fails, fallback to localhost
    }
    return 'http://localhost:8080/api/v1';
  }
}