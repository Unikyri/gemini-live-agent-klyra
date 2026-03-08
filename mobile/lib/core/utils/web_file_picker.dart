/// Web-compatible file picker utilities for Klyra
///
/// Provides a unified interface for file picking that works across
/// all platforms, with special handling for web browsers.
///
/// Usage:
/// ```dart
/// import 'package:klyra/core/utils/web_file_picker.dart';
///
/// final result = await WebFilePicker.pickFile(
///   allowedExtensions: ['pdf', 'txt'],
///   type: FileType.custom,
/// );
///
/// if (result != null) {
///   final bytes = result.bytes;
///   final name = result.name;
/// }
/// ```
library;

import 'dart:typed_data';
import 'package:file_picker/file_picker.dart';
import 'package:klyra/core/utils/platform_utils.dart';

/// Wrapper around file_picker package with web-specific optimizations
class WebFilePicker {
  /// Pick a single file with optional type filtering
  ///
  /// On web, this uses the browser's file input dialog.
  /// On mobile/desktop, this uses the native file picker.
  ///
  /// Returns null if the user cancels the picker.
  static Future<PickedFileResult?> pickFile({
    FileType type = FileType.any,
    List<String>? allowedExtensions,
    bool allowMultiple = false,
  }) async {
    try {
      final result = await FilePicker.platform.pickFiles(
        type: type,
        allowedExtensions: allowedExtensions,
        allowMultiple: allowMultiple,
        withData: PlatformUtils.isWeb, // Load file bytes on web
      );

      if (result == null || result.files.isEmpty) {
        return null;
      }

      final file = result.files.first;

      // On web, file.bytes is already populated
      // On mobile/desktop, we have file.path instead
      return PickedFileResult(
        name: file.name,
        extension: file.extension,
        size: file.size,
        bytes: file.bytes,
        path: file.path,
      );
    } catch (e) {
      throw FilePickerException('Failed to pick file: $e');
    }
  }

  /// Pick an image file (common use case)
  static Future<PickedFileResult?> pickImage() async {
    return pickFile(
      type: FileType.image,
    );
  }

  /// Pick a PDF file
  static Future<PickedFileResult?> pickPDF() async {
    return pickFile(
      type: FileType.custom,
      allowedExtensions: ['pdf'],
    );
  }

  /// Pick a document file (PDF, TXT, MD)
  static Future<PickedFileResult?> pickDocument() async {
    return pickFile(
      type: FileType.custom,
      allowedExtensions: ['pdf', 'txt', 'md'],
    );
  }
}

/// Represents a picked file with platform-agnostic access
class PickedFileResult {
  final String name;
  final String? extension;
  final int size;
  final Uint8List? bytes; // Available on web, or if withData=true
  final String? path; // Available on mobile/desktop

  PickedFileResult({
    required this.name,
    this.extension,
    required this.size,
    this.bytes,
    this.path,
  });

  /// Returns true if file data is available in memory
  bool get hasBytes => bytes != null;

  /// Returns true if file path is available on disk
  bool get hasPath => path != null;

  /// Returns file size in megabytes
  double get sizeInMB => size / (1024 * 1024);

  /// Returns file size in kilobytes
  double get sizeInKB => size / 1024;

  /// Converts to file_picker PlatformFile for shared upload pipelines.
  PlatformFile toPlatformFile() {
    return PlatformFile(
      name: name,
      size: size,
      bytes: bytes,
      path: path,
    );
  }
}

/// Exception thrown when file picking fails
class FilePickerException implements Exception {
  final String message;
  FilePickerException(this.message);

  @override
  String toString() => 'FilePickerException: $message';
}