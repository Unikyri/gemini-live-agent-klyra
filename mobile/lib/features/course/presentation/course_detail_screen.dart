import 'dart:ui';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:klyra/core/widgets/avatar_image.dart';
import 'package:klyra/features/course/domain/course_models.dart';
import 'package:klyra/features/course/presentation/course_controller.dart';
import 'package:klyra/features/course/presentation/material_list_view.dart';

class CourseDetailScreen extends ConsumerWidget {
  final String courseId;

  const CourseDetailScreen({super.key, required this.courseId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final coursesAsync = ref.watch(courseControllerProvider);
    final theme = Theme.of(context);

    return coursesAsync.when(
      loading: () =>
          const Scaffold(body: Center(child: CircularProgressIndicator())),
      error: (err, _) => Scaffold(
        body: Center(
          child: Text(
            'Could not load course.',
            style: theme.textTheme.bodyLarge,
          ),
        ),
      ),
      data: (courses) {
        // Find the course from the already-fetched list (no extra network call)
        final course = courses.where((c) => c.id == courseId).firstOrNull;
        if (course == null) {
          return Scaffold(
            appBar: AppBar(),
            body: const Center(child: Text('Course not found.')),
          );
        }
        return _CourseDetailView(course: course);
      },
    );
  }
}

class _CourseDetailView extends ConsumerStatefulWidget {
  final Course course;
  const _CourseDetailView({required this.course});

  @override
  ConsumerState<_CourseDetailView> createState() => _CourseDetailViewState();
}

class _CourseDetailViewState extends ConsumerState<_CourseDetailView>
    with SingleTickerProviderStateMixin {
  late AnimationController _fadeController;
  late Animation<double> _fadeAnim;

  @override
  void initState() {
    super.initState();
    _fadeController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 700),
    );
    _fadeAnim = CurvedAnimation(parent: _fadeController, curve: Curves.easeOut);
    _fadeController.forward();
  }

  @override
  void dispose() {
    _fadeController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final course = widget.course;

    return Scaffold(
      body: CustomScrollView(
        slivers: [
          // --- Hero App Bar with Avatar ---
          SliverAppBar(
            expandedHeight: 240,
            pinned: true,
            stretch: true,
            backgroundColor: theme.colorScheme.background,
            flexibleSpace: FlexibleSpaceBar(
              title: Text(
                course.name,
                style: const TextStyle(
                  fontWeight: FontWeight.bold,
                  shadows: [Shadow(blurRadius: 8, color: Colors.black87)],
                ),
              ),
              background: Stack(
                fit: StackFit.expand,
                children: [
                  // Gradient background
                  Container(
                    decoration: BoxDecoration(
                      gradient: LinearGradient(
                        begin: Alignment.topLeft,
                        end: Alignment.bottomRight,
                        colors: [
                          theme.colorScheme.primary.withOpacity(0.6),
                          theme.colorScheme.secondary.withOpacity(0.3),
                        ],
                      ),
                    ),
                  ),
                  // Avatar image (if ready)
                  if (course.avatarModelUrl != null &&
                      course.avatarStatus == 'ready')
                    Positioned(
                      right: 24,
                      bottom: 16,
                      child: FadeTransition(
                        opacity: _fadeAnim,
                        child: AvatarImage(
                          avatarUrl: course.avatarModelUrl,
                          status: course.avatarStatus,
                          size: 180,
                          fit: BoxFit.contain,
                        ),
                      ),
                    ),
                  // Bottom blur/fade
                  Positioned(
                    bottom: 0,
                    left: 0,
                    right: 0,
                    child: ClipRect(
                      child: BackdropFilter(
                        filter: ImageFilter.blur(sigmaX: 0, sigmaY: 4),
                        child: Container(height: 60, color: Colors.transparent),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),

          // --- Topics & Materials ---
          course.topics.isEmpty
              ? SliverFillRemaining(child: _buildEmptyTopics(theme, course))
              : SliverPadding(
                  padding: const EdgeInsets.all(24),
                  sliver: SliverList(
                    delegate: SliverChildBuilderDelegate((context, index) {
                      final topic = course.topics[index];
                      return Padding(
                        padding: const EdgeInsets.only(bottom: 32),
                        child: _TopicSection(courseId: course.id, topic: topic),
                      );
                    }, childCount: course.topics.length),
                  ),
                ),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => _showAddTopicDialog(context, ref, course),
        icon: const Icon(Icons.add),
        label: const Text('Add Topic'),
      ),
    );
  }

  Widget _buildEmptyTopics(ThemeData theme, Course course) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(
            Icons.topic_outlined,
            size: 80,
            color: Colors.white.withOpacity(0.15),
          ),
          const SizedBox(height: 16),
          Text('No topics yet.', style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          Text(
            'Topics organize your learning material.\nTap + to add a topic.',
            style: theme.textTheme.bodySmall?.copyWith(color: Colors.white38),
            textAlign: TextAlign.center,
          ),
        ],
      ),
    );
  }

  void _showAddTopicDialog(BuildContext ctx, WidgetRef ref, Course course) {
    final controller = TextEditingController();
    showDialog(
      context: ctx,
      builder: (_) => AlertDialog(
        title: const Text('New Topic'),
        content: TextField(
          controller: controller,
          autofocus: true,
          decoration: const InputDecoration(
            hintText: 'e.g. Chapter 1 — Newton\'s Laws',
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () async {
              final title = controller.text.trim();
              if (title.isEmpty) return;
              Navigator.pop(ctx);

              try {
                // Call the addTopic action from CourseController
                await ref
                    .read(courseControllerProvider.notifier)
                    .addTopic(course.id, title);

                if (ctx.mounted) {
                  ScaffoldMessenger.of(ctx).showSnackBar(
                    SnackBar(
                      content: Text('Topic "$title" added successfully!'),
                      duration: const Duration(seconds: 2),
                    ),
                  );
                }
              } catch (e) {
                if (ctx.mounted) {
                  ScaffoldMessenger.of(ctx).showSnackBar(
                    SnackBar(
                      content: Text('Failed to add topic: $e'),
                      backgroundColor: Colors.red,
                      duration: const Duration(seconds: 3),
                    ),
                  );
                }
              }
            },
            child: const Text('Add'),
          ),
        ],
      ),
    );
  }
}

class _TopicSection extends ConsumerWidget {
  final String courseId;
  final Topic topic;

  const _TopicSection({required this.courseId, required this.topic});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    return Card(
      color: theme.colorScheme.surface,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: Colors.white.withOpacity(0.07)),
      ),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    topic.title,
                    style: theme.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold),
                  ),
                ),
                PopupMenuButton<String>(
                  icon: const Icon(Icons.more_vert),
                  onSelected: (value) {
                    if (value == 'edit') _showEditTopicDialog(context, ref);
                    if (value == 'delete') _showDeleteTopicDialog(context, ref);
                  },
                  itemBuilder: (_) => [
                    const PopupMenuItem(value: 'edit', child: Text('Editar título')),
                    const PopupMenuItem(value: 'delete', child: Text('Eliminar tema')),
                  ],
                ),
              ],
            ),
            MaterialListView(
              courseId: courseId,
              topicId: topic.id,
              topicTitle: topic.title,
            ),
            const SizedBox(height: 16),
            // Start Tutor Session button — navigates to TutorSessionScreen
            OutlinedButton.icon(
              onPressed: () =>
                  context.push('/course/$courseId/topic/${topic.id}/summary'),
              icon: const Icon(Icons.play_arrow_rounded, size: 20),
              label: const Text('Review Summary & Start'),
              style: OutlinedButton.styleFrom(
                foregroundColor: theme.colorScheme.primary,
                side: BorderSide(
                  color: theme.colorScheme.primary.withOpacity(0.6),
                ),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  void _showEditTopicDialog(BuildContext context, WidgetRef ref) {
    final controller = TextEditingController(text: topic.title);
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Editar título'),
        content: TextField(
          controller: controller,
          decoration: const InputDecoration(labelText: 'Título del tema'),
          autofocus: true,
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancelar')),
          ElevatedButton(
            onPressed: () async {
              final title = controller.text.trim();
              if (title.isEmpty) return;
              Navigator.pop(ctx);
              await ref.read(courseControllerProvider.notifier).updateTopic(courseId, topic.id, title: title);
            },
            child: const Text('Guardar'),
          ),
        ],
      ),
    );
  }

  void _showDeleteTopicDialog(BuildContext context, WidgetRef ref) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('¿Eliminar tema?'),
        content: Text(
          'Se eliminará el tema «${topic.title}» y todo su material asociado. Esta acción no se puede deshacer.',
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancelar')),
          TextButton(
            onPressed: () async {
              Navigator.pop(ctx);
              await ref.read(courseControllerProvider.notifier).deleteTopic(courseId, topic.id);
            },
            style: TextButton.styleFrom(foregroundColor: Colors.red),
            child: const Text('Eliminar'),
          ),
        ],
      ),
    );
  }
}
