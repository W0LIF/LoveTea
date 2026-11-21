import 'dart:convert';

import '../storage/local_database.dart';
import 'sync_api.dart';

class SyncService {
  final SyncApi _api;
  final LocalDatabase _database;

  SyncService(this._api, this._database);

  Future<int> synchronize(String userId) async {
    await pushPending(userId);
    return pullAll(userId);
  }

  Future<void> pushPending(String userId) async {
    while (true) {
      final pending = await _database.pendingOperations(userId);
      if (pending.isEmpty) return;

      final operations = pending.map((item) {
        final payload = item.payload.isEmpty
            ? <String, dynamic>{}
            : Map<String, dynamic>.from(jsonDecode(item.payload) as Map);
        return <String, dynamic>{
          'operation_id': item.operationId,
          'entity_type': item.entityType,
          'entity_id': item.entityId,
          'operation': item.operation,
          'base_version': item.baseVersion,
          if (payload.isNotEmpty) 'payload': payload,
        };
      }).toList();
      final results = await _api.push(operations);
      final resultsById = {
        for (final result in results) result.operationId: result,
      };
      var resolvedCount = 0;

      for (final operation in pending) {
        final result = resultsById[operation.operationId];
        if (result == null) continue;
        if (!{'applied', 'conflict', 'blocked'}.contains(result.status)) {
          throw FormatException(
            'Unknown sync operation status: ${result.status}',
          );
        }
        await _database.resolveOutboxOperation(
          userId: userId,
          operationId: result.operationId,
          status: result.status,
        );
        resolvedCount++;
      }

      if (resolvedCount == 0) return;
    }
  }

  Future<int> pullAll(String userId) async {
    var cursor = await _database.cursorFor(userId);
    var appliedChanges = 0;

    while (true) {
      final page = await _api.pull(cursor: cursor);
      if (page.hasMore && page.nextCursor <= cursor) {
        throw StateError('Sync server returned a non-advancing cursor.');
      }

      await _database.applySyncPage(userId: userId, page: page);
      appliedChanges += page.changes.length;
      cursor = page.nextCursor;

      if (!page.hasMore) return appliedChanges;
    }
  }
}
