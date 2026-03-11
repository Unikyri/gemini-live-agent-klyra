import 'package:dio/dio.dart';
import 'package:file_picker/file_picker.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:klyra/core/network/dio_client.dart';
import 'package:klyra/features/course/domain/material_models.dart';

part 'material_remote_datasource.g.dart';

@riverpod
MaterialRemoteDataSource materialRemoteDataSource(
    Ref ref) {
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

  /// Uploads a picked file as multipart form data to the materials endpoint.
  /// Supports both web (bytes) and native (path) platform payloads.
  Future<Material> uploadMaterial(
      String courseId, String topicId, PlatformFile file) async {
    MultipartFile multipart;

    if (file.bytes != null) {
      multipart = MultipartFile.fromBytes(
        file.bytes!,
        filename: file.name,
      );
    } else if (file.path != null) {
      multipart = await MultipartFile.fromFile(
        file.path!,
        filename: file.name,
      );
    } else {
      throw Exception('Picked file has no bytes or path');
    }

    final formData = FormData.fromMap({
      'file': multipart,
    });
    final response = await _dio.post(
      '/courses/$courseId/topics/$topicId/materials',
      data: formData,
    );
    if (response.statusCode == 201) {
      return Material.fromJson(response.data);
    }
    throw Exception('Failed to upload material');
  }
}
