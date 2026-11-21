import 'package:dio/dio.dart';

import '../config/app_config.dart';
import '../storage/token_storage.dart';

class ApiClient {
  final TokenStorage tokenStorage;

  late final Dio dio;
  Future<String?>? _refreshInProgress;

  ApiClient(this.tokenStorage) {
    dio = Dio(
      BaseOptions(
        baseUrl: '${AppConfig.baseUrl}${AppConfig.apiPrefix}',
        connectTimeout: const Duration(seconds: 15),
        receiveTimeout: const Duration(seconds: 15),
        sendTimeout: const Duration(seconds: 15),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
        },
      ),
    );

    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) async {
          if (options.extra['skipBearerToken'] == true) {
            handler.next(options);
            return;
          }

          final accessToken = await tokenStorage.getAccessToken();
          if (accessToken != null && accessToken.isNotEmpty) {
            options.headers['Authorization'] = 'Bearer $accessToken';
          }

          handler.next(options);
        },
        onError: (error, handler) async {
          final request = error.requestOptions;
          final statusCode = error.response?.statusCode;
          final canRefresh =
              statusCode == 401 &&
              request.extra['skipTokenRefresh'] != true &&
              request.extra['retriedAfterRefresh'] != true &&
              !request.path.endsWith('/auth/login') &&
              !request.path.endsWith('/auth/register') &&
              !request.path.endsWith('/auth/refresh');

          if (!canRefresh) {
            handler.next(error);
            return;
          }

          try {
            final accessToken = await _refreshAccessToken();
            if (accessToken == null) {
              handler.next(error);
              return;
            }

            request.headers['Authorization'] = 'Bearer $accessToken';
            request.extra['retriedAfterRefresh'] = true;
            handler.resolve(await dio.fetch<dynamic>(request));
          } on DioException catch (refreshError) {
            if (refreshError.response?.statusCode == 400 ||
                refreshError.response?.statusCode == 401) {
              await tokenStorage.clearTokens();
            }
            handler.reject(refreshError);
          } catch (refreshError) {
            handler.reject(
              DioException(requestOptions: request, error: refreshError),
            );
          }
        },
      ),
    );
  }

  Future<String?> _refreshAccessToken() {
    return _refreshInProgress ??= _requestNewAccessToken().whenComplete(() {
      _refreshInProgress = null;
    });
  }

  Future<String?> _requestNewAccessToken() async {
    final refreshToken = await tokenStorage.getRefreshToken();
    if (refreshToken == null || refreshToken.isEmpty) return null;

    final response = await dio.post<dynamic>(
      '/auth/refresh',
      data: {'refresh_token': refreshToken},
      options: Options(
        headers: {'Authorization': null},
        extra: {'skipBearerToken': true, 'skipTokenRefresh': true},
      ),
    );
    final data = Map<String, dynamic>.from(response.data as Map);
    final accessToken = data['access_token'] as String;
    final rotatedRefreshToken = data['refresh_token'] as String;
    await tokenStorage.saveTokens(
      accessToken: accessToken,
      refreshToken: rotatedRefreshToken,
    );
    return accessToken;
  }
}
