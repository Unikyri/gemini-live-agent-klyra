import 'package:riverpod_annotation/riverpod_annotation.dart';

import 'package:klyra/features/auth/data/auth_repository.dart';
import 'package:klyra/features/auth/domain/auth_models.dart';

part 'auth_controller.g.dart';

@Riverpod(keepAlive: true)
class AuthController extends _$AuthController {
  @override
  FutureOr<User?> build() async {
    // Attempt to restore the session from secure storage on app start.
    // For Sprint 2/3: Auto-login as guest to get fresh tokens.
    // TODO Sprint 4: Implement proper token refresh or validate with GET /me
    final repo = ref.watch(authRepositoryProvider);
    final isLoggedIn = await repo.isLoggedIn();
    if (isLoggedIn) {
      // Auto-login as guest to get fresh tokens (tokens may have expired)
      try {
        return await repo.signInAsGuest();
      } catch (e) {
        // If auto-login fails, clear session and return null (force manual login)
        await repo.signOut();
        return null;
      }
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
