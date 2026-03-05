// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'material_models.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$Material {

 String get id;@JsonKey(name: 'topic_id') String get topicId;@JsonKey(name: 'format_type') MaterialFormatType get formatType;@JsonKey(name: 'storage_url') String get storageUrl;@JsonKey(name: 'extracted_text') String? get extractedText; MaterialStatus get status;@JsonKey(name: 'original_name') String get originalName;@JsonKey(name: 'size_bytes') int get sizeBytes;@JsonKey(name: 'created_at') DateTime get createdAt;@JsonKey(name: 'updated_at') DateTime get updatedAt;
/// Create a copy of Material
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$MaterialCopyWith<Material> get copyWith => _$MaterialCopyWithImpl<Material>(this as Material, _$identity);

  /// Serializes this Material to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is Material&&(identical(other.id, id) || other.id == id)&&(identical(other.topicId, topicId) || other.topicId == topicId)&&(identical(other.formatType, formatType) || other.formatType == formatType)&&(identical(other.storageUrl, storageUrl) || other.storageUrl == storageUrl)&&(identical(other.extractedText, extractedText) || other.extractedText == extractedText)&&(identical(other.status, status) || other.status == status)&&(identical(other.originalName, originalName) || other.originalName == originalName)&&(identical(other.sizeBytes, sizeBytes) || other.sizeBytes == sizeBytes)&&(identical(other.createdAt, createdAt) || other.createdAt == createdAt)&&(identical(other.updatedAt, updatedAt) || other.updatedAt == updatedAt));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,topicId,formatType,storageUrl,extractedText,status,originalName,sizeBytes,createdAt,updatedAt);

@override
String toString() {
  return 'Material(id: $id, topicId: $topicId, formatType: $formatType, storageUrl: $storageUrl, extractedText: $extractedText, status: $status, originalName: $originalName, sizeBytes: $sizeBytes, createdAt: $createdAt, updatedAt: $updatedAt)';
}


}

/// @nodoc
abstract mixin class $MaterialCopyWith<$Res>  {
  factory $MaterialCopyWith(Material value, $Res Function(Material) _then) = _$MaterialCopyWithImpl;
@useResult
$Res call({
 String id,@JsonKey(name: 'topic_id') String topicId,@JsonKey(name: 'format_type') MaterialFormatType formatType,@JsonKey(name: 'storage_url') String storageUrl,@JsonKey(name: 'extracted_text') String? extractedText, MaterialStatus status,@JsonKey(name: 'original_name') String originalName,@JsonKey(name: 'size_bytes') int sizeBytes,@JsonKey(name: 'created_at') DateTime createdAt,@JsonKey(name: 'updated_at') DateTime updatedAt
});




}
/// @nodoc
class _$MaterialCopyWithImpl<$Res>
    implements $MaterialCopyWith<$Res> {
  _$MaterialCopyWithImpl(this._self, this._then);

  final Material _self;
  final $Res Function(Material) _then;

/// Create a copy of Material
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? topicId = null,Object? formatType = null,Object? storageUrl = null,Object? extractedText = freezed,Object? status = null,Object? originalName = null,Object? sizeBytes = null,Object? createdAt = null,Object? updatedAt = null,}) {
  return _then(_self.copyWith(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,topicId: null == topicId ? _self.topicId : topicId // ignore: cast_nullable_to_non_nullable
as String,formatType: null == formatType ? _self.formatType : formatType // ignore: cast_nullable_to_non_nullable
as MaterialFormatType,storageUrl: null == storageUrl ? _self.storageUrl : storageUrl // ignore: cast_nullable_to_non_nullable
as String,extractedText: freezed == extractedText ? _self.extractedText : extractedText // ignore: cast_nullable_to_non_nullable
as String?,status: null == status ? _self.status : status // ignore: cast_nullable_to_non_nullable
as MaterialStatus,originalName: null == originalName ? _self.originalName : originalName // ignore: cast_nullable_to_non_nullable
as String,sizeBytes: null == sizeBytes ? _self.sizeBytes : sizeBytes // ignore: cast_nullable_to_non_nullable
as int,createdAt: null == createdAt ? _self.createdAt : createdAt // ignore: cast_nullable_to_non_nullable
as DateTime,updatedAt: null == updatedAt ? _self.updatedAt : updatedAt // ignore: cast_nullable_to_non_nullable
as DateTime,
  ));
}

}


/// Adds pattern-matching-related methods to [Material].
extension MaterialPatterns on Material {
/// A variant of `map` that fallback to returning `orElse`.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case _:
///     return orElse();
/// }
/// ```

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _Material value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _Material() when $default != null:
return $default(_that);case _:
  return orElse();

}
}
/// A `switch`-like method, using callbacks.
///
/// Callbacks receives the raw object, upcasted.
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case final Subclass2 value:
///     return ...;
/// }
/// ```

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _Material value)  $default,){
final _that = this;
switch (_that) {
case _Material():
return $default(_that);case _:
  throw StateError('Unexpected subclass');

}
}
/// A variant of `map` that fallback to returning `null`.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case _:
///     return null;
/// }
/// ```

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _Material value)?  $default,){
final _that = this;
switch (_that) {
case _Material() when $default != null:
return $default(_that);case _:
  return null;

}
}
/// A variant of `when` that fallback to an `orElse` callback.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case _:
///     return orElse();
/// }
/// ```

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String id, @JsonKey(name: 'topic_id')  String topicId, @JsonKey(name: 'format_type')  MaterialFormatType formatType, @JsonKey(name: 'storage_url')  String storageUrl, @JsonKey(name: 'extracted_text')  String? extractedText,  MaterialStatus status, @JsonKey(name: 'original_name')  String originalName, @JsonKey(name: 'size_bytes')  int sizeBytes, @JsonKey(name: 'created_at')  DateTime createdAt, @JsonKey(name: 'updated_at')  DateTime updatedAt)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _Material() when $default != null:
return $default(_that.id,_that.topicId,_that.formatType,_that.storageUrl,_that.extractedText,_that.status,_that.originalName,_that.sizeBytes,_that.createdAt,_that.updatedAt);case _:
  return orElse();

}
}
/// A `switch`-like method, using callbacks.
///
/// As opposed to `map`, this offers destructuring.
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case Subclass2(:final field2):
///     return ...;
/// }
/// ```

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String id, @JsonKey(name: 'topic_id')  String topicId, @JsonKey(name: 'format_type')  MaterialFormatType formatType, @JsonKey(name: 'storage_url')  String storageUrl, @JsonKey(name: 'extracted_text')  String? extractedText,  MaterialStatus status, @JsonKey(name: 'original_name')  String originalName, @JsonKey(name: 'size_bytes')  int sizeBytes, @JsonKey(name: 'created_at')  DateTime createdAt, @JsonKey(name: 'updated_at')  DateTime updatedAt)  $default,) {final _that = this;
switch (_that) {
case _Material():
return $default(_that.id,_that.topicId,_that.formatType,_that.storageUrl,_that.extractedText,_that.status,_that.originalName,_that.sizeBytes,_that.createdAt,_that.updatedAt);case _:
  throw StateError('Unexpected subclass');

}
}
/// A variant of `when` that fallback to returning `null`
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case _:
///     return null;
/// }
/// ```

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String id, @JsonKey(name: 'topic_id')  String topicId, @JsonKey(name: 'format_type')  MaterialFormatType formatType, @JsonKey(name: 'storage_url')  String storageUrl, @JsonKey(name: 'extracted_text')  String? extractedText,  MaterialStatus status, @JsonKey(name: 'original_name')  String originalName, @JsonKey(name: 'size_bytes')  int sizeBytes, @JsonKey(name: 'created_at')  DateTime createdAt, @JsonKey(name: 'updated_at')  DateTime updatedAt)?  $default,) {final _that = this;
switch (_that) {
case _Material() when $default != null:
return $default(_that.id,_that.topicId,_that.formatType,_that.storageUrl,_that.extractedText,_that.status,_that.originalName,_that.sizeBytes,_that.createdAt,_that.updatedAt);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _Material implements Material {
  const _Material({required this.id, @JsonKey(name: 'topic_id') required this.topicId, @JsonKey(name: 'format_type') required this.formatType, @JsonKey(name: 'storage_url') required this.storageUrl, @JsonKey(name: 'extracted_text') this.extractedText, required this.status, @JsonKey(name: 'original_name') required this.originalName, @JsonKey(name: 'size_bytes') required this.sizeBytes, @JsonKey(name: 'created_at') required this.createdAt, @JsonKey(name: 'updated_at') required this.updatedAt});
  factory _Material.fromJson(Map<String, dynamic> json) => _$MaterialFromJson(json);

@override final  String id;
@override@JsonKey(name: 'topic_id') final  String topicId;
@override@JsonKey(name: 'format_type') final  MaterialFormatType formatType;
@override@JsonKey(name: 'storage_url') final  String storageUrl;
@override@JsonKey(name: 'extracted_text') final  String? extractedText;
@override final  MaterialStatus status;
@override@JsonKey(name: 'original_name') final  String originalName;
@override@JsonKey(name: 'size_bytes') final  int sizeBytes;
@override@JsonKey(name: 'created_at') final  DateTime createdAt;
@override@JsonKey(name: 'updated_at') final  DateTime updatedAt;

/// Create a copy of Material
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$MaterialCopyWith<_Material> get copyWith => __$MaterialCopyWithImpl<_Material>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$MaterialToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _Material&&(identical(other.id, id) || other.id == id)&&(identical(other.topicId, topicId) || other.topicId == topicId)&&(identical(other.formatType, formatType) || other.formatType == formatType)&&(identical(other.storageUrl, storageUrl) || other.storageUrl == storageUrl)&&(identical(other.extractedText, extractedText) || other.extractedText == extractedText)&&(identical(other.status, status) || other.status == status)&&(identical(other.originalName, originalName) || other.originalName == originalName)&&(identical(other.sizeBytes, sizeBytes) || other.sizeBytes == sizeBytes)&&(identical(other.createdAt, createdAt) || other.createdAt == createdAt)&&(identical(other.updatedAt, updatedAt) || other.updatedAt == updatedAt));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,topicId,formatType,storageUrl,extractedText,status,originalName,sizeBytes,createdAt,updatedAt);

@override
String toString() {
  return 'Material(id: $id, topicId: $topicId, formatType: $formatType, storageUrl: $storageUrl, extractedText: $extractedText, status: $status, originalName: $originalName, sizeBytes: $sizeBytes, createdAt: $createdAt, updatedAt: $updatedAt)';
}


}

/// @nodoc
abstract mixin class _$MaterialCopyWith<$Res> implements $MaterialCopyWith<$Res> {
  factory _$MaterialCopyWith(_Material value, $Res Function(_Material) _then) = __$MaterialCopyWithImpl;
@override @useResult
$Res call({
 String id,@JsonKey(name: 'topic_id') String topicId,@JsonKey(name: 'format_type') MaterialFormatType formatType,@JsonKey(name: 'storage_url') String storageUrl,@JsonKey(name: 'extracted_text') String? extractedText, MaterialStatus status,@JsonKey(name: 'original_name') String originalName,@JsonKey(name: 'size_bytes') int sizeBytes,@JsonKey(name: 'created_at') DateTime createdAt,@JsonKey(name: 'updated_at') DateTime updatedAt
});




}
/// @nodoc
class __$MaterialCopyWithImpl<$Res>
    implements _$MaterialCopyWith<$Res> {
  __$MaterialCopyWithImpl(this._self, this._then);

  final _Material _self;
  final $Res Function(_Material) _then;

/// Create a copy of Material
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? topicId = null,Object? formatType = null,Object? storageUrl = null,Object? extractedText = freezed,Object? status = null,Object? originalName = null,Object? sizeBytes = null,Object? createdAt = null,Object? updatedAt = null,}) {
  return _then(_Material(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,topicId: null == topicId ? _self.topicId : topicId // ignore: cast_nullable_to_non_nullable
as String,formatType: null == formatType ? _self.formatType : formatType // ignore: cast_nullable_to_non_nullable
as MaterialFormatType,storageUrl: null == storageUrl ? _self.storageUrl : storageUrl // ignore: cast_nullable_to_non_nullable
as String,extractedText: freezed == extractedText ? _self.extractedText : extractedText // ignore: cast_nullable_to_non_nullable
as String?,status: null == status ? _self.status : status // ignore: cast_nullable_to_non_nullable
as MaterialStatus,originalName: null == originalName ? _self.originalName : originalName // ignore: cast_nullable_to_non_nullable
as String,sizeBytes: null == sizeBytes ? _self.sizeBytes : sizeBytes // ignore: cast_nullable_to_non_nullable
as int,createdAt: null == createdAt ? _self.createdAt : createdAt // ignore: cast_nullable_to_non_nullable
as DateTime,updatedAt: null == updatedAt ? _self.updatedAt : updatedAt // ignore: cast_nullable_to_non_nullable
as DateTime,
  ));
}


}

// dart format on
