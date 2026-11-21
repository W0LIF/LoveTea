import '../../../core/network/api_client.dart';

class UserProfile {
  final String email;
  final String? displayName;

  const UserProfile({required this.email, required this.displayName});

  factory UserProfile.fromJson(Map<String, dynamic> json) => UserProfile(
    email: json['email'] as String,
    displayName: json['display_name'] as String?,
  );
}

class ProfileApi {
  final ApiClient _apiClient;

  ProfileApi(this._apiClient);

  Future<UserProfile> getProfile() async {
    final response = await _apiClient.dio.get('/users/me');
    return UserProfile.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }

  Future<void> updateDisplayName(String displayName) async {
    await _apiClient.dio.patch(
      '/users/me',
      data: {'display_name': displayName.trim()},
    );
  }
}
