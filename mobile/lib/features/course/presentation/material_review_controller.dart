import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:klyra/features/course/data/interpretation_remote_datasource.dart';
import 'package:klyra/features/course/domain/interpretation_models.dart';

part 'material_review_controller.g.dart';

class MaterialReviewState {
  final InterpretationResult interpretation;
  final List<MaterialCorrection> corrections;

  const MaterialReviewState({
    required this.interpretation,
    required this.corrections,
  });
}

@riverpod
class MaterialReviewController extends _$MaterialReviewController {
  @override
  FutureOr<MaterialReviewState> build({
    required String courseId,
    required String topicId,
    required String materialId,
  }) async {
    final ds = ref.read(interpretationRemoteDataSourceProvider);
    final interp = await ds.getInterpretation(
      courseId: courseId,
      topicId: topicId,
      materialId: materialId,
    );
    final corr = await ds.getCorrections(
      courseId: courseId,
      topicId: topicId,
      materialId: materialId,
    );
    return MaterialReviewState(interpretation: interp, corrections: corr);
  }

  Future<void> submitCorrection({
    required int blockIndex,
    required String originalText,
    required String correctedText,
  }) async {
    final ds = ref.read(interpretationRemoteDataSourceProvider);
    final created = await ds.submitCorrection(
      courseId: courseId,
      topicId: topicId,
      materialId: materialId,
      blockIndex: blockIndex,
      originalText: originalText,
      correctedText: correctedText,
    );
    final current = state.value;
    if (current == null) return;
    final updated = List<MaterialCorrection>.from(current.corrections);
    final idx = updated.indexWhere((c) => c.blockIndex == created.blockIndex);
    if (idx >= 0) {
      updated[idx] = created;
    } else {
      updated.add(created);
      updated.sort((a, b) => a.blockIndex.compareTo(b.blockIndex));
    }
    state = AsyncData(
      MaterialReviewState(
        interpretation: current.interpretation,
        corrections: updated,
      ),
    );
  }
}

