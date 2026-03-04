import 'package:freezed_annotation/freezed_annotation.dart';

part 'auth_models.freezed.dart';
part 'auth_models.g.dart';

@freezed
class User with _$User {
  const factory User({
    required String id,
    required String email,
    required String name,
    @JsonKey(name: 'profile_image_url') String? profileImageUrl,
  }) = _User;

  factory User.fromJson(Map<String, dynamic> json) => _$UserFromJson(json);
}

@freezed
class AuthResult with _$AuthResult {
  const factory AuthResult({
    @JsonKey(name: 'access_token') required String accessToken,
    @JsonKey(name: 'refresh_token') required String refreshToken,
    required User user,
  }) = _AuthResult;

  factory AuthResult.fromJson(Map<String, dynamic> json) => _$AuthResultFromJson(json);
}
