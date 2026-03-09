import 'package:flutter/foundation.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import 'package:klyra/core/storage/secure_storage.dart';
import 'package:klyra/features/auth/data/auth_remote_datasource.dart';
import 'package:klyra/features/auth/domain/auth_models.dart';

part 'auth_repository.g.dart';

@riverpod
AuthRepository authRepository(Ref ref) {
  final remoteDataSource = ref.watch(authRemoteDataSourceProvider);
  final secureStorage = ref.watch(secureStorageProvider);
  return AuthRepository(remoteDataSource, secureStorage);
}

class AuthRepository {
  final AuthRemoteDataSource _remoteDataSource;
  final FlutterSecureStorage _secureStorage;

  AuthRepository(this._remoteDataSource, this._secureStorage);

  Future<User> signIn() async {
    // 1. Authenticate with Google and Backend
    final AuthResult result = await _remoteDataSource.signInWithGoogle();

    // 2. Securely store the JWT tokens (per US5 security requirements)
    await _secureStorage.write(key: StorageKeys.accessToken, value: result.accessToken);
    await _secureStorage.write(key: StorageKeys.refreshToken, value: result.refreshToken);

    // 3. Return the user info
    return result.user;
  }

  /// Guest login (development/testing only) - no backend validation
  Future<User> signInAsGuest({String email = 'guest@example.com', String name = 'Guest User'}) async {
    debugPrint('[Auth] Starting guest login for $email');
    final AuthResult result = await _remoteDataSource.signInAsGuest(email: email, name: name);
    debugPrint('[Auth] Guest login successful - access_token: ${result.accessToken.substring(0, 20)}...');
    await _secureStorage.write(key: StorageKeys.accessToken, value: result.accessToken);
    await _secureStorage.write(key: StorageKeys.refreshToken, value: result.refreshToken);
    debugPrint('[Auth] ✓ Tokens saved to secure storage');
    return result.user;
  }

  Future<void> signOut() async {
    await _remoteDataSource.signOut();
    await _secureStorage.delete(key: StorageKeys.accessToken);
    await _secureStorage.delete(key: StorageKeys.refreshToken);
  }

  /// Checks if a valid token exists locally. Used for auto-login.
  Future<bool> isLoggedIn() async {
    final token = await _secureStorage.read(key: StorageKeys.accessToken);
    return token != null && token.isNotEmpty;
  }
}
