// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'material_controller.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(MaterialController)
final materialControllerProvider = MaterialControllerFamily._();

final class MaterialControllerProvider
    extends $AsyncNotifierProvider<MaterialController, List<Material>> {
  MaterialControllerProvider._({
    required MaterialControllerFamily super.from,
    required ({String courseId, String topicId}) super.argument,
  }) : super(
         retry: null,
         name: r'materialControllerProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$materialControllerHash();

  @override
  String toString() {
    return r'materialControllerProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  MaterialController create() => MaterialController();

  @override
  bool operator ==(Object other) {
    return other is MaterialControllerProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$materialControllerHash() =>
    r'3cce5094add608e8e48763dcb055697def0eabef';

final class MaterialControllerFamily extends $Family
    with
        $ClassFamilyOverride<
          MaterialController,
          AsyncValue<List<Material>>,
          List<Material>,
          FutureOr<List<Material>>,
          ({String courseId, String topicId})
        > {
  MaterialControllerFamily._()
    : super(
        retry: null,
        name: r'materialControllerProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  MaterialControllerProvider call({
    required String courseId,
    required String topicId,
  }) => MaterialControllerProvider._(
    argument: (courseId: courseId, topicId: topicId),
    from: this,
  );

  @override
  String toString() => r'materialControllerProvider';
}

abstract class _$MaterialController extends $AsyncNotifier<List<Material>> {
  late final _$args = ref.$arg as ({String courseId, String topicId});
  String get courseId => _$args.courseId;
  String get topicId => _$args.topicId;

  FutureOr<List<Material>> build({
    required String courseId,
    required String topicId,
  });
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<List<Material>>, List<Material>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<List<Material>>, List<Material>>,
              AsyncValue<List<Material>>,
              Object?,
              Object?
            >;
    element.handleCreate(
      ref,
      () => build(courseId: _$args.courseId, topicId: _$args.topicId),
    );
  }
}
