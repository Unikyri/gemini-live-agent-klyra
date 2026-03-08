import 'package:riverpod_annotation/riverpod_annotation.dart';

import 'package:klyra/features/auth/data/auth_repository.dart';
import 'package:klyra/features/auth/domain/auth_models.dart';

part 'auth_controller.g.dart';

@Riverpod(keepAlive: true)
class AuthController extends _$AuthController {
  @override
  FutureOr<User?> build() async {
    // Attempt to restore the session from secure storage on app start.
    // For Sprint 2, we only check if a token exists.
    // Sprint 3 TODO: call GET /me to restore full User object and validate against server.
    final repo = ref.watch(authRepositoryProvider);
    final isLoggedIn = await repo.isLoggedIn();
    if (isLoggedIn) {
      // Return a sentinel User to signal that the session is valid.
      // This is enough to redirect to the dashboard via GoRouter guard.
      // The full User profile will be loaded by the Dashboard screen.
      return const User(id: 'cached', email: '', name: 'Cached Session');
    }
    return null;
  }

  Future<void> signInWithGoogle() async {
    state = const AsyncValue.loading();
    state = await AsyncValue.guard(() async {
      final repo = ref.read(authRepositoryProvider);
      return await repo.signIn();
    });
  }

  /// Sign in as a guest user (development/testing only)
  Future<void> signInAsGuest({String email = 'guest@example.com', String name = 'Guest User'}) async {
    state = const AsyncValue.loading();
    state = await AsyncValue.guard(() async {
      final repo = ref.read(authRepositoryProvider);
      return await repo.signInAsGuest(email: email, name: name);
    });
  }

  Future<void> signOut() async {
    state = const AsyncValue.loading();
    state = await AsyncValue.guard(() async {
      final repo = ref.read(authRepositoryProvider);
      await repo.signOut();
      return null;
    });
  }
}
