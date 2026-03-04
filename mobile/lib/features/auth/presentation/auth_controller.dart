import 'package:riverpod_annotation/riverpod_annotation.dart';

import 'package:klyra/features/auth/data/auth_repository.dart';
import 'package:klyra/features/auth/domain/auth_models.dart';

part 'auth_controller.g.dart';

@Riverpod(keepAlive: true)
class AuthController extends _$AuthController {
  @override
  FutureOr<User?> build() async {
    // Check if user is already logged in on app start
    final repo = ref.watch(authRepositoryProvider);
    final isLoggedIn = await repo.isLoggedIn();
    if (isLoggedIn) {
      // For MVP, we don't have a /me endpoint yet, so we just return a dummy user 
      // or we'd ideally fetch the profile. Let's return null to force login for now
      // unless we actually persist the user object too.
      // Better strategy: persist User object or fetch /me.
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

  Future<void> signOut() async {
    state = const AsyncValue.loading();
    state = await AsyncValue.guard(() async {
      final repo = ref.read(authRepositoryProvider);
      await repo.signOut();
      return null;
    });
  }
}
