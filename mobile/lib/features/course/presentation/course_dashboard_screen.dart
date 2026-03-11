import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:klyra/core/widgets/avatar_image.dart';
import 'package:klyra/features/course/presentation/course_controller.dart';
import 'package:klyra/features/course/domain/course_models.dart';
import 'package:klyra/features/course/presentation/create_course_modal.dart';

class CourseDashboardScreen extends ConsumerWidget {
  const CourseDashboardScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final coursesAsync = ref.watch(courseControllerProvider);
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Your Learning Instances'),
        elevation: 0,
        backgroundColor: Colors.transparent,
      ),
      body: coursesAsync.when(
        data: (courses) {
          if (courses.isEmpty) {
            return _buildEmptyState(theme);
          }
          return RefreshIndicator(
            onRefresh: () async {
              // Ignore the returned value using a wrapper, compatible with both Future<void> and Future<List<Course>> signatures 
              await ref.refresh(courseControllerProvider.future);
            },
            child: ListView.separated(
              padding: const EdgeInsets.all(24),
              itemCount: courses.length,
              separatorBuilder: (_, __) => const SizedBox(height: 16),
              itemBuilder: (context, index) {
                final course = courses[index];
                return _CourseCard(course: course);
              },
            ),
          );
        },
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (err, stack) => Center(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Icon(Icons.error_outline, size: 64, color: Colors.redAccent),
              const SizedBox(height: 16),
              Text('Failed to load courses', style: theme.textTheme.titleLarge),
              // Do not expose internal error details to the user (security: info disclosure)
              Text('Something went wrong. Please try again.', style: theme.textTheme.bodyMedium),
              const SizedBox(height: 24),
              ElevatedButton(
                onPressed: () => ref.refresh(courseControllerProvider),
                child: const Text('Retry'),
              )
            ],
          ),
        ),
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => CreateCourseModal.show(context),
        icon: const Icon(Icons.add),
        label: const Text('New Course'),
      ),
    );
  }

  Widget _buildEmptyState(ThemeData theme) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(Icons.school_outlined, size: 100, color: Colors.white.withOpacity(0.2)),
          const SizedBox(height: 24),
          Text('No courses yet.', style: theme.textTheme.titleLarge),
          const SizedBox(height: 8),
          Text(
            'Create your first learning instance to begin.',
            style: theme.textTheme.bodyMedium,
          ),
        ],
      ),
    );
  }
}

class _CourseCard extends ConsumerWidget {
  final Course course;

  const _CourseCard({required this.course});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    
    // Status visual logic
    Color statusColor;
    IconData statusIcon;
    bool isAnimating = false;

    switch (course.avatarStatus) {
      case 'ready':
        statusColor = theme.colorScheme.secondary; // Cyan
        statusIcon = Icons.check_circle;
        break;
      case 'processing':
      case 'pending':
        statusColor = Colors.orangeAccent;
        statusIcon = Icons.auto_mode;
        isAnimating = true;
        break;
      case 'failed':
      default:
        statusColor = theme.colorScheme.error;
        statusIcon = Icons.error;
        break;
    }

    return Container(
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: Colors.white.withOpacity(0.05)),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(0.2),
            blurRadius: 10,
            offset: const Offset(0, 4),
          )
        ],
      ),
      clipBehavior: Clip.antiAlias,
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          onTap: () => context.push('/course/${course.id}'),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              // Avatar Header Area
              Container(
                height: 120,
                decoration: const BoxDecoration(
                  color: Colors.black12,
                ),
                child: course.avatarStatus == 'ready'
                    ? Center(
                        child: AvatarImage(
                          avatarUrl: course.avatarModelUrl,
                          status: course.avatarStatus,
                          size: 120,
                          fit: BoxFit.cover,
                        ),
                      )
                    : Center(
                        child: isAnimating
                            ? const CircularProgressIndicator(color: Colors.orangeAccent)
                            : Icon(statusIcon, color: statusColor, size: 40),
                      ),
              ),
              
              // Details Area
              Padding(
                padding: const EdgeInsets.all(20),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            course.name,
                            style: theme.textTheme.titleLarge?.copyWith(fontWeight: FontWeight.bold),
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                          ),
                          const SizedBox(height: 4),
                          Text(
                            course.educationLevel.toUpperCase(),
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: theme.colorScheme.primary,
                              letterSpacing: 1.2,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(width: 8),
                    PopupMenuButton<String>(
                      icon: const Icon(Icons.more_vert),
                      onSelected: (value) {
                        if (value == 'edit') _showEditCourseDialog(context, ref);
                        if (value == 'delete') _showDeleteCourseDialog(context, ref);
                      },
                      itemBuilder: (_) => [
                        const PopupMenuItem(value: 'edit', child: Text('Editar nombre')),
                        const PopupMenuItem(value: 'delete', child: Text('Eliminar curso')),
                      ],
                    ),
                    const SizedBox(width: 8),
                    // Status Badge
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                      decoration: BoxDecoration(
                        color: statusColor.withOpacity(0.1),
                        borderRadius: BorderRadius.circular(20),
                        border: Border.all(color: statusColor.withOpacity(0.5)),
                      ),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(statusIcon, size: 14, color: statusColor),
                          const SizedBox(width: 4),
                          Text(
                            course.avatarStatus.toUpperCase(),
                            style: theme.textTheme.labelSmall?.copyWith(
                              color: statusColor,
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                        ],
                      ),
                    )
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  void _showEditCourseDialog(BuildContext context, WidgetRef ref) {
    final nameController = TextEditingController(text: course.name);
    String selectedLevel = course.educationLevel;
    const levels = ['elementary', 'middle_school', 'high_school', 'university', 'postgraduate', 'other'];
    showDialog(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setState) => AlertDialog(
          title: const Text('Editar nombre'),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                TextField(
                  controller: nameController,
                  decoration: const InputDecoration(labelText: 'Nombre del curso'),
                  autofocus: true,
                ),
                const SizedBox(height: 16),
                DropdownButtonFormField<String>(
                  value: selectedLevel,
                  decoration: const InputDecoration(labelText: 'Nivel educativo'),
                  items: levels.map((l) => DropdownMenuItem(value: l, child: Text(l))).toList(),
                  onChanged: (v) => setState(() => selectedLevel = v ?? selectedLevel),
                ),
              ],
            ),
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancelar')),
            ElevatedButton(
              onPressed: () async {
                final name = nameController.text.trim();
                if (name.isEmpty) return;
                Navigator.pop(ctx);
                await ref.read(courseControllerProvider.notifier).updateCourse(
                  course.id,
                  name: name,
                  educationLevel: selectedLevel,
                );
              },
              child: const Text('Guardar'),
            ),
          ],
        ),
      ),
    );
  }

  void _showDeleteCourseDialog(BuildContext context, WidgetRef ref) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('¿Eliminar curso?'),
        content: Text(
          'Se eliminará el curso «${course.name}» y todos sus temas, materiales y contenido asociado. Esta acción no se puede deshacer.',
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancelar')),
          TextButton(
            onPressed: () async {
              Navigator.pop(ctx);
              await ref.read(courseControllerProvider.notifier).deleteCourse(course.id);
            },
            style: TextButton.styleFrom(foregroundColor: Colors.red),
            child: const Text('Eliminar'),
          ),
        ],
      ),
    );
  }
}
