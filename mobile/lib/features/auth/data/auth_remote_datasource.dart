import 'package:dio/dio.dart';
import 'package:google_sign_in/google_sign_in.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import 'package:klyra/core/network/dio_client.dart';
import 'package:klyra/features/auth/domain/auth_models.dart';

part 'auth_remote_datasource.g.dart';

@riverpod
AuthRemoteDataSource authRemoteDataSource(Ref ref) {
  final dio = ref.watch(dioClientProvider);
  // For Android, google_sign_in expects the Web OAuth client as serverClientId
  // to issue an ID token with the correct audience (`aud`).
  const webServerClientId = '782011204480-ri7aibqr5f922bqa0se6dpbtc5e5jvj8.apps.googleusercontent.com';
  final googleSignIn = GoogleSignIn(
    serverClientId: webServerClientId,
    scopes: ['email', 'profile'],
  );
  return AuthRemoteDataSource(dio, googleSignIn);
}

class AuthRemoteDataSource {
  final Dio _dio;
  final GoogleSignIn _googleSignIn;

  AuthRemoteDataSource(this._dio, this._googleSignIn);

  /// Triggers Google OAuth flow and exchanges the ID token with the Klyra backend.
  Future<AuthResult> signInWithGoogle() async {
    try {
      // 1. Trigger native Google Sign-In UI
      final GoogleSignInAccount? googleUser = await _googleSignIn.signIn();
      if (googleUser == null) {
        throw Exception("Sign in aborted by user");
      }

      // 2. Obtain auth details containing the ID token
      final GoogleSignInAuthentication googleAuth = await googleUser.authentication;
      final idToken = googleAuth.idToken;

      if (idToken == null) {
        throw Exception(
          "Failed to retrieve ID token from Google. "
          "Ensure google-services.json exists in mobile/android/app/ "
          "and SHA-1 fingerprint is registered in Google Cloud Console."
        );
      }

      // 3. Send ID token to Klyra backend to get our own JWTs
      final response = await _dio.post(
        '/auth/google',
        data: {
          'id_token': idToken,
        },
      );

      if (response.statusCode == 200 || response.statusCode == 201) {
        return AuthResult.fromJson(response.data);
      } else {
        throw Exception("Backend authentication failed: ${response.statusCode}");
      }
    } catch (e) {
      rethrow;
    }
  }

  Future<void> signOut() async {
    await _googleSignIn.signOut();
  }
}
