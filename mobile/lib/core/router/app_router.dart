import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import 'package:klyra/features/auth/presentation/auth_controller.dart';
import 'package:klyra/features/auth/presentation/login_screen.dart';

part 'app_router.g.dart';

@riverpod
GoRouter appRouter(AppRouterRef ref) {
  final authState = ref.watch(authControllerProvider);

  return GoRouter(
    initialLocation: '/login',
    redirect: (context, state) {
      // If the user isn't authenticated, redirect to login
      final isAuth = authState.value != null;
      final isLoggingIn = state.uri.path == '/login';
      
      if (!isAuth && !isLoggingIn) return '/login';
      if (isAuth && isLoggingIn) return '/home';
      
      return null;
    },
    routes: [
      GoRoute(
        path: '/login',
        builder: (context, state) => const LoginScreen(),
      ),
      GoRoute(
        path: '/home',
        builder: (context, state) => const Scaffold(
          body: Center(child: Text('Dashboard Placeholder. (US6)')),
        ),
      ),
    ],
  );
}
