/// Platform detection utilities for Klyra
///
/// Provides runtime checks for the current platform (web, mobile, desktop)
/// to enable platform-specific behavior in the Flutter application.
///
/// Usage:
/// ```dart
/// import 'package:klyra/core/utils/platform_utils.dart';
///
/// if (PlatformUtils.isWeb) {
///   // Use web-compatible file picker
/// } else {
///   // Use mobile/desktop file picker
/// }
/// ```
library;

import 'package:flutter/foundation.dart'
    show TargetPlatform, defaultTargetPlatform, kIsWeb;

class PlatformUtils {
  static TargetPlatform get _platform => defaultTargetPlatform;

  /// Returns true if running on web platform (browser)
  static bool get isWeb => kIsWeb;

  /// Returns true if running on mobile platform (Android or iOS)
  static bool get isMobile =>
      !kIsWeb &&
      (_platform == TargetPlatform.android || _platform == TargetPlatform.iOS);

  /// Returns true if running on desktop platform (Windows, macOS, Linux)
  static bool get isDesktop =>
      !kIsWeb &&
      (_platform == TargetPlatform.windows ||
          _platform == TargetPlatform.macOS ||
          _platform == TargetPlatform.linux);

  /// Returns true if running on Android
  static bool get isAndroid => !kIsWeb && _platform == TargetPlatform.android;

  /// Returns true if running on iOS
  static bool get isIOS => !kIsWeb && _platform == TargetPlatform.iOS;

  /// Returns true if running on Windows
  static bool get isWindows => !kIsWeb && _platform == TargetPlatform.windows;

  /// Returns true if running on macOS
  static bool get isMacOS => !kIsWeb && _platform == TargetPlatform.macOS;

  /// Returns true if running on Linux
  static bool get isLinux => !kIsWeb && _platform == TargetPlatform.linux;

  /// Returns a human-readable string describing the current platform
  static String get platformName {
    if (kIsWeb) return 'Web';
    if (_platform == TargetPlatform.android) return 'Android';
    if (_platform == TargetPlatform.iOS) return 'iOS';
    if (_platform == TargetPlatform.windows) return 'Windows';
    if (_platform == TargetPlatform.macOS) return 'macOS';
    if (_platform == TargetPlatform.linux) return 'Linux';
    return 'Unknown';
  }

  /// Returns true if the platform supports file system access
  static bool get supportsFileSystem => !kIsWeb;

  /// Returns true if the platform requires mobile-specific permissions
  static bool get requiresMobilePermissions => isMobile;
}