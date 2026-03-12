import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:google_sign_in/google_sign_in.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import 'package:klyra/core/config/env.dart';
import 'package:klyra/core/network/dio_client.dart';
import 'package:klyra/features/auth/domain/auth_models.dart';

part 'auth_remote_datasource.g.dart';

@riverpod
AuthRemoteDataSource authRemoteDataSource(Ref ref) {
  final dio = ref.watch(dioClientProvider);
  
  // GoogleSignIn is only used for mobile/desktop, not web (use-case: guest login + OAuth on mobile)
  GoogleSignIn? googleSignIn;
  if (!kIsWeb) {
    // For Android, google_sign_in expects the Web OAuth client as serverClientId
    // to issue an ID token with the correct audience (`aud`).
    googleSignIn = GoogleSignIn(
      serverClientId: EnvInfo.googleWebClientId,
      scopes: ['email', 'profile'],
    );
  }
  
  return AuthRemoteDataSource(dio, googleSignIn);
}

class AuthRemoteDataSource {
  final Dio _dio;
  final GoogleSignIn? _googleSignIn; // nullable for web where GoogleSignIn is not used

  AuthRemoteDataSource(this._dio, this._googleSignIn);

  /// Generic method to sign in with any provider using the unified endpoint.
  /// Implements the Strategy Pattern at the frontend: marshals provider-specific
  /// credentials into the unified /auth/login request format.
  /// 
  /// Falls back to legacy endpoints if unified endpoint is unavailable.
  Future<AuthResult> _signInWithProvider(
    String provider,
    Map<String, dynamic> credentials,
  ) async {
    try {
      // Try unified endpoint first (Phase 4+)
      final response = await _dio.post(
        '/auth/login',
        data: {
          'provider': provider,
          ...credentials,
        },
      );

      if (response.statusCode == 200 || response.statusCode == 201) {
        return AuthResult.fromJson(response.data);
      } else {
        throw Exception("Unified endpoint failed: ${response.statusCode}");
      }
    } on DioException catch (e) {
      // Graceful fallback to legacy endpoints if unified endpoint is unavailable
      if (e.response?.statusCode == 404) {
        print('[Auth] Unified endpoint /auth/login not found, falling back to legacy route');
        return await _signInWithLegacyEndpoint(provider, credentials);
      }
      rethrow;
    }
  }

  /// Fallback method for legacy endpoints (Phase 3 and earlier)
  /// Supports: /auth/google, /auth/google-mock
  Future<AuthResult> _signInWithLegacyEndpoint(
    String provider,
    Map<String, dynamic> credentials,
  ) async {
    String endpoint;
    Map<String, dynamic> payload;

    switch (provider) {
      case 'google':
        endpoint = '/auth/google';
        payload = {'id_token': credentials['id_token']};
        break;
      case 'guest':
        endpoint = '/auth/google-mock';
        payload = {
          'email': credentials['email'],
          'name': credentials['name'],
        };
        break;
      default:
        throw Exception("Unsupported provider: $provider");
    }

    final response = await _dio.post(endpoint, data: payload);
    if (response.statusCode == 200 || response.statusCode == 201) {
      return AuthResult.fromJson(response.data);
    } else {
      throw Exception("Legacy endpoint failed: ${response.statusCode}");
    }
  }

  /// Triggers Google OAuth flow and exchanges the ID token with the Klyra backend.
  /// Not available on web (returns unsupported error).
  Future<AuthResult> signInWithGoogle() async {
    if (_googleSignIn == null) {
      throw Exception(
        "Google Sign-In is not available on this platform. "
        "For web, use 'Continue as Guest' or configure OAuth properly."
      );
    }
    
    try {
      // 1. Trigger native Google Sign-In UI
      final GoogleSignInAccount? googleUser = await _googleSignIn!.signIn();
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

      // 3. Send ID token to Klyra backend via unified endpoint
      return await _signInWithProvider('google', {
        'id_token': idToken,
      });
    } catch (e) {
      rethrow;
    }
  }

  /// Development-only: Sign in as a guest/test user (localhost mock only)
  /// Uses the unified endpoint (/auth/login with provider=guest) on Phase 4+
  /// Falls back to legacy /auth/google-mock if needed.
  Future<AuthResult> signInAsGuest({
    String email = 'guest@example.com',
    String name = 'Guest User',
  }) async {
    try {
      return await _signInWithProvider('guest', {
        'email': email,
        'name': name,
      });
    } catch (e) {
      rethrow;
    }
  }

  Future<void> signOut() async {
    if (_googleSignIn != null) {
      await _googleSignIn!.signOut();
    }
    // Guest users don't have a Google session to sign out of
  }
}
