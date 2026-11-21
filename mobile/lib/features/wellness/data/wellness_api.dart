import 'package:dio/dio.dart';

import '../../../core/network/api_client.dart';
import '../../planning/data/planning_api.dart';

class WellnessApi {
  final ApiClient _apiClient;

  WellnessApi(this._apiClient);

  Dio get _dio => _apiClient.dio;

  Future<List<Map<String, dynamic>>> listMood() async {
    final response = await _dio.get(
      '/wellness/mood',
      queryParameters: {
        'from': PlanningApi.formatDate(
          DateTime.now().subtract(const Duration(days: 30)),
        ),
        'to': PlanningApi.formatDate(DateTime.now()),
      },
    );
    final data = response.data;
    if (data is! List) {
      throw const FormatException('Ожидался список отметок настроения.');
    }
    return data
        .whereType<Map>()
        .map((item) => Map<String, dynamic>.from(item))
        .toList();
  }

  Future<List<Map<String, dynamic>>> listSleep() async {
    final response = await _dio.get(
      '/wellness/sleep',
      queryParameters: {
        'from': PlanningApi.formatDate(
          DateTime.now().subtract(const Duration(days: 30)),
        ),
        'to': PlanningApi.formatDate(DateTime.now()),
      },
    );
    final data = response.data;
    if (data is! List) {
      throw const FormatException('Ожидался список записей сна.');
    }
    return data
        .whereType<Map>()
        .map((item) => Map<String, dynamic>.from(item))
        .toList();
  }

  Future<void> saveMood({
    required String mood,
    required String note,
    required bool shareWithCouple,
    int? version,
  }) async {
    await _dio.post(
      '/wellness/mood',
      data: {
        'entry_date': PlanningApi.formatDate(DateTime.now()),
        'mood': mood,
        'note': note.trim(),
        'shared_with_couple': shareWithCouple,
        'version': ?version,
      },
    );
  }

  Future<void> saveSleep({
    required DateTime sleptAt,
    required DateTime wokeAt,
    required int quality,
    required bool shareWithCouple,
    int? version,
  }) async {
    await _dio.post(
      '/wellness/sleep',
      data: {
        'entry_date': PlanningApi.formatDate(wokeAt),
        'sleep_started_at': sleptAt.toUtc().toIso8601String(),
        'woke_at': wokeAt.toUtc().toIso8601String(),
        'quality': quality,
        'shared_with_couple': shareWithCouple,
        'version': ?version,
      },
    );
  }
}
