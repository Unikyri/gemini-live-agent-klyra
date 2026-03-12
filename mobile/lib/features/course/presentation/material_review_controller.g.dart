// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'material_review_controller.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(MaterialReviewController)
final materialReviewControllerProvider = MaterialReviewControllerFamily._();

final class MaterialReviewControllerProvider
    extends
        $AsyncNotifierProvider<MaterialReviewController, MaterialReviewState> {
  MaterialReviewControllerProvider._({
    required MaterialReviewControllerFamily super.from,
    required ({String courseId, String topicId, String materialId})
    super.argument,
  }) : super(
         retry: null,
         name: r'materialReviewControllerProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$materialReviewControllerHash();

  @override
  String toString() {
    return r'materialReviewControllerProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  MaterialReviewController create() => MaterialReviewController();

  @override
  bool operator ==(Object other) {
    return other is MaterialReviewControllerProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$materialReviewControllerHash() =>
    r'c90568347e5dfae7748851e27976e5e0e69e6137';

final class MaterialReviewControllerFamily extends $Family
    with
        $ClassFamilyOverride<
          MaterialReviewController,
          AsyncValue<MaterialReviewState>,
          MaterialReviewState,
          FutureOr<MaterialReviewState>,
          ({String courseId, String topicId, String materialId})
        > {
  MaterialReviewControllerFamily._()
    : super(
        retry: null,
        name: r'materialReviewControllerProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  MaterialReviewControllerProvider call({
    required String courseId,
    required String topicId,
    required String materialId,
  }) => MaterialReviewControllerProvider._(
    argument: (courseId: courseId, topicId: topicId, materialId: materialId),
    from: this,
  );

  @override
  String toString() => r'materialReviewControllerProvider';
}

abstract class _$MaterialReviewController
    extends $AsyncNotifier<MaterialReviewState> {
  late final _$args =
      ref.$arg as ({String courseId, String topicId, String materialId});
  String get courseId => _$args.courseId;
  String get topicId => _$args.topicId;
  String get materialId => _$args.materialId;

  FutureOr<MaterialReviewState> build({
    required String courseId,
    required String topicId,
    required String materialId,
  });
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref as $Ref<AsyncValue<MaterialReviewState>, MaterialReviewState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<MaterialReviewState>, MaterialReviewState>,
              AsyncValue<MaterialReviewState>,
              Object?,
              Object?
            >;
    element.handleCreate(
      ref,
      () => build(
        courseId: _$args.courseId,
        topicId: _$args.topicId,
        materialId: _$args.materialId,
      ),
    );
  }
}
