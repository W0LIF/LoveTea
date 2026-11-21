import 'package:dio/dio.dart';

import '../../../core/network/api_client.dart';

class OnboardingTest {
  final String slug;
  final int version;
  final Map<String, String> title;
  final Map<String, String> description;
  final List<OnboardingQuestion> questions;

  const OnboardingTest({
    required this.slug,
    required this.version,
    required this.title,
    required this.description,
    required this.questions,
  });

  factory OnboardingTest.fromJson(Map<String, dynamic> json) =>
      OnboardingTest(
        slug: json['slug'] as String,
        version: (json['version'] as num).toInt(),
        title: _translatedText(json['title']),
        description: _translatedText(json['description']),
        questions: (json['questions'] as List)
            .map(
              (question) => OnboardingQuestion.fromJson(
                Map<String, dynamic>.from(question as Map),
              ),
            )
            .toList()
          ..sort((left, right) => left.position.compareTo(right.position)),
      );
}

class OnboardingQuestion {
  final String key;
  final Map<String, String> prompt;
  final String type;
  final int position;
  final bool required;
  final List<OnboardingOption> options;
  final double? min;
  final double? max;
  final int? maxLength;

  const OnboardingQuestion({
    required this.key,
    required this.prompt,
    required this.type,
    required this.position,
    required this.required,
    required this.options,
    required this.min,
    required this.max,
    required this.maxLength,
  });

  factory OnboardingQuestion.fromJson(Map<String, dynamic> json) {
    final settings = Map<String, dynamic>.from(json['settings'] as Map);
    final options = json['options'] as List;
    return OnboardingQuestion(
      key: json['key'] as String,
      prompt: _translatedText(json['prompt']),
      type: json['type'] as String,
      position: (json['position'] as num).toInt(),
      required: json['required'] as bool,
      options: options
          .map(
            (option) => OnboardingOption.fromJson(
              Map<String, dynamic>.from(option as Map),
            ),
          )
          .toList(),
      min: (settings['min'] as num?)?.toDouble(),
      max: (settings['max'] as num?)?.toDouble(),
      maxLength: (settings['max_length'] as num?)?.toInt(),
    );
  }
}

class OnboardingOption {
  final String value;
  final Map<String, String> label;

  const OnboardingOption({required this.value, required this.label});

  factory OnboardingOption.fromJson(Map<String, dynamic> json) =>
      OnboardingOption(
        value: json['value'] as String,
        label: _translatedText(json['label']),
      );
}

class OnboardingSubmission {
  final String testSlug;
  final int testVersion;
  final Map<String, dynamic> answers;

  const OnboardingSubmission({
    required this.testSlug,
    required this.testVersion,
    required this.answers,
  });

  factory OnboardingSubmission.fromJson(Map<String, dynamic> json) =>
      OnboardingSubmission(
        testSlug: json['test_slug'] as String,
        testVersion: (json['test_version'] as num).toInt(),
        answers: Map<String, dynamic>.from(json['answers'] as Map),
      );
}

class OnboardingApi {
  final ApiClient _apiClient;

  OnboardingApi(this._apiClient);

  Dio get _dio => _apiClient.dio;

  Future<List<OnboardingTest>> listTests() async {
    final response = await _dio.get('/onboarding/tests');
    final data = response.data;
    if (data is! List) {
      throw const FormatException('Ожидался каталог onboarding-тестов.');
    }

    return data
        .whereType<Map>()
        .map(
          (test) => OnboardingTest.fromJson(Map<String, dynamic>.from(test)),
        )
        .toList();
  }

  Future<List<OnboardingSubmission>> listMyAnswers() async {
    final response = await _dio.get('/onboarding/answers');
    final data = response.data;
    if (data is! List) {
      throw const FormatException('Ожидался список ответов onboarding.');
    }

    return data
        .whereType<Map>()
        .map(
          (answer) =>
              OnboardingSubmission.fromJson(Map<String, dynamic>.from(answer)),
        )
        .toList();
  }

  Future<void> submitAnswers({
    required String testSlug,
    required int version,
    required Map<String, dynamic> answers,
  }) async {
    await _dio.post(
      '/onboarding/answers',
      data: {
        'test_slug': testSlug,
        'version': version,
        'answers': answers,
      },
    );
  }
}

Map<String, String> _translatedText(Object? value) {
  if (value is! Map) {
    throw const FormatException('Ожидался перевод onboarding-текста.');
  }
  return value.map(
    (key, text) => MapEntry(key.toString(), text.toString()),
  );
}
