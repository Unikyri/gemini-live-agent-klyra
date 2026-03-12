import 'package:flutter/material.dart';
import 'package:klyra/features/course/domain/interpretation_models.dart';
import 'package:klyra/features/export/pdf_export_service.dart';

class ExportButton extends StatefulWidget {
  final InterpretationResult interpretation;
  final List<MaterialCorrection> corrections;
  final String courseName;
  final String materialName;

  const ExportButton({
    super.key,
    required this.interpretation,
    required this.corrections,
    required this.courseName,
    required this.materialName,
  });

  @override
  State<ExportButton> createState() => _ExportButtonState();
}

class _ExportButtonState extends State<ExportButton> {
  bool _busy = false;

  @override
  Widget build(BuildContext context) {
    return ElevatedButton.icon(
      onPressed: _busy
          ? null
          : () async {
              setState(() => _busy = true);
              try {
                await PdfExportService().exportMaterialAsPdf(
                  interpretation: widget.interpretation,
                  corrections: widget.corrections,
                  courseName: widget.courseName,
                  materialName: widget.materialName,
                  createdAt: DateTime.now(),
                );
              } catch (e) {
                if (context.mounted) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    SnackBar(content: Text('No se pudo exportar: $e')),
                  );
                }
              } finally {
                if (mounted) setState(() => _busy = false);
              }
            },
      icon: _busy
          ? const SizedBox(
              width: 18,
              height: 18,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          : const Icon(Icons.picture_as_pdf_rounded),
      label: Text(_busy ? 'Exportando...' : 'Exportar PDF'),
    );
  }
}

