import 'package:dio/dio.dart';

import '../network/api_client.dart';

class SyncChange {
  final int cursor;
  final String entityType;
  final String entityId;
  final int version;
  final String operation;
  final Map<String, dynamic> payload;
  final DateTime createdAt;

  const SyncChange({
    required this.cursor,
    required this.entityType,
    required this.entityId,
    required this.version,
    required this.operation,
    required this.payload,
    required this.createdAt,
  });

  factory SyncChange.fromJson(Map<String, dynamic> json) => SyncChange(
    cursor: (json['cursor'] as num).toInt(),
    entityType: json['entity_type'] as String,
    entityId: json['entity_id'] as String,
    version: (json['version'] as num).toInt(),
    operation: json['operation'] as String,
    payload: Map<String, dynamic>.from(json['payload'] as Map),
    createdAt: DateTime.parse(json['created_at'] as String),
  );
}

class SyncPage {
  final List<SyncChange> changes;
  final int nextCursor;
  final bool hasMore;

  const SyncPage({
    required this.changes,
    required this.nextCursor,
    required this.hasMore,
  });

  factory SyncPage.fromJson(Map<String, dynamic> json) {
    final rawChanges = json['changes'];
    if (rawChanges is! List) {
      throw const FormatException('Ожидался список изменений sync.');
    }

    return SyncPage(
      changes: rawChanges
          .whereType<Map>()
          .map(
            (change) => SyncChange.fromJson(Map<String, dynamic>.from(change)),
          )
          .toList(),
      nextCursor: (json['next_cursor'] as num).toInt(),
      hasMore: json['has_more'] as bool,
    );
  }
}

class SyncPushResult {
  final String operationId;
  final String status;
  final bool replayed;
  final String? error;

  const SyncPushResult({
    required this.operationId,
    required this.status,
    required this.replayed,
    required this.error,
  });

  factory SyncPushResult.fromJson(Map<String, dynamic> json) => SyncPushResult(
    operationId: json['operation_id'] as String,
    status: json['status'] as String,
    replayed: json['replayed'] as bool,
    error: json['error'] as String?,
  );
}

class SyncApi {
  final ApiClient _apiClient;

  SyncApi(this._apiClient);

  Dio get _dio => _apiClient.dio;

  Future<SyncPage> pull({required int cursor, int limit = 100}) async {
    if (cursor < 0 || limit < 1 || limit > 100) {
      throw ArgumentError.value(
        (cursor: cursor, limit: limit),
        'pagination',
        'cursor must be non-negative and limit must be between 1 and 100',
      );
    }

    final response = await _dio.get(
      '/sync/pull',
      queryParameters: {'cursor': cursor, 'limit': limit},
    );
    return SyncPage.fromJson(Map<String, dynamic>.from(response.data as Map));
  }

  Future<List<SyncPushResult>> push(
    List<Map<String, dynamic>> operations,
  ) async {
    if (operations.isEmpty || operations.length > 50) {
      throw ArgumentError.value(
        operations.length,
        'operations.length',
        'must be between 1 and 50',
      );
    }

    final response = await _dio.post(
      '/sync/push',
      data: {'operations': operations},
    );
    final body = Map<String, dynamic>.from(response.data as Map);
    final rawResults = body['results'];
    if (rawResults is! List) {
      throw const FormatException('Ожидался результат sync push.');
    }
    return rawResults
        .whereType<Map>()
        .map(
          (result) =>
              SyncPushResult.fromJson(Map<String, dynamic>.from(result)),
        )
        .toList();
  }
}
