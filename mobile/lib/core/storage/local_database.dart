import 'dart:convert';

import 'package:drift/drift.dart';
import 'package:drift_flutter/drift_flutter.dart';

import '../sync/sync_api.dart';

part 'local_database.g.dart';

class SyncCursors extends Table {
  TextColumn get userId => text()();
  IntColumn get cursor => integer()();

  @override
  Set<Column<Object>> get primaryKey => {userId};
}

class SyncRecords extends Table {
  TextColumn get userId => text()();
  TextColumn get entityType => text()();
  TextColumn get entityId => text()();
  IntColumn get version => integer()();
  IntColumn get cursor => integer()();
  TextColumn get payload => text()();
  BoolColumn get isCouple => boolean().withDefault(const Constant(false))();
  BoolColumn get isDeleted => boolean().withDefault(const Constant(false))();

  @override
  Set<Column<Object>> get primaryKey => {userId, entityType, entityId};
}

class SyncOutbox extends Table {
  TextColumn get userId => text()();
  TextColumn get operationId => text()();
  TextColumn get entityType => text()();
  TextColumn get entityId => text()();
  TextColumn get operation => text()();
  IntColumn get baseVersion => integer()();
  TextColumn get payload => text()();
  TextColumn get status => text().withDefault(const Constant('pending'))();
  BoolColumn get isCouple => boolean().withDefault(const Constant(false))();
  DateTimeColumn get createdAt => dateTime()();

  @override
  Set<Column<Object>> get primaryKey => {userId, operationId};
}

class SyncCacheMarkers extends Table {
  TextColumn get userId => text()();
  TextColumn get entityType => text()();
  DateTimeColumn get cachedAt => dateTime()();

  @override
  Set<Column<Object>> get primaryKey => {userId, entityType};
}

@DriftDatabase(
  tables: [SyncCursors, SyncRecords, SyncOutbox, SyncCacheMarkers],
)
class LocalDatabase extends _$LocalDatabase {
  LocalDatabase() : super(driftDatabase(name: 'lovetea_local'));

  LocalDatabase.forTesting(super.executor);

  @override
  int get schemaVersion => 1;

  Future<int> cursorFor(String userId) async {
    final row = await (select(
      syncCursors,
    )..where((cursor) => cursor.userId.equals(userId))).getSingleOrNull();
    return row?.cursor ?? 0;
  }

  Future<void> applySyncPage({
    required String userId,
    required SyncPage page,
  }) async {
    await transaction(() async {
      for (final change in page.changes) {
        if (change.entityType == 'couple_access_revoked') {
          await (delete(syncRecords)..where(
                (record) =>
                    record.userId.equals(userId) & record.isCouple.equals(true),
              ))
              .go();
          await (delete(syncOutbox)..where(
                (operation) =>
                    operation.userId.equals(userId) &
                    operation.isCouple.equals(true),
              ))
              .go();
          continue;
        }

        final existing =
            await (select(syncRecords)..where(
                  (record) =>
                      record.userId.equals(userId) &
                      record.entityType.equals(change.entityType) &
                      record.entityId.equals(change.entityId),
                ))
                .getSingleOrNull();
        if (existing != null && existing.version > change.version) continue;

        final isCouple =
            change.payload['scope'] == 'couple' ||
            (change.payload['scope'] == null && existing?.isCouple == true);
        await into(syncRecords).insertOnConflictUpdate(
          SyncRecordsCompanion.insert(
            userId: userId,
            entityType: change.entityType,
            entityId: change.entityId,
            version: change.version,
            cursor: change.cursor,
            payload: jsonEncode(change.payload),
            isCouple: Value(isCouple),
            isDeleted: Value(change.operation == 'delete'),
          ),
        );
      }

      final currentCursor = await cursorFor(userId);
      if (page.nextCursor < currentCursor) {
        throw StateError('Sync cursor cannot move backwards.');
      }
      await into(syncCursors).insertOnConflictUpdate(
        SyncCursorsCompanion.insert(userId: userId, cursor: page.nextCursor),
      );
    });
  }

  Future<List<SyncRecord>> cachedRecords({
    required String userId,
    required String entityType,
  }) =>
      (select(syncRecords)..where(
            (record) =>
                record.userId.equals(userId) &
                record.entityType.equals(entityType) &
                record.isDeleted.equals(false),
          ))
          .get();

  Future<void> cacheSnapshot({
    required String userId,
    required String entityType,
    required List<Map<String, dynamic>> items,
  }) async {
    final currentCursor = await cursorFor(userId);
    await transaction(() async {
      for (final item in items) {
        final id = item['id'];
        final version = item['version'];
        if (id is! String || version is! num) {
          throw const FormatException(
            'Sync cache items require string ids and numeric versions.',
          );
        }
        final existing = await (select(syncRecords)..where(
              (record) =>
                  record.userId.equals(userId) &
                  record.entityType.equals(entityType) &
                  record.entityId.equals(id),
            ))
            .getSingleOrNull();
        if (existing != null && existing.version > version.toInt()) continue;
        await into(syncRecords).insertOnConflictUpdate(
          SyncRecordsCompanion.insert(
            userId: userId,
            entityType: entityType,
            entityId: id,
            version: version.toInt(),
            cursor: currentCursor,
            payload: jsonEncode(item),
            isCouple: Value(item['scope'] == 'couple'),
            isDeleted: const Value(false),
          ),
        );
      }

      await into(syncCacheMarkers).insertOnConflictUpdate(
        SyncCacheMarkersCompanion.insert(
          userId: userId,
          entityType: entityType,
          cachedAt: DateTime.now().toUtc(),
        ),
      );
    });
  }

  Future<List<Map<String, dynamic>>?> cachedSnapshot({
    required String userId,
    required String entityType,
  }) async {
    final marker =
        await (select(syncCacheMarkers)..where(
              (cache) =>
                  cache.userId.equals(userId) &
                  cache.entityType.equals(entityType),
            ))
            .getSingleOrNull();
    if (marker == null) return null;

    final records = await cachedRecords(
      userId: userId,
      entityType: entityType,
    );
    return records
        .map(
          (record) =>
              Map<String, dynamic>.from(jsonDecode(record.payload) as Map),
        )
        .toList();
  }

  Future<List<SyncOutboxData>> pendingOperations(
    String userId, {
    int limit = 50,
  }) =>
      (select(syncOutbox)
            ..where(
              (operation) =>
                  operation.userId.equals(userId) &
                  operation.status.equals('pending'),
            )
            ..orderBy([(operation) => OrderingTerm.asc(operation.createdAt)])
            ..limit(limit))
          .get();

  Future<void> resolveOutboxOperation({
    required String userId,
    required String operationId,
    required String status,
  }) async {
    if (status == 'applied') {
      await (delete(syncOutbox)..where(
            (operation) =>
                operation.userId.equals(userId) &
                operation.operationId.equals(operationId),
          ))
          .go();
      return;
    }

    await (update(syncOutbox)..where(
          (operation) =>
              operation.userId.equals(userId) &
              operation.operationId.equals(operationId),
        ))
        .write(SyncOutboxCompanion(status: Value(status)));
  }

  Future<void> enqueueOperation({
    required String userId,
    required String operationId,
    required String entityType,
    required String entityId,
    required String operation,
    required int baseVersion,
    required Map<String, dynamic> payload,
  }) async {
    await into(syncOutbox).insertOnConflictUpdate(
      SyncOutboxCompanion.insert(
        userId: userId,
        operationId: operationId,
        entityType: entityType,
        entityId: entityId,
        operation: operation,
        baseVersion: baseVersion,
        payload: jsonEncode(payload),
        isCouple: Value(payload['scope'] == 'couple'),
        createdAt: DateTime.now().toUtc(),
      ),
    );
  }
}
