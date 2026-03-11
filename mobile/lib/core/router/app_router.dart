import 'package:go_router/go_router.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import 'package:klyra/features/auth/presentation/auth_controller.dart';
import 'package:klyra/features/auth/presentation/login_screen.dart';
import 'package:klyra/features/course/presentation/course_dashboard_screen.dart';
import 'package:klyra/features/course/presentation/course_detail_screen.dart';
import 'package:klyra/features/course/presentation/screens/material_summary_screen.dart';
import 'package:klyra/features/tutor/presentation/tutor_session_screen.dart';

part 'app_router.g.dart';

@riverpod
GoRouter appRouter(Ref ref) {
  final authState = ref.watch(authControllerProvider);

  return GoRouter(
    initialLocation: '/login',
    redirect: (context, state) {
      final isAuth = authState.value != null;
      final isLoggingIn = state.uri.path == '/login';

      if (!isAuth && !isLoggingIn) return '/login';
      if (isAuth && isLoggingIn) return '/home';

      return null;
    },
    routes: [
      GoRoute(path: '/login', builder: (context, state) => const LoginScreen()),
      GoRoute(
        path: '/home',
        builder: (context, state) => const CourseDashboardScreen(),
      ),
      GoRoute(
        path: '/course/:courseId',
        builder: (context, state) {
          final courseId = state.pathParameters['courseId']!;
          return CourseDetailScreen(courseId: courseId);
        },
      ),
      GoRoute(
        path: '/tutor/:courseId',
        builder: (context, state) {
          final courseId = state.pathParameters['courseId']!;
          return TutorSessionScreen(courseId: courseId);
        },
      ),
      GoRoute(
        path: '/tutor/:courseId/:topicId',
        builder: (context, state) {
          final courseId = state.pathParameters['courseId']!;
          final topicId = state.pathParameters['topicId']!;
          return TutorSessionScreen(courseId: courseId, topicId: topicId);
        },
      ),
      GoRoute(
        path: '/course/:courseId/topic/:topicId/summary',
        builder: (context, state) {
          final courseId = state.pathParameters['courseId']!;
          final topicId = state.pathParameters['topicId']!;
          return MaterialSummaryScreen(courseId: courseId, topicId: topicId);
        },
      ),
    ],
  );
}
