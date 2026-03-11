import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:klyra/features/course/domain/course_models.dart';
import 'package:klyra/features/course/presentation/course_controller.dart';
import 'package:klyra/features/course/presentation/course_detail_screen.dart';
import 'package:riverpod/riverpod.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class _FakeCourseController extends CourseController {
  _FakeCourseController(this._courses);

  final List<Course> _courses;

  @override
  FutureOr<List<Course>> build() => _courses;
}

void main() {
  testWidgets(
    'CourseDetailScreen muestra boton global de tutor y navega a /tutor/:courseId',
    (tester) async {
      final now = DateTime.utc(2026, 1, 1);
      final course = Course(
        id: 'course-1',
        userId: 'user-1',
        name: 'Fisica',
        educationLevel: 'secundaria',
        avatarStatus: 'ready',
        createdAt: now,
        updatedAt: now,
        topics: const [],
      );

      final router = GoRouter(
        initialLocation: '/course/course-1',
        routes: [
          GoRoute(
            path: '/course/:courseId',
            builder: (context, state) =>
                CourseDetailScreen(courseId: state.pathParameters['courseId']!),
          ),
          GoRoute(
            path: '/tutor/:courseId',
            builder: (context, state) => Scaffold(
              body: Text('Tutor ${state.pathParameters['courseId']}'),
            ),
          ),
        ],
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            courseControllerProvider.overrideWith(
              () => _FakeCourseController([course]),
            ),
          ],
          child: MaterialApp.router(routerConfig: router),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Hablar con el tutor'), findsOneWidget);
      expect(find.text('Review Summary & Start'), findsNothing);

      await tester.tap(find.text('Hablar con el tutor'));
      await tester.pumpAndSettle();

      expect(find.text('Tutor course-1'), findsOneWidget);
    },
  );
}
