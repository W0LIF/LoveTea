import 'package:dio/dio.dart';

import '../../../core/network/api_client.dart';

class MomentsApi {
  final ApiClient _apiClient;

  MomentsApi(this._apiClient);

  Dio get _dio => _apiClient.dio;

  Future<List<Map<String, dynamic>>> listMoments() async {
    final response = await _dio.get('/moments');
    final data = response.data;
    if (data is! List) {
      throw const FormatException('Ожидался список моментов от сервера.');
    }
    return data
        .whereType<Map>()
        .map((item) => Map<String, dynamic>.from(item))
        .toList();
  }

  Future<void> createMoment({
    required String title,
    required String caption,
    required String scope,
    required DateTime happenedAt,
    List<String> mediaIds = const [],
  }) async {
    await _dio.post(
      '/moments',
      data: {
        'scope': scope,
        'title': title.trim(),
        'caption': caption.trim(),
        'happened_at': happenedAt.toUtc().toIso8601String(),
        'media_ids': mediaIds,
      },
    );
  }
}
