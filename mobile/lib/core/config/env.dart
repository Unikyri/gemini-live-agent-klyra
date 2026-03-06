import 'dart:io';

class EnvInfo {
  // Android emulator uses 10.0.2.2 to reach the host machine's localhost.
  // iOS simulator and desktop use 127.0.0.1 / localhost directly.
  static String get backendBaseUrl {
    if (Platform.isAndroid) {
      return 'http://10.0.2.2:8080/api/v1';
    }
    return 'http://localhost:8080/api/v1';
  }
}
