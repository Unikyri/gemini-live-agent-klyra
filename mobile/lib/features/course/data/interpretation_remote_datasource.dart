import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:klyra/core/network/dio_client.dart';
import 'package:klyra/features/course/domain/interpretation_models.dart';

final interpretationRemoteDataSourceProvider =
    Provider<InterpretationRemoteDataSource>((ref) {
  final dio = ref.watch(dioClientProvider);
  return InterpretationRemoteDataSource(dio);
});

class InterpretationRemoteDataSource {
  final Dio _dio;

  InterpretationRemoteDataSource(this._dio);

  Future<InterpretationResult> getInterpretation({
    required String courseId,
    required String topicId,
    required String materialId,
  }) async {
    final resp = await _dio.get(
      '/courses/$courseId/topics/$topicId/materials/$materialId/interpretation',
    );
    if (resp.statusCode == 200) {
      return InterpretationResult.fromJson(resp.data as Map<String, dynamic>);
    }
    throw Exception('Failed to load interpretation');
  }

  Future<List<MaterialCorrection>> getCorrections({
    required String courseId,
    required String topicId,
    required String materialId,
  }) async {
    final resp = await _dio.get(
      '/courses/$courseId/topics/$topicId/materials/$materialId/corrections',
    );
    if (resp.statusCode == 200) {
      final data = (resp.data['corrections'] as List?) ?? const [];
      return data
          .whereType<Map<String, dynamic>>()
          .map(MaterialCorrection.fromJson)
          .toList();
    }
    throw Exception('Failed to load corrections');
  }

  Future<MaterialCorrection> submitCorrection({
    required String courseId,
    required String topicId,
    required String materialId,
    required int blockIndex,
    required String originalText,
    required String correctedText,
  }) async {
    final resp = await _dio.post(
      '/courses/$courseId/topics/$topicId/materials/$materialId/corrections',
      data: {
        'block_index': blockIndex,
        'original_text': originalText,
        'corrected_text': correctedText,
      },
    );
    if (resp.statusCode == 201 || resp.statusCode == 200) {
      return MaterialCorrection.fromJson(resp.data as Map<String, dynamic>);
    }
    throw Exception('Failed to submit correction');
  }

  Future<void> deleteCorrection({
    required String courseId,
    required String topicId,
    required String materialId,
    required String correctionId,
  }) async {
    final resp = await _dio.delete(
      '/courses/$courseId/topics/$topicId/materials/$materialId/corrections/$correctionId',
    );
    if (resp.statusCode == 204) return;
    throw Exception('Failed to delete correction');
  }
}

