// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'material_remote_datasource.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(materialRemoteDataSource)
final materialRemoteDataSourceProvider = MaterialRemoteDataSourceProvider._();

final class MaterialRemoteDataSourceProvider
    extends
        $FunctionalProvider<
          MaterialRemoteDataSource,
          MaterialRemoteDataSource,
          MaterialRemoteDataSource
        >
    with $Provider<MaterialRemoteDataSource> {
  MaterialRemoteDataSourceProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'materialRemoteDataSourceProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$materialRemoteDataSourceHash();

  @$internal
  @override
  $ProviderElement<MaterialRemoteDataSource> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  MaterialRemoteDataSource create(Ref ref) {
    return materialRemoteDataSource(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(MaterialRemoteDataSource value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<MaterialRemoteDataSource>(value),
    );
  }
}

String _$materialRemoteDataSourceHash() =>
    r'830c0522e352db0e3234c4cdd82c29ecf3f22e3c';
