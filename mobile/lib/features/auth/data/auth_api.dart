import 'package:dio/dio.dart';

import '../../../core/network/api_client.dart';
import '../../../core/storage/token_storage.dart';
import 'models/auth_response.dart';

class AuthApi {
  final ApiClient apiClient;
  final TokenStorage tokenStorage;

  AuthApi({required this.apiClient, required this.tokenStorage});

  Dio get _dio => apiClient.dio;

  Future<AuthResponse> register({
    required String email,
    required String password,
  }) async {
    final response = await _dio.post(
      '/auth/register',
      data: {'email': email.trim(), 'password': password},
    );

    final authResponse = AuthResponse.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );

    await _saveAuth(authResponse);

    return authResponse;
  }

  Future<AuthResponse> login({
    required String email,
    required String password,
  }) async {
    final response = await _dio.post(
      '/auth/login',
      data: {'email': email.trim(), 'password': password},
    );

    final authResponse = AuthResponse.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );

    await _saveAuth(authResponse);

    return authResponse;
  }

  Future<AuthResponse> refresh() async {
    final refreshToken = await tokenStorage.getRefreshToken();

    if (refreshToken == null || refreshToken.isEmpty) {
      throw Exception('Refresh-токен отсутствует');
    }

    final response = await _dio.post(
      '/auth/refresh',
      data: {'refresh_token': refreshToken},
      options: Options(headers: {'Authorization': null}),
    );

    final authResponse = AuthResponse.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );

    await _saveAuth(authResponse);

    return authResponse;
  }

  Future<AuthUser> getMe() async {
    final response = await _dio.get('/auth/me');

    final data = Map<String, dynamic>.from(response.data as Map);

    // Поддерживаем как объект пользователя напрямую,
    // так и ответ с обёрткой { "user": {...} }.
    final userJson = data['user'] is Map
        ? Map<String, dynamic>.from(data['user'] as Map)
        : data;

    return AuthUser.fromJson(userJson);
  }

  Future<bool> restoreSession() async {
    final accessToken = await tokenStorage.getAccessToken();
    final refreshToken = await tokenStorage.getRefreshToken();
    if (accessToken == null || accessToken.isEmpty) {
      if (refreshToken == null || refreshToken.isEmpty) return false;
      await refresh();
    }

    try {
      await getMe();
      return true;
    } on DioException catch (error) {
      if (error.response?.statusCode != 401) rethrow;
      await refresh();
      await getMe();
      return true;
    }
  }

  Future<void> logout() async {
    final refreshToken = await tokenStorage.getRefreshToken();

    try {
      if (refreshToken != null && refreshToken.isNotEmpty) {
        await _dio.post('/auth/logout', data: {'refresh_token': refreshToken});
      }
    } finally {
      await tokenStorage.clearTokens();
    }
  }

  Future<void> _saveAuth(AuthResponse response) async {
    await tokenStorage.saveTokens(
      accessToken: response.accessToken,
      refreshToken: response.refreshToken,
    );
  }
}
