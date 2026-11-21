import 'package:dio/dio.dart';

import '../../../core/network/api_client.dart';
import '../../../core/storage/local_database.dart';

class PlanningApi {
  final ApiClient _apiClient;
  final LocalDatabase _database;

  PlanningApi(this._apiClient, this._database);

  Dio get _dio => _apiClient.dio;

  Future<List<Map<String, dynamic>>> listTasks({String? userId}) =>
      _getList(
        '/tasks',
        entityType: 'task',
        userId: userId,
        queryParameters: {'limit': 100, 'offset': 0},
      );

  Future<List<Map<String, dynamic>>> listEvents({String? userId}) => _getList(
    '/calendar/events',
    entityType: 'calendar_event',
    userId: userId,
    queryParameters: {'limit': 100, 'offset': 0},
  );

  Future<List<Map<String, dynamic>>> listImportantDates({String? userId}) =>
      _getList(
        '/calendar/dates',
        entityType: 'important_date',
        userId: userId,
        queryParameters: {'limit': 100, 'offset': 0},
      );

  Future<void> createTask({
    required String title,
    required String scope,
    String description = '',
    DateTime? dueAt,
  }) async {
    await _dio.post(
      '/tasks',
      data: {
        'scope': scope,
        'title': title.trim(),
        'description': description.trim(),
        if (dueAt != null) 'due_at': dueAt.toUtc().toIso8601String(),
      },
    );
  }

  Future<void> setTaskDone({
    required String id,
    required int version,
    required bool done,
  }) async {
    await _dio.patch(
      '/tasks/$id',
      data: {'version': version, 'status': done ? 'done' : 'open'},
    );
  }

  Future<void> createImportantDate({
    required String title,
    required String scope,
    required DateTime date,
    required bool repeatsYearly,
  }) async {
    final dateOnly = _formatDate(date);
    await _dio.post(
      '/calendar/dates',
      data: {
        'scope': scope,
        'title': title.trim(),
        'date': dateOnly,
        'repeats_yearly': repeatsYearly,
      },
    );
  }

  Future<void> createEvent({
    required String title,
    required String scope,
    required DateTime startsAt,
    required DateTime endsAt,
  }) async {
    await _dio.post(
      '/calendar/events',
      data: {
        'scope': scope,
        'title': title.trim(),
        'event_type': 'date',
        'starts_at': startsAt.toUtc().toIso8601String(),
        'ends_at': endsAt.toUtc().toIso8601String(),
      },
    );
  }

  Future<List<Map<String, dynamic>>> _getList(
    String path, {
    Map<String, Object>? queryParameters,
    String? userId,
    required String entityType,
  }) async {
    try {
      final response = await _dio.get(path, queryParameters: queryParameters);
      final data = response.data;
      if (data is! List) {
        throw const FormatException('Ожидался список данных от сервера.');
      }

      final items = data
          .whereType<Map>()
          .map((item) => Map<String, dynamic>.from(item))
          .toList();
      if (userId != null) {
        await _database.cacheSnapshot(
          userId: userId,
          entityType: entityType,
          items: items,
        );
      }
      return items;
    } on DioException catch (error) {
      final isOffline =
          error.type == DioExceptionType.connectionError ||
          error.type == DioExceptionType.connectionTimeout;
      if (!isOffline || userId == null) rethrow;

      final cached = await _database.cachedSnapshot(
        userId: userId,
        entityType: entityType,
      );
      if (cached == null) rethrow;
      return cached;
    }
  }

  static String formatDate(DateTime date) => _formatDate(date);

  static String _formatDate(DateTime date) =>
      '${date.year.toString().padLeft(4, '0')}-'
      '${date.month.toString().padLeft(2, '0')}-'
      '${date.day.toString().padLeft(2, '0')}';
}
