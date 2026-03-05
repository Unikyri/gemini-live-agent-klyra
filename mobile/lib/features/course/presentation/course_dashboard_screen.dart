import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
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

class _CourseCard extends StatelessWidget {
  final Course course;

  const _CourseCard({required this.course});

  @override
  Widget build(BuildContext context) {
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
          onTap: () {
            // TODO: Navigate to Course Detail (US7)
          },
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              // Avatar Header Area
              Container(
                height: 120,
                decoration: BoxDecoration(
                  color: Colors.black12,
                  image: course.avatarModelUrl != null && course.avatarStatus == 'ready'
                      ? DecorationImage(
                          image: NetworkImage(course.avatarModelUrl!),
                          fit: BoxFit.cover,
                        )
                      : null,
                ),
                child: course.avatarStatus != 'ready'
                    ? Center(
                        child: isAnimating
                            ? const CircularProgressIndicator(color: Colors.orangeAccent)
                            : Icon(statusIcon, color: statusColor, size: 40),
                      )
                    : null,
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
                    const SizedBox(width: 16),
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
}
