import 'dart:io';
import 'package:dio/dio.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:klyra/core/network/dio_client.dart';
import 'package:klyra/features/course/domain/material_models.dart';

part 'material_remote_datasource.g.dart';

@riverpod
MaterialRemoteDataSource materialRemoteDataSource(
    MaterialRemoteDataSourceRef ref) {
  final dio = ref.watch(dioClientProvider);
  return MaterialRemoteDataSource(dio);
}

class MaterialRemoteDataSource {
  final Dio _dio;

  MaterialRemoteDataSource(this._dio);

  Future<List<Material>> getMaterials(
      String courseId, String topicId) async {
    final response = await _dio.get(
      '/courses/$courseId/topics/$topicId/materials',
    );
    if (response.statusCode == 200) {
      final List<dynamic> data = response.data['materials'] ?? [];
      return data.map((json) => Material.fromJson(json)).toList();
    }
    throw Exception('Failed to load materials');
  }

  /// Uploads a file as multipart form data to the materials endpoint.
  Future<Material> uploadMaterial(
      String courseId, String topicId, File file) async {
    final formData = FormData.fromMap({
      'file': await MultipartFile.fromFile(
        file.path,
        filename: file.path.split(Platform.pathSeparator).last,
      ),
    });
    final response = await _dio.post(
      '/courses/$courseId/topics/$topicId/materials',
      data: formData,
      options: Options(contentType: 'multipart/form-data'),
    );
    if (response.statusCode == 201) {
      return Material.fromJson(response.data);
    }
    throw Exception('Failed to upload material');
  }
}
