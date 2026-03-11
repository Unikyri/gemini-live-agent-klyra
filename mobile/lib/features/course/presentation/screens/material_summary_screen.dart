import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:klyra/features/course/data/course_remote_datasource.dart';
import 'package:klyra/features/course/domain/topic_readiness.dart';
import 'package:klyra/features/course/presentation/widgets/latex_markdown.dart';

class MaterialSummaryScreen extends ConsumerStatefulWidget {
  final String courseId;
  final String topicId;

  const MaterialSummaryScreen({
    super.key,
    required this.courseId,
    required this.topicId,
  });

  @override
  ConsumerState<MaterialSummaryScreen> createState() =>
      _MaterialSummaryScreenState();
}

class _MaterialSummaryScreenState extends ConsumerState<MaterialSummaryScreen> {
  TopicReadiness? _readiness;
  String? _summary;
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });

    try {
      final ds = ref.read(courseRemoteDataSourceProvider);
      final readiness = await ds.checkTopicReadiness(
        widget.courseId,
        widget.topicId,
      );
      String? summary;
      if (readiness.isReady) {
        summary = await ds.fetchTopicSummary(widget.courseId, widget.topicId);
      }

      if (!mounted) return;
      setState(() {
        _readiness = readiness;
        _summary = summary;
        _loading = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = 'Could not load summary review.';
        _loading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(title: const Text('Summary Review')),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error != null
          ? Center(child: Text(_error!, style: theme.textTheme.bodyLarge))
          : _buildBody(context, theme),
    );
  }

  Widget _buildBody(BuildContext context, ThemeData theme) {
    final readiness = _readiness;
    if (readiness == null) {
      return Center(
        child: Text('Topic not found', style: theme.textTheme.bodyLarge),
      );
    }

    if (!readiness.isReady) {
      return Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Icon(
              Icons.info_outline,
              size: 48,
              color: Colors.orangeAccent,
            ),
            const SizedBox(height: 12),
            Text(readiness.message, style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
            Text(
              'Validated: ${readiness.validatedCount} / Total: ${readiness.totalCount}',
              style:
                  theme.textTheme.bodySmall?.copyWith(color: Colors.white70),
            ),
            const SizedBox(height: 24),
            ElevatedButton.icon(
              onPressed: () =>
                  context.push('/tutor/${widget.courseId}/${widget.topicId}'),
              icon: const Icon(Icons.play_arrow_rounded),
              label: const Text('Iniciar tutor de todas formas'),
            ),
            const SizedBox(height: 12),
            TextButton.icon(
              onPressed: () => context.pop(),
              icon: const Icon(Icons.upload_file_rounded),
              label: const Text('Seguir subiendo materiales'),
            ),
          ],
        ),
      );
    }

    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: Colors.green.withOpacity(0.1),
              borderRadius: BorderRadius.circular(10),
              border: Border.all(color: Colors.green.withOpacity(0.45)),
            ),
            child: Text(
              'Context ready (${readiness.validatedCount} validated materials). Review before starting tutor.',
              style: theme.textTheme.bodyMedium,
            ),
          ),
          const SizedBox(height: 16),
          Expanded(
            child: Card(
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: LatexMarkdown(
                  markdownData: _summary ?? 'No summary available.',
                ),
              ),
            ),
          ),
          const SizedBox(height: 12),
          ElevatedButton.icon(
            onPressed: () =>
                context.push('/tutor/${widget.courseId}/${widget.topicId}'),
            icon: const Icon(Icons.play_arrow_rounded),
            label: const Text('Start Tutor Session'),
          ),
        ],
      ),
    );
  }
}
