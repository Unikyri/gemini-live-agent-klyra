import 'dart:io';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:klyra/features/course/data/material_remote_datasource.dart';
import 'package:klyra/features/course/domain/material_models.dart';

part 'material_controller.g.dart';

@riverpod
class MaterialController extends _$MaterialController {
  @override
  FutureOr<List<Material>> build(
      {required String courseId, required String topicId}) async {
    return _fetchMaterials();
  }

  Future<List<Material>> _fetchMaterials() {
    final ds = ref.read(materialRemoteDataSourceProvider);
    return ds.getMaterials(courseId, topicId);
  }

  Future<void> uploadFile(File file) async {
    // Optimistically stay in loading state while uploading
    state = const AsyncValue.loading();
    state = await AsyncValue.guard(() async {
      final ds = ref.read(materialRemoteDataSourceProvider);
      await ds.uploadMaterial(courseId, topicId, file);
      // Always re-fetch after upload so we get the server-assigned ID and status
      return _fetchMaterials();
    });
  }
}
