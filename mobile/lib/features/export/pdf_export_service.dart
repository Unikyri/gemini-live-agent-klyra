import 'dart:io';

import 'package:pdf/pdf.dart';
import 'package:pdf/widgets.dart' as pw;
import 'package:share_plus/share_plus.dart';

import 'package:klyra/features/course/domain/interpretation_models.dart';

class PdfExportService {
  final Future<Directory> Function() _tempDirProvider;
  final Future<void> Function(List<XFile> files, {String? subject}) _shareFn;

  PdfExportService({
    Future<Directory> Function()? tempDirProvider,
    Future<void> Function(List<XFile> files, {String? subject})? shareFn,
  })  : _tempDirProvider = tempDirProvider ?? _defaultTempDirProvider,
        _shareFn = shareFn ?? _defaultShareFn;

  static Future<Directory> _defaultTempDirProvider() async {
    // Avoid path_provider in unit tests (MissingPluginException).
    return Directory.systemTemp;
  }

  static Future<void> _defaultShareFn(List<XFile> files, {String? subject}) async {
    await Share.shareXFiles(files, subject: subject);
  }

  Future<void> exportMaterialAsPdf({
    required InterpretationResult interpretation,
    required List<MaterialCorrection> corrections,
    required String courseName,
    required String materialName,
    required DateTime createdAt,
  }) async {
    final doc = pw.Document();

    final corrByIndex = {for (final c in corrections) c.blockIndex: c};

    doc.addPage(
      pw.MultiPage(
        pageFormat: PdfPageFormat.a4,
        margin: const pw.EdgeInsets.all(24),
        build: (context) => [
          pw.Text(
            'Guía de estudio',
            style: pw.TextStyle(
              fontSize: 20,
              fontWeight: pw.FontWeight.bold,
            ),
          ),
          pw.SizedBox(height: 8),
          pw.Text('Curso: $courseName'),
          pw.Text('Material: $materialName'),
          pw.Text('Fecha: ${createdAt.toIso8601String()}'),
          if ((interpretation.summary ?? '').trim().isNotEmpty) ...[
            pw.SizedBox(height: 12),
            pw.Text(
              'Resumen',
              style: pw.TextStyle(
                fontSize: 14,
                fontWeight: pw.FontWeight.bold,
              ),
            ),
            pw.SizedBox(height: 6),
            pw.Text(interpretation.summary!.trim()),
          ],
          pw.SizedBox(height: 16),
          pw.Text(
            'Bloques',
            style: pw.TextStyle(
              fontSize: 14,
              fontWeight: pw.FontWeight.bold,
            ),
          ),
          pw.SizedBox(height: 8),
          ...interpretation.blocks.map((b) {
            final corr = corrByIndex[b.blockIndex];
            final content = corr?.correctedText ?? _blockPrimaryText(b);
            final label = _blockLabel(b.blockType);
            final isCorrected = corr != null;
            return pw.Container(
              margin: const pw.EdgeInsets.only(bottom: 10),
              padding: const pw.EdgeInsets.all(10),
              decoration: pw.BoxDecoration(
                border: pw.Border.all(
                  color: isCorrected ? PdfColors.green : PdfColors.grey400,
                  width: 1,
                ),
                borderRadius: pw.BorderRadius.circular(6),
              ),
              child: pw.Column(
                crossAxisAlignment: pw.CrossAxisAlignment.start,
                children: [
                  pw.Row(
                    children: [
                      pw.Text(
                        'Bloque ${b.blockIndex} · $label',
                        style: pw.TextStyle(
                          fontSize: 11,
                          fontWeight: pw.FontWeight.bold,
                        ),
                      ),
                      if (isCorrected) ...[
                        pw.SizedBox(width: 8),
                        pw.Text(
                          '(Corregido)',
                          style: const pw.TextStyle(
                            fontSize: 10,
                            color: PdfColors.green,
                          ),
                        ),
                      ],
                    ],
                  ),
                  pw.SizedBox(height: 6),
                  pw.Text(
                    content.isEmpty ? '(vacío)' : content,
                    style: pw.TextStyle(
                      fontSize: 11,
                      fontStyle: (b.blockType == InterpretationBlockType.figure)
                          ? pw.FontStyle.italic
                          : pw.FontStyle.normal,
                    ),
                  ),
                ],
              ),
            );
          }),
        ],
      ),
    );

    final dir = await _tempDirProvider();
    final safeName = materialName.replaceAll(RegExp(r'[^a-zA-Z0-9._-]+'), '_');
    final file = File('${dir.path}/klyra_$safeName.pdf');
    await file.writeAsBytes(await doc.save(), flush: true);

    await _shareFn(
      [XFile(file.path, mimeType: 'application/pdf', name: file.uri.pathSegments.last)],
      subject: 'Guía de estudio - $materialName',
    );
  }
}

String _blockLabel(InterpretationBlockType t) {
  switch (t) {
    case InterpretationBlockType.text:
      return 'Texto';
    case InterpretationBlockType.equation:
      return 'Ecuación';
    case InterpretationBlockType.figure:
      return 'Figura';
    case InterpretationBlockType.audioTranscript:
      return 'Transcripción';
  }
}

String _blockPrimaryText(InterpretationBlock b) {
  switch (b.blockType) {
    case InterpretationBlockType.equation:
      return b.latex ?? b.content ?? '';
    case InterpretationBlockType.figure:
      return b.figureDescription ?? b.content ?? '';
    case InterpretationBlockType.audioTranscript:
      return b.content ?? '';
    case InterpretationBlockType.text:
      return b.content ?? '';
  }
}

