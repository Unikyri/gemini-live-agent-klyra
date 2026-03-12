import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:klyra/features/course/domain/interpretation_models.dart';
import 'package:klyra/features/export/pdf_export_service.dart';
import 'package:share_plus_platform_interface/share_plus_platform_interface.dart';

class _FakeSharePlatform extends SharePlatform {
  int calls = 0;
  List<XFile>? lastFiles;

  @override
  Future<ShareResult> shareXFiles(
    List<XFile> files, {
    String? subject,
    String? text,
    List<String>? emails,
  }) async {
    calls += 1;
    lastFiles = files;
    return ShareResult('unavailable', ShareResultStatus.unavailable);
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('PdfExportService genera archivo y llama share', () async {
    var shareCalls = 0;
    List<XFile>? lastFiles;

    final interpretation = InterpretationResult.fromJson({
      'summary': 'resumen',
      'blocks': [
        {'block_index': 0, 'block_type': 'text', 'content': 'hola'},
        {'block_index': 1, 'block_type': 'equation', 'latex': r'x^2'},
      ],
    });

    final service = PdfExportService(
      tempDirProvider: () async => Directory.systemTemp,
      shareFn: (files, {subject}) async {
        shareCalls += 1;
        lastFiles = files;
      },
    );
    await service.exportMaterialAsPdf(
      interpretation: interpretation,
      corrections: const [],
      courseName: 'Curso',
      materialName: 'Material',
      createdAt: DateTime(2026, 1, 1),
    );

    expect(shareCalls, 1);
    expect(lastFiles, isNotNull);
    expect(lastFiles!.single.path.endsWith('.pdf'), true);
    expect(File(lastFiles!.single.path).existsSync(), true);
  });
}

