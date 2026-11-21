import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:lovetea/core/network/api_client.dart';
import 'package:lovetea/core/providers.dart';
import 'package:lovetea/core/storage/token_storage.dart';
import 'package:lovetea/features/onboarding/data/onboarding_api.dart';
import 'package:lovetea/features/onboarding/presentation/onboarding_screen.dart';

void main() {
  testWidgets('shows an honest empty state when no tests are published', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          onboardingApiProvider.overrideWithValue(_EmptyOnboardingApi()),
        ],
        child: const MaterialApp(home: OnboardingScreen()),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Тесты появятся позже'), findsOneWidget);
    expect(
      find.text('Пока нет опубликованных тестов. Загляните сюда позже.'),
      findsOneWidget,
    );
  });
}

class _EmptyOnboardingApi extends OnboardingApi {
  _EmptyOnboardingApi() : super(ApiClient(TokenStorage()));

  @override
  Future<List<OnboardingTest>> listTests() async => [];

  @override
  Future<List<OnboardingSubmission>> listMyAnswers() async => [];
}
