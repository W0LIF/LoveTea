import 'package:flutter_test/flutter_test.dart';
import 'package:lovetea/features/onboarding/data/onboarding_api.dart';

void main() {
  test('parses translated questions and sorts them by position', () {
    final test = OnboardingTest.fromJson({
      'slug': 'getting-to-know-you',
      'version': 3,
      'title': {'en': 'Getting to know you'},
      'description': {'en': 'A short test'},
      'questions': [
        {
          'key': 'mood',
          'prompt': {'ru': 'Как настроение?', 'en': 'How are you?'},
          'type': 'single_choice',
          'position': 2,
          'required': true,
          'options': [
            {
              'value': 'good',
              'label': {'ru': 'Хорошо', 'en': 'Good'},
            },
          ],
          'settings': {},
        },
        {
          'key': 'age',
          'prompt': {'en': 'How old are you?'},
          'type': 'number',
          'position': 1,
          'required': false,
          'options': [],
          'settings': {'min': 18, 'max': 99},
        },
      ],
    });

    expect(test.slug, 'getting-to-know-you');
    expect(test.version, 3);
    expect(test.questions.map((question) => question.key), ['age', 'mood']);
    expect(test.questions.first.min, 18);
    expect(test.questions.last.options.single.value, 'good');
    expect(test.questions.last.options.single.label['ru'], 'Хорошо');
  });

  test('parses a saved submission without coercing answer values', () {
    final submission = OnboardingSubmission.fromJson({
      'test_slug': 'habits',
      'test_version': 2,
      'answers': {
        'wake_up_early': true,
        'favorite_days': ['saturday', 'sunday'],
        'hours': 7.5,
      },
    });

    expect(submission.testSlug, 'habits');
    expect(submission.testVersion, 2);
    expect(submission.answers['wake_up_early'], isTrue);
    expect(submission.answers['favorite_days'], ['saturday', 'sunday']);
    expect(submission.answers['hours'], 7.5);
  });
}
