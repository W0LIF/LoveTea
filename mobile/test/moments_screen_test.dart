import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:lovetea/core/network/api_client.dart';
import 'package:lovetea/core/providers.dart';
import 'package:lovetea/core/storage/token_storage.dart';
import 'package:lovetea/features/moments/data/moments_api.dart';
import 'package:lovetea/features/moments/presentation/moments_screen.dart';

void main() {
  testWidgets('saves a text-only moment without requiring photo access', (
    tester,
  ) async {
    final api = _FakeMomentsApi();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [momentsApiProvider.overrideWithValue(api)],
        child: const MaterialApp(home: MomentsScreen()),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Добавить момент').first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Сохранить').last);
    await tester.pump();

    expect(
      find.text('Добавьте заголовок длиной до 200 символов.'),
      findsOneWidget,
    );
    expect(api.savedTitle, isNull);

    await tester.enterText(find.byType(TextField).first, 'Первый день');
    await tester.tap(find.text('Сохранить').last);
    await tester.pumpAndSettle();

    expect(api.savedTitle, 'Первый день');
    expect(api.savedMediaIds, isEmpty);
    expect(find.text('Первый момент ещё впереди'), findsOneWidget);
  });
}

class _FakeMomentsApi extends MomentsApi {
  _FakeMomentsApi() : super(ApiClient(TokenStorage()));

  String? savedTitle;
  List<String> savedMediaIds = [];

  @override
  Future<List<Map<String, dynamic>>> listMoments() async => [];

  @override
  Future<void> createMoment({
    required String title,
    required String caption,
    required String scope,
    required DateTime happenedAt,
    List<String> mediaIds = const [],
  }) async {
    savedTitle = title;
    savedMediaIds = mediaIds;
  }
}
