import 'dart:io';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:file_picker/file_picker.dart';
import 'package:klyra/features/course/domain/material_models.dart';
import 'package:klyra/features/course/presentation/material_controller.dart';

/// Displays all materials for a given topic and allows uploading new ones.
class MaterialListView extends ConsumerWidget {
  final String courseId;
  final String topicId;
  final String topicTitle;

  const MaterialListView({
    super.key,
    required this.courseId,
    required this.topicId,
    required this.topicTitle,
  });

  Future<void> _pickAndUpload(BuildContext context, WidgetRef ref) async {
    // SECURITY: restrict allowed extensions at the client level before picking
    final result = await FilePicker.platform.pickFiles(
      type: FileType.custom,
      allowedExtensions: ['pdf', 'txt', 'md'],
    );
    if (result == null || result.files.single.path == null) return;

    final file = File(result.files.single.path!);
    await ref
        .read(materialControllerProvider(
            courseId: courseId, topicId: topicId).notifier)
        .uploadFile(file);

    if (context.mounted) {
      final state = ref.read(
          materialControllerProvider(courseId: courseId, topicId: topicId));
      if (state.hasError) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Upload failed. Check the file type and size (max 20 MB).'),
            backgroundColor: Colors.redAccent,
          ),
        );
      } else {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Material uploaded successfully!'),
            backgroundColor: Colors.green,
          ),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final materialsAsync = ref.watch(
        materialControllerProvider(courseId: courseId, topicId: topicId));
    final theme = Theme.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Topic header row
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(topicTitle,
                style: theme.textTheme.titleMedium
                    ?.copyWith(fontWeight: FontWeight.bold)),
            IconButton(
              tooltip: 'Upload material',
              icon: Icon(Icons.upload_file_rounded,
                  color: theme.colorScheme.primary),
              onPressed: materialsAsync.isLoading
                  ? null
                  : () => _pickAndUpload(context, ref),
            ),
          ],
        ),
        const Divider(height: 8),
        materialsAsync.when(
          data: (materials) {
            if (materials.isEmpty) {
              return Padding(
                padding: const EdgeInsets.symmetric(vertical: 12),
                child: Text(
                  'No materials yet. Tap ↑ to upload a PDF, TXT, or MD.',
                  style: theme.textTheme.bodySmall
                      ?.copyWith(color: Colors.white38),
                ),
              );
            }
            return ListView.builder(
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              itemCount: materials.length,
              itemBuilder: (_, i) => _MaterialTile(material: materials[i]),
            );
          },
          loading: () =>
              const Padding(
                padding: EdgeInsets.all(16),
                child: Center(child: CircularProgressIndicator()),
              ),
          error: (_, __) => Padding(
            padding: const EdgeInsets.symmetric(vertical: 8),
            child: Text('Could not load materials.',
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: Colors.redAccent)),
          ),
        ),
      ],
    );
  }
}

class _MaterialTile extends StatelessWidget {
  final Material material;
  const _MaterialTile({required this.material});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final status = material.status;

    Color statusColor;
    IconData statusIcon;
    if (status.isReady) {
      statusColor = Colors.greenAccent;
      statusIcon = Icons.check_circle_outline;
    } else if (status.isProcessing) {
      statusColor = Colors.orangeAccent;
      statusIcon = Icons.hourglass_top_rounded;
    } else {
      statusColor = Colors.redAccent;
      statusIcon = Icons.error_outline;
    }

    final IconData fileIcon = switch (material.formatType) {
      MaterialFormatType.pdf => Icons.picture_as_pdf_rounded,
      MaterialFormatType.txt => Icons.article_rounded,
      MaterialFormatType.md => Icons.code_rounded,
      MaterialFormatType.audio => Icons.headphones_rounded,
    };

    return ListTile(
      contentPadding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
      leading: Icon(fileIcon, color: theme.colorScheme.primary, size: 28),
      title: Text(
        material.originalName,
        style: theme.textTheme.bodyMedium
            ?.copyWith(fontWeight: FontWeight.w500),
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
      ),
      subtitle: Text(
        '${(material.sizeBytes / 1024).toStringAsFixed(1)} KB',
        style: theme.textTheme.bodySmall?.copyWith(color: Colors.white38),
      ),
      trailing: Chip(
        label: Text(
          status.name.toUpperCase(),
          style: theme.textTheme.labelSmall
              ?.copyWith(color: statusColor, fontWeight: FontWeight.bold),
        ),
        avatar: Icon(statusIcon, size: 14, color: statusColor),
        backgroundColor: statusColor.withOpacity(0.1),
        side: BorderSide(color: statusColor.withOpacity(0.4)),
        padding: const EdgeInsets.symmetric(horizontal: 4),
      ),
    );
  }
}
