// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'course_models.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$Course {

 String get id;@JsonKey(name: 'user_id') String get userId; String get name;@JsonKey(name: 'education_level') String get educationLevel;@JsonKey(name: 'avatar_model_url') String? get avatarModelUrl;@JsonKey(name: 'avatar_status') String get avatarStatus;// pending, generating, ready, failed
@JsonKey(name: 'created_at') DateTime get createdAt;@JsonKey(name: 'updated_at') DateTime get updatedAt;
/// Create a copy of Course
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$CourseCopyWith<Course> get copyWith => _$CourseCopyWithImpl<Course>(this as Course, _$identity);

  /// Serializes this Course to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is Course&&(identical(other.id, id) || other.id == id)&&(identical(other.userId, userId) || other.userId == userId)&&(identical(other.name, name) || other.name == name)&&(identical(other.educationLevel, educationLevel) || other.educationLevel == educationLevel)&&(identical(other.avatarModelUrl, avatarModelUrl) || other.avatarModelUrl == avatarModelUrl)&&(identical(other.avatarStatus, avatarStatus) || other.avatarStatus == avatarStatus)&&(identical(other.createdAt, createdAt) || other.createdAt == createdAt)&&(identical(other.updatedAt, updatedAt) || other.updatedAt == updatedAt));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,userId,name,educationLevel,avatarModelUrl,avatarStatus,createdAt,updatedAt);

@override
String toString() {
  return 'Course(id: $id, userId: $userId, name: $name, educationLevel: $educationLevel, avatarModelUrl: $avatarModelUrl, avatarStatus: $avatarStatus, createdAt: $createdAt, updatedAt: $updatedAt)';
}


}

/// @nodoc
abstract mixin class $CourseCopyWith<$Res>  {
  factory $CourseCopyWith(Course value, $Res Function(Course) _then) = _$CourseCopyWithImpl;
@useResult
$Res call({
 String id,@JsonKey(name: 'user_id') String userId, String name,@JsonKey(name: 'education_level') String educationLevel,@JsonKey(name: 'avatar_model_url') String? avatarModelUrl,@JsonKey(name: 'avatar_status') String avatarStatus,@JsonKey(name: 'created_at') DateTime createdAt,@JsonKey(name: 'updated_at') DateTime updatedAt
});




}
/// @nodoc
class _$CourseCopyWithImpl<$Res>
    implements $CourseCopyWith<$Res> {
  _$CourseCopyWithImpl(this._self, this._then);

  final Course _self;
  final $Res Function(Course) _then;

/// Create a copy of Course
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? userId = null,Object? name = null,Object? educationLevel = null,Object? avatarModelUrl = freezed,Object? avatarStatus = null,Object? createdAt = null,Object? updatedAt = null,}) {
  return _then(_self.copyWith(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,userId: null == userId ? _self.userId : userId // ignore: cast_nullable_to_non_nullable
as String,name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,educationLevel: null == educationLevel ? _self.educationLevel : educationLevel // ignore: cast_nullable_to_non_nullable
as String,avatarModelUrl: freezed == avatarModelUrl ? _self.avatarModelUrl : avatarModelUrl // ignore: cast_nullable_to_non_nullable
as String?,avatarStatus: null == avatarStatus ? _self.avatarStatus : avatarStatus // ignore: cast_nullable_to_non_nullable
as String,createdAt: null == createdAt ? _self.createdAt : createdAt // ignore: cast_nullable_to_non_nullable
as DateTime,updatedAt: null == updatedAt ? _self.updatedAt : updatedAt // ignore: cast_nullable_to_non_nullable
as DateTime,
  ));
}

}


/// Adds pattern-matching-related methods to [Course].
extension CoursePatterns on Course {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _Course value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _Course() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _Course value)  $default,){
final _that = this;
switch (_that) {
case _Course():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _Course value)?  $default,){
final _that = this;
switch (_that) {
case _Course() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String id, @JsonKey(name: 'user_id')  String userId,  String name, @JsonKey(name: 'education_level')  String educationLevel, @JsonKey(name: 'avatar_model_url')  String? avatarModelUrl, @JsonKey(name: 'avatar_status')  String avatarStatus, @JsonKey(name: 'created_at')  DateTime createdAt, @JsonKey(name: 'updated_at')  DateTime updatedAt)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _Course() when $default != null:
return $default(_that.id,_that.userId,_that.name,_that.educationLevel,_that.avatarModelUrl,_that.avatarStatus,_that.createdAt,_that.updatedAt);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String id, @JsonKey(name: 'user_id')  String userId,  String name, @JsonKey(name: 'education_level')  String educationLevel, @JsonKey(name: 'avatar_model_url')  String? avatarModelUrl, @JsonKey(name: 'avatar_status')  String avatarStatus, @JsonKey(name: 'created_at')  DateTime createdAt, @JsonKey(name: 'updated_at')  DateTime updatedAt)  $default,) {final _that = this;
switch (_that) {
case _Course():
return $default(_that.id,_that.userId,_that.name,_that.educationLevel,_that.avatarModelUrl,_that.avatarStatus,_that.createdAt,_that.updatedAt);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String id, @JsonKey(name: 'user_id')  String userId,  String name, @JsonKey(name: 'education_level')  String educationLevel, @JsonKey(name: 'avatar_model_url')  String? avatarModelUrl, @JsonKey(name: 'avatar_status')  String avatarStatus, @JsonKey(name: 'created_at')  DateTime createdAt, @JsonKey(name: 'updated_at')  DateTime updatedAt)?  $default,) {final _that = this;
switch (_that) {
case _Course() when $default != null:
return $default(_that.id,_that.userId,_that.name,_that.educationLevel,_that.avatarModelUrl,_that.avatarStatus,_that.createdAt,_that.updatedAt);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _Course implements Course {
  const _Course({required this.id, @JsonKey(name: 'user_id') required this.userId, required this.name, @JsonKey(name: 'education_level') required this.educationLevel, @JsonKey(name: 'avatar_model_url') this.avatarModelUrl, @JsonKey(name: 'avatar_status') required this.avatarStatus, @JsonKey(name: 'created_at') required this.createdAt, @JsonKey(name: 'updated_at') required this.updatedAt});
  factory _Course.fromJson(Map<String, dynamic> json) => _$CourseFromJson(json);

@override final  String id;
@override@JsonKey(name: 'user_id') final  String userId;
@override final  String name;
@override@JsonKey(name: 'education_level') final  String educationLevel;
@override@JsonKey(name: 'avatar_model_url') final  String? avatarModelUrl;
@override@JsonKey(name: 'avatar_status') final  String avatarStatus;
// pending, generating, ready, failed
@override@JsonKey(name: 'created_at') final  DateTime createdAt;
@override@JsonKey(name: 'updated_at') final  DateTime updatedAt;

/// Create a copy of Course
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$CourseCopyWith<_Course> get copyWith => __$CourseCopyWithImpl<_Course>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$CourseToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _Course&&(identical(other.id, id) || other.id == id)&&(identical(other.userId, userId) || other.userId == userId)&&(identical(other.name, name) || other.name == name)&&(identical(other.educationLevel, educationLevel) || other.educationLevel == educationLevel)&&(identical(other.avatarModelUrl, avatarModelUrl) || other.avatarModelUrl == avatarModelUrl)&&(identical(other.avatarStatus, avatarStatus) || other.avatarStatus == avatarStatus)&&(identical(other.createdAt, createdAt) || other.createdAt == createdAt)&&(identical(other.updatedAt, updatedAt) || other.updatedAt == updatedAt));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,id,userId,name,educationLevel,avatarModelUrl,avatarStatus,createdAt,updatedAt);

@override
String toString() {
  return 'Course(id: $id, userId: $userId, name: $name, educationLevel: $educationLevel, avatarModelUrl: $avatarModelUrl, avatarStatus: $avatarStatus, createdAt: $createdAt, updatedAt: $updatedAt)';
}


}

/// @nodoc
abstract mixin class _$CourseCopyWith<$Res> implements $CourseCopyWith<$Res> {
  factory _$CourseCopyWith(_Course value, $Res Function(_Course) _then) = __$CourseCopyWithImpl;
@override @useResult
$Res call({
 String id,@JsonKey(name: 'user_id') String userId, String name,@JsonKey(name: 'education_level') String educationLevel,@JsonKey(name: 'avatar_model_url') String? avatarModelUrl,@JsonKey(name: 'avatar_status') String avatarStatus,@JsonKey(name: 'created_at') DateTime createdAt,@JsonKey(name: 'updated_at') DateTime updatedAt
});




}
/// @nodoc
class __$CourseCopyWithImpl<$Res>
    implements _$CourseCopyWith<$Res> {
  __$CourseCopyWithImpl(this._self, this._then);

  final _Course _self;
  final $Res Function(_Course) _then;

/// Create a copy of Course
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? userId = null,Object? name = null,Object? educationLevel = null,Object? avatarModelUrl = freezed,Object? avatarStatus = null,Object? createdAt = null,Object? updatedAt = null,}) {
  return _then(_Course(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as String,userId: null == userId ? _self.userId : userId // ignore: cast_nullable_to_non_nullable
as String,name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,educationLevel: null == educationLevel ? _self.educationLevel : educationLevel // ignore: cast_nullable_to_non_nullable
as String,avatarModelUrl: freezed == avatarModelUrl ? _self.avatarModelUrl : avatarModelUrl // ignore: cast_nullable_to_non_nullable
as String?,avatarStatus: null == avatarStatus ? _self.avatarStatus : avatarStatus // ignore: cast_nullable_to_non_nullable
as String,createdAt: null == createdAt ? _self.createdAt : createdAt // ignore: cast_nullable_to_non_nullable
as DateTime,updatedAt: null == updatedAt ? _self.updatedAt : updatedAt // ignore: cast_nullable_to_non_nullable
as DateTime,
  ));
}


}


/// @nodoc
mixin _$CreateCourseRequest {

 String get name;@JsonKey(name: 'education_level') String get educationLevel;
/// Create a copy of CreateCourseRequest
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$CreateCourseRequestCopyWith<CreateCourseRequest> get copyWith => _$CreateCourseRequestCopyWithImpl<CreateCourseRequest>(this as CreateCourseRequest, _$identity);

  /// Serializes this CreateCourseRequest to a JSON map.
  Map<String, dynamic> toJson();


@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is CreateCourseRequest&&(identical(other.name, name) || other.name == name)&&(identical(other.educationLevel, educationLevel) || other.educationLevel == educationLevel));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,name,educationLevel);

@override
String toString() {
  return 'CreateCourseRequest(name: $name, educationLevel: $educationLevel)';
}


}

/// @nodoc
abstract mixin class $CreateCourseRequestCopyWith<$Res>  {
  factory $CreateCourseRequestCopyWith(CreateCourseRequest value, $Res Function(CreateCourseRequest) _then) = _$CreateCourseRequestCopyWithImpl;
@useResult
$Res call({
 String name,@JsonKey(name: 'education_level') String educationLevel
});




}
/// @nodoc
class _$CreateCourseRequestCopyWithImpl<$Res>
    implements $CreateCourseRequestCopyWith<$Res> {
  _$CreateCourseRequestCopyWithImpl(this._self, this._then);

  final CreateCourseRequest _self;
  final $Res Function(CreateCourseRequest) _then;

/// Create a copy of CreateCourseRequest
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? name = null,Object? educationLevel = null,}) {
  return _then(_self.copyWith(
name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,educationLevel: null == educationLevel ? _self.educationLevel : educationLevel // ignore: cast_nullable_to_non_nullable
as String,
  ));
}

}


/// Adds pattern-matching-related methods to [CreateCourseRequest].
extension CreateCourseRequestPatterns on CreateCourseRequest {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _CreateCourseRequest value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _CreateCourseRequest() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _CreateCourseRequest value)  $default,){
final _that = this;
switch (_that) {
case _CreateCourseRequest():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _CreateCourseRequest value)?  $default,){
final _that = this;
switch (_that) {
case _CreateCourseRequest() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String name, @JsonKey(name: 'education_level')  String educationLevel)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _CreateCourseRequest() when $default != null:
return $default(_that.name,_that.educationLevel);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String name, @JsonKey(name: 'education_level')  String educationLevel)  $default,) {final _that = this;
switch (_that) {
case _CreateCourseRequest():
return $default(_that.name,_that.educationLevel);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String name, @JsonKey(name: 'education_level')  String educationLevel)?  $default,) {final _that = this;
switch (_that) {
case _CreateCourseRequest() when $default != null:
return $default(_that.name,_that.educationLevel);case _:
  return null;

}
}

}

/// @nodoc
@JsonSerializable()

class _CreateCourseRequest implements CreateCourseRequest {
  const _CreateCourseRequest({required this.name, @JsonKey(name: 'education_level') required this.educationLevel});
  factory _CreateCourseRequest.fromJson(Map<String, dynamic> json) => _$CreateCourseRequestFromJson(json);

@override final  String name;
@override@JsonKey(name: 'education_level') final  String educationLevel;

/// Create a copy of CreateCourseRequest
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$CreateCourseRequestCopyWith<_CreateCourseRequest> get copyWith => __$CreateCourseRequestCopyWithImpl<_CreateCourseRequest>(this, _$identity);

@override
Map<String, dynamic> toJson() {
  return _$CreateCourseRequestToJson(this, );
}

@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _CreateCourseRequest&&(identical(other.name, name) || other.name == name)&&(identical(other.educationLevel, educationLevel) || other.educationLevel == educationLevel));
}

@JsonKey(includeFromJson: false, includeToJson: false)
@override
int get hashCode => Object.hash(runtimeType,name,educationLevel);

@override
String toString() {
  return 'CreateCourseRequest(name: $name, educationLevel: $educationLevel)';
}


}

/// @nodoc
abstract mixin class _$CreateCourseRequestCopyWith<$Res> implements $CreateCourseRequestCopyWith<$Res> {
  factory _$CreateCourseRequestCopyWith(_CreateCourseRequest value, $Res Function(_CreateCourseRequest) _then) = __$CreateCourseRequestCopyWithImpl;
@override @useResult
$Res call({
 String name,@JsonKey(name: 'education_level') String educationLevel
});




}
/// @nodoc
class __$CreateCourseRequestCopyWithImpl<$Res>
    implements _$CreateCourseRequestCopyWith<$Res> {
  __$CreateCourseRequestCopyWithImpl(this._self, this._then);

  final _CreateCourseRequest _self;
  final $Res Function(_CreateCourseRequest) _then;

/// Create a copy of CreateCourseRequest
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? name = null,Object? educationLevel = null,}) {
  return _then(_CreateCourseRequest(
name: null == name ? _self.name : name // ignore: cast_nullable_to_non_nullable
as String,educationLevel: null == educationLevel ? _self.educationLevel : educationLevel // ignore: cast_nullable_to_non_nullable
as String,
  ));
}


}

// dart format on
