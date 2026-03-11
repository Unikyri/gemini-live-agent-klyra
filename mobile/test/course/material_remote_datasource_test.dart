import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:dio/src/adapters/io_adapter.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:klyra/features/course/data/material_remote_datasource.dart';

class _FakeAdapter extends IOHttpClientAdapter {
  RequestOptions? lastRequest;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    lastRequest = options;
    final payload = jsonEncode({
      'id': 'm1',
      'topic_id': 't1',
      'format_type': 'pdf',
      'storage_url': 'http://localhost:8080/static/materials/m1.pdf',
      'extracted_text': null,
      'status': 'pending',
      'original_name': 'm1.pdf',
      'size_bytes': 12,
      'created_at': DateTime.utc(2026, 1, 1).toIso8601String(),
      'updated_at': DateTime.utc(2026, 1, 1).toIso8601String(),
    });
    return ResponseBody.fromString(
      payload,
      201,
      headers: {Headers.contentTypeHeader: ['application/json']},
    );
  }
}

void main() {
  group('MaterialRemoteDataSource.uploadMaterial', () {
    test('uploads web bytes (no path) successfully', () async {
      final dio = Dio();
      final adapter = _FakeAdapter();
      dio.httpClientAdapter = adapter;
      final ds = MaterialRemoteDataSource(dio);

      final file = PlatformFile(
        name: 'doc.pdf',
        size: 4,
        bytes: Uint8List.fromList([1, 2, 3, 4]),
      );

      final material = await ds.uploadMaterial('c1', 't1', file);

      expect(material.id, 'm1');
      expect(adapter.lastRequest?.method, 'POST');
      expect(adapter.lastRequest?.path, '/courses/c1/topics/t1/materials');
      expect(adapter.lastRequest?.data, isA<FormData>());
      final formData = adapter.lastRequest?.data as FormData;
      expect(formData.files.first.value.contentType.toString(), 'application/pdf');
    });

    test('uploads file path successfully', () async {
      final dio = Dio();
      final adapter = _FakeAdapter();
      dio.httpClientAdapter = adapter;
      final ds = MaterialRemoteDataSource(dio);

      final dir = await Directory.systemTemp.createTemp('klyra_upload_test');
      final fileOnDisk = File('${dir.path}${Platform.pathSeparator}doc.txt');
      await fileOnDisk.writeAsString('hello world');

      final file = PlatformFile(
        name: 'doc.txt',
        size: await fileOnDisk.length(),
        path: fileOnDisk.path,
      );

      final material = await ds.uploadMaterial('c1', 't1', file);

      expect(material.originalName, 'm1.pdf');
      expect(adapter.lastRequest?.method, 'POST');
      expect(adapter.lastRequest?.path, '/courses/c1/topics/t1/materials');
      final formData = adapter.lastRequest?.data as FormData;
      expect(formData.files.first.value.contentType.toString(), 'text/plain');
    });

    test('throws when file has no bytes and no path', () async {
      final dio = Dio();
      final ds = MaterialRemoteDataSource(dio);

      final file = PlatformFile(name: 'invalid.bin', size: 0);

      expect(
        () => ds.uploadMaterial('c1', 't1', file),
        throwsA(isA<Exception>()),
      );
    });
  });
}
