import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:lovetea/core/network/api_client.dart';
import 'package:lovetea/core/providers.dart';
import 'package:lovetea/core/storage/token_storage.dart';
import 'package:lovetea/features/auth/data/profile_api.dart';
import 'package:lovetea/features/auth/presentation/screens/profile_screen.dart';

void main() {
  testWidgets('loads profile and saves an updated display name', (
    tester,
  ) async {
    final api = _FakeProfileApi();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [profileApiProvider.overrideWithValue(api)],
        child: const MaterialApp(home: ProfileScreen()),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('kitty@example.com'), findsOneWidget);
    await tester.enterText(
      find.byKey(const ValueKey('profile-display-name')),
      'Мурка',
    );
    await tester.tap(find.text('Сохранить профиль'));
    await tester.pumpAndSettle();

    expect(api.savedDisplayName, 'Мурка');
  });
}

class _FakeProfileApi extends ProfileApi {
  _FakeProfileApi() : super(ApiClient(TokenStorage()));

  String? savedDisplayName;

  @override
  Future<UserProfile> getProfile() async =>
      const UserProfile(email: 'kitty@example.com', displayName: 'Котёнок');

  @override
  Future<void> updateDisplayName(String displayName) async {
    savedDisplayName = displayName;
  }
}
