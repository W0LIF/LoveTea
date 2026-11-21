class AuthUser {
  final String id;
  final String email;
  final String? emailVerifiedAt;
  final String? createdAt;

  const AuthUser({
    required this.id,
    required this.email,
    this.emailVerifiedAt,
    this.createdAt,
  });

  factory AuthUser.fromJson(Map<String, dynamic> json) {
    return AuthUser(
      id: json['id'].toString(),
      email: json['email'] as String,
      emailVerifiedAt: json['email_verified_at'] as String?,
      createdAt: json['created_at'] as String?,
    );
  }
}

class AuthResponse {
  final AuthUser user;
  final String accessToken;
  final String tokenType;
  final int expiresIn;
  final String refreshToken;

  const AuthResponse({
    required this.user,
    required this.accessToken,
    required this.tokenType,
    required this.expiresIn,
    required this.refreshToken,
  });

  factory AuthResponse.fromJson(Map<String, dynamic> json) {
    return AuthResponse(
      user: AuthUser.fromJson(json['user'] as Map<String, dynamic>),
      accessToken: json['access_token'] as String,
      tokenType: json['token_type'] as String,
      expiresIn: (json['expires_in'] as num).toInt(),
      refreshToken: json['refresh_token'] as String,
    );
  }
}
