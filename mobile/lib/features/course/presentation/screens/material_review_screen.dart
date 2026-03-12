import 'package:flutter/material.dart';
import 'package:flutter_math_fork/flutter_math.dart';
import 'package:flutter_markdown/flutter_markdown.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:klyra/features/course/domain/interpretation_models.dart';
import 'package:klyra/features/course/presentation/material_review_controller.dart';
import 'package:klyra/features/export/presentation/export_button.dart';

class MaterialReviewScreen extends ConsumerWidget {
  final String courseId;
  final String topicId;
  final String materialId;
  final String materialName;
  final String courseName;

  const MaterialReviewScreen({
    super.key,
    required this.courseId,
    required this.topicId,
    required this.materialId,
    required this.materialName,
    this.courseName = 'Curso',
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncState = ref.watch(
      materialReviewControllerProvider(
        courseId: courseId,
        topicId: topicId,
        materialId: materialId,
      ),
    );
    final ctrl = ref.read(
      materialReviewControllerProvider(
        courseId: courseId,
        topicId: topicId,
        materialId: materialId,
      ).notifier,
    );

    return Scaffold(
      appBar: AppBar(
        title: Text('Revisión: $materialName'),
        actions: [
          IconButton(
            tooltip: 'Recargar',
            onPressed: asyncState.isLoading
                ? null
                : () => ref.invalidate(
                      materialReviewControllerProvider(
                        courseId: courseId,
                        topicId: topicId,
                        materialId: materialId,
                      ),
                    ),
            icon: const Icon(Icons.refresh_rounded),
          ),
        ],
      ),
      body: asyncState.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _ErrorState(message: e.toString()),
        data: (state) => _BlocksList(
          blocks: state.interpretation.blocks,
          corrections: state.corrections,
          onEdit: (block, suggested) async {
            final corrected = await _showCorrectionDialog(
              context,
              block: block,
              suggested: suggested,
            );
            if (corrected == null) return;
            await ctrl.submitCorrection(
              blockIndex: block.blockIndex,
              originalText: suggested,
              correctedText: corrected,
            );
            if (context.mounted) {
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(
                  content: Text('Corrección guardada.'),
                ),
              );
            }
          },
        ),
      ),
      bottomNavigationBar: asyncState.maybeWhen(
        data: (state) => Padding(
          padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
          child: ExportButton(
            interpretation: state.interpretation,
            corrections: state.corrections,
            courseName: courseName,
            materialName: materialName,
          ),
        ),
        orElse: () => null,
      ),
    );
  }
}

class _BlocksList extends StatelessWidget {
  final List<InterpretationBlock> blocks;
  final List<MaterialCorrection> corrections;
  final Future<void> Function(InterpretationBlock block, String suggested)
      onEdit;

  const _BlocksList({
    required this.blocks,
    required this.corrections,
    required this.onEdit,
  });

  @override
  Widget build(BuildContext context) {
    final byIndex = {for (final c in corrections) c.blockIndex: c};

    return ListView.separated(
      padding: const EdgeInsets.all(12),
      itemCount: blocks.length,
      separatorBuilder: (_, __) => const SizedBox(height: 12),
      itemBuilder: (context, i) {
        final b = blocks[i];
        final corr = byIndex[b.blockIndex];
        final displayText = corr?.correctedText ?? _blockPrimaryText(b);
        final isCorrected = corr != null;

        return InkWell(
          onTap: () => onEdit(b, displayText),
          borderRadius: BorderRadius.circular(12),
          child: Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: Colors.white.withValues(alpha: 0.03),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(
                color: isCorrected ? Colors.greenAccent : Colors.white12,
              ),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Text(
                      'Bloque ${b.blockIndex}',
                      style: Theme.of(context).textTheme.labelLarge,
                    ),
                    const SizedBox(width: 8),
                    if (isCorrected)
                      const Chip(
                        label: Text('Corregido'),
                        visualDensity: VisualDensity.compact,
                      ),
                  ],
                ),
                const SizedBox(height: 8),
                _BlockBody(block: b, overrideText: corr?.correctedText),
              ],
            ),
          ),
        );
      },
    );
  }
}

class _BlockBody extends StatelessWidget {
  final InterpretationBlock block;
  final String? overrideText;

  const _BlockBody({required this.block, this.overrideText});

  @override
  Widget build(BuildContext context) {
    switch (block.blockType) {
      case InterpretationBlockType.equation:
        final latex = overrideText ?? block.latex ?? block.content ?? '';
        return Math.tex(
          latex.isEmpty ? r'\text{(sin ecuación)}' : latex,
          textStyle: Theme.of(context).textTheme.bodyLarge,
        );
      case InterpretationBlockType.figure:
        final text = overrideText ?? block.figureDescription ?? block.content ?? '';
        return Text(
          text.isEmpty ? '(sin descripción)' : text,
          style: Theme.of(context)
              .textTheme
              .bodyMedium
              ?.copyWith(fontStyle: FontStyle.italic),
        );
      case InterpretationBlockType.audioTranscript:
        final text = overrideText ?? block.content ?? '';
        return Text(
          text.isEmpty ? '(sin transcripción)' : text,
          style: Theme.of(context).textTheme.bodyMedium,
        );
      case InterpretationBlockType.text:
        final md = overrideText ?? block.content ?? '';
        return MarkdownBody(data: md.isEmpty ? '(vacío)' : md);
    }
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

class _ErrorState extends StatelessWidget {
  final String message;
  const _ErrorState({required this.message});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Text(
          message,
          style: Theme.of(context)
              .textTheme
              .bodyMedium
              ?.copyWith(color: Colors.redAccent),
        ),
      ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState();
  @override
  Widget build(BuildContext context) {
    return const Center(
      child: Text('Aún no hay interpretación para este material.'),
    );
  }
}

Future<String?> _showCorrectionDialog(
  BuildContext context, {
  required InterpretationBlock block,
  required String suggested,
}) async {
  final controller = TextEditingController(text: suggested);
  return showDialog<String>(
    context: context,
    builder: (ctx) => AlertDialog(
      title: Text('Corregir bloque ${block.blockIndex}'),
      content: TextField(
        controller: controller,
        maxLines: 6,
        decoration: const InputDecoration(
          labelText: 'Corrección',
          hintText: 'Ej: La variable es Z, no 2',
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(ctx).pop(),
          child: const Text('Cancelar'),
        ),
        ElevatedButton(
          onPressed: () => Navigator.of(ctx).pop(controller.text.trim()),
          child: const Text('Guardar'),
        ),
      ],
    ),
  );
}

