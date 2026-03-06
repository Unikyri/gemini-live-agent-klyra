import 'package:dio/dio.dart';
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

  final secureStorage = ref.watch(secureStorageProvider);

  dio.interceptors.add(
    InterceptorsWrapper(
      onRequest: (options, handler) async {
        // Exclude auth routes from requiring a token
        if (!options.path.contains('/auth/google')) {
          final token = await secureStorage.read(key: StorageKeys.accessToken);
          if (token != null) {
            options.headers['Authorization'] = 'Bearer $token';
          }
        }
        return handler.next(options);
      },
      onError: (DioException e, handler) async {
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
