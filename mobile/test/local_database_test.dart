import 'package:drift/native.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:lovetea/core/storage/local_database.dart';
import 'package:lovetea/core/sync/sync_api.dart';

void main() {
  late LocalDatabase database;

  setUp(() {
    database = LocalDatabase.forTesting(NativeDatabase.memory());
  });

  tearDown(() async {
    await database.close();
  });

  test('stores pulled records and advances cursor transactionally', () async {
    await database.applySyncPage(
      userId: 'user-1',
      page: SyncPage(
        changes: [
          SyncChange(
            cursor: 1,
            entityType: 'task',
            entityId: 'task-1',
            version: 1,
            operation: 'upsert',
            payload: {'scope': 'personal', 'title': 'Read'},
            createdAt: DateTime.utc(2026, 11, 21),
          ),
        ],
        nextCursor: 1,
        hasMore: false,
      ),
    );

    final records = await database.cachedRecords(
      userId: 'user-1',
      entityType: 'task',
    );
    expect(await database.cursorFor('user-1'), 1);
    expect(records.single.payload, contains('"title":"Read"'));
    expect(records.single.isDeleted, isFalse);
  });

  test(
    'tombstones hide records and revocation removes shared cached data',
    () async {
      await database.applySyncPage(
        userId: 'user-1',
        page: SyncPage(
          changes: [
            _change(
              cursor: 1,
              id: 'personal-1',
              version: 1,
              payload: {'scope': 'personal'},
            ),
            _change(
              cursor: 2,
              id: 'couple-1',
              version: 1,
              payload: {'scope': 'couple'},
            ),
          ],
          nextCursor: 2,
          hasMore: false,
        ),
      );
      await database.applySyncPage(
        userId: 'user-1',
        page: SyncPage(
          changes: [
            _change(
              cursor: 3,
              id: 'personal-1',
              version: 2,
              operation: 'delete',
              payload: {'scope': 'personal'},
            ),
            _change(
              cursor: 4,
              id: 'couple-1',
              version: 2,
              entityType: 'couple_access_revoked',
              operation: 'delete',
              payload: const {},
            ),
          ],
          nextCursor: 4,
          hasMore: false,
        ),
      );

      expect(
        await database.cachedRecords(userId: 'user-1', entityType: 'task'),
        isEmpty,
      );
      expect(await database.cursorFor('user-1'), 4);
    },
  );

  test(
    'does not replace a newer cached version with an older change',
    () async {
      await database.applySyncPage(
        userId: 'user-1',
        page: SyncPage(
          changes: [
            _change(
              cursor: 2,
              id: 'task-1',
              version: 2,
              payload: {'scope': 'personal', 'title': 'New'},
            ),
          ],
          nextCursor: 2,
          hasMore: false,
        ),
      );
      await database.applySyncPage(
        userId: 'user-1',
        page: SyncPage(
          changes: [
            _change(
              cursor: 3,
              id: 'task-1',
              version: 1,
              payload: {'scope': 'personal', 'title': 'Old'},
            ),
          ],
          nextCursor: 3,
          hasMore: false,
        ),
      );

      final record = (await database.cachedRecords(
        userId: 'user-1',
        entityType: 'task',
      )).single;
      expect(record.version, 2);
      expect(record.payload, contains('"title":"New"'));
      expect(await database.cursorFor('user-1'), 3);
    },
  );

  test('rolls back cached changes when cursor application fails', () async {
    await database.applySyncPage(
      userId: 'user-1',
      page: const SyncPage(changes: [], nextCursor: 1, hasMore: false),
    );
    await expectLater(
      database.applySyncPage(
        userId: 'user-1',
        page: SyncPage(
          changes: [
            _change(
              cursor: 1,
              id: 'task-1',
              version: 1,
              payload: {'scope': 'personal', 'title': 'Pending'},
            ),
          ],
          nextCursor: 0,
          hasMore: false,
        ),
      ),
      throwsStateError,
    );

    expect(await database.cursorFor('user-1'), 1);
    expect(
      await database.cachedRecords(userId: 'user-1', entityType: 'task'),
      isEmpty,
    );
  });

  test(
    'keeps conflicts in the outbox and removes only applied operations',
    () async {
      await database.enqueueOperation(
        userId: 'user-1',
        operationId: 'op-applied',
        entityType: 'task',
        entityId: 'task-1',
        operation: 'upsert',
        baseVersion: 0,
        payload: {'scope': 'personal', 'title': 'First'},
      );
      await database.enqueueOperation(
        userId: 'user-1',
        operationId: 'op-conflict',
        entityType: 'task',
        entityId: 'task-2',
        operation: 'upsert',
        baseVersion: 1,
        payload: {'scope': 'couple', 'title': 'Shared'},
      );

      await database.resolveOutboxOperation(
        userId: 'user-1',
        operationId: 'op-applied',
        status: 'applied',
      );
      await database.resolveOutboxOperation(
        userId: 'user-1',
        operationId: 'op-conflict',
        status: 'conflict',
      );

      expect(await database.pendingOperations('user-1'), isEmpty);
      final conflicts = await (database.select(
        database.syncOutbox,
      )..where((operation) => operation.userId.equals('user-1'))).get();
      expect(conflicts, hasLength(1));
      expect(conflicts.single.status, 'conflict');
    },
  );
}

SyncChange _change({
  required int cursor,
  required String id,
  required int version,
  required Map<String, dynamic> payload,
  String entityType = 'task',
  String operation = 'upsert',
}) => SyncChange(
  cursor: cursor,
  entityType: entityType,
  entityId: id,
  version: version,
  operation: operation,
  payload: payload,
  createdAt: DateTime.utc(2026, 11, 21),
);
