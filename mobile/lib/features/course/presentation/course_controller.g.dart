// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'course_controller.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(CourseController)
final courseControllerProvider = CourseControllerProvider._();

final class CourseControllerProvider
    extends $AsyncNotifierProvider<CourseController, List<Course>> {
  CourseControllerProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'courseControllerProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$courseControllerHash();

  @$internal
  @override
  CourseController create() => CourseController();
}

String _$courseControllerHash() => r'2cab8419e46bfe90db371703338e7093958de3c8';

abstract class _$CourseController extends $AsyncNotifier<List<Course>> {
  FutureOr<List<Course>> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<List<Course>>, List<Course>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<List<Course>>, List<Course>>,
              AsyncValue<List<Course>>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
