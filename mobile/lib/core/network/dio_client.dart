import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:klyra/core/config/env.dart';
import 'package:klyra/core/storage/secure_storage.dart';

part 'dio_client.g.dart';

@Riverpod(keepAlive: true)
Dio dioClient(Ref ref) {
  final dio = Dio(BaseOptions(
    baseUrl: EnvInfo.backendBaseUrl,
    connectTimeout: const Duration(seconds: 15),
    receiveTimeout: const Duration(seconds: 15),
    contentType: 'application/json',
  ));

  // Add logging interceptor in debug mode
  if (!kIsWeb) {
    dio.interceptors.add(
      LoggingInterceptor(),
    );
  }

  final secureStorage = ref.watch(secureStorageProvider);

  dio.interceptors.add(
    InterceptorsWrapper(
      onRequest: (options, handler) async {
        // Exclude auth routes from requiring a token
        if (!options.path.contains('/auth/google')) {
          final token = await secureStorage.read(key: StorageKeys.accessToken);
          debugPrint('[Auth] Token read from storage: ${token != null ? "✓ ${token.substring(0, 20)}..." : "✗ NULL"}');
          if (token != null) {
            options.headers['Authorization'] = 'Bearer $token';
            debugPrint('[Auth] Authorization header added');
          } else {
            debugPrint('[Auth] ⚠ No token available - request will be unauthorized');
          }
        }
        return handler.next(options);
      },
      onError: (DioException e, handler) async {
        // Log network errors with full context
        debugPrint('[DioError] ${e.requestOptions.method} ${e.requestOptions.path}');
        debugPrint('[DioError] Status: ${e.response?.statusCode}');
        debugPrint('[DioError] Type: ${e.type}');
        debugPrint('[DioError] Message: ${e.message}');
        debugPrint('[DioError] Response: ${e.response?.toString()}');
        
        // Handle 401 Unauthorized globally (e.g. trigger logout or token refresh)
        // For Sprint 2 MVP, we just pass the error. Refresh logic planned for later.
        if (e.response?.statusCode == 401) {
          // TODO: Implement refresh token logic if 401 occurs
        }
        return handler.next(e);
      },
    ),
  );

  return dio;
}

/// Custom logging interceptor for Dio
class LoggingInterceptor extends Interceptor {
  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    debugPrint('► [${options.method}] ${options.uri}');
    if (options.data != null) {
      debugPrint('   Body: ${options.data}');
    }
    debugPrint('   Headers: ${options.headers}');
    super.onRequest(options, handler);
  }

  @override
  void onResponse(Response response, ResponseInterceptorHandler handler) {
    debugPrint('◄ [${response.statusCode}] ${response.requestOptions.path}');
    debugPrint('   Response: ${response.data}');
    super.onResponse(response, handler);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    debugPrint('✗ [${err.type}] ${err.requestOptions.method} ${err.requestOptions.path}');
    debugPrint('   Error: ${err.message}');
    super.onError(err, handler);
  }
}
