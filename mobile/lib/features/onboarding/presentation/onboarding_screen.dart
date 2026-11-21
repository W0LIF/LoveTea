import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_error_message.dart';
import '../../../core/providers.dart';
import '../../../core/theme/app_theme.dart';
import '../data/onboarding_api.dart';

class OnboardingScreen extends ConsumerStatefulWidget {
  const OnboardingScreen({super.key});

  @override
  ConsumerState<OnboardingScreen> createState() => _OnboardingScreenState();
}

class _OnboardingScreenState extends ConsumerState<OnboardingScreen> {
  List<OnboardingTest> _tests = [];
  final Map<String, OnboardingSubmission> _submissions = {};
  Object? _error;
  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });
    try {
      final api = ref.read(onboardingApiProvider);
      final results = await Future.wait<Object>([
        api.listTests(),
        api.listMyAnswers(),
      ]);
      if (!mounted) return;
      setState(() {
        _tests = results[0] as List<OnboardingTest>;
        _submissions
          ..clear()
          ..addEntries(
            (results[1] as List<OnboardingSubmission>).map(
              (submission) => MapEntry(submission.testSlug, submission),
            ),
          );
      });
    } catch (error) {
      if (mounted) setState(() => _error = error);
    } finally {
      if (mounted) setState(() => _isLoading = false);
    }
  }

  Future<void> _openTest(OnboardingTest test) async {
    final savedAnswers = _submissions[test.slug];
    final completed = await Navigator.of(context).push<bool>(
      MaterialPageRoute(
        builder: (_) => OnboardingTestRunner(
          test: test,
          initialAnswers: savedAnswers?.testVersion == test.version
              ? savedAnswers!.answers
              : const {},
        ),
      ),
    );
    if (!mounted || completed != true) return;
    await _load();
    if (mounted) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('Ответы сохранены.')));
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Знакомство с собой')),
    body: RefreshIndicator(
      onRefresh: _load,
      child: ListView(
        padding: const EdgeInsets.fromLTRB(18, 8, 18, 30),
        children: [
          Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              color: const Color(0xFFF2EAF8),
              borderRadius: BorderRadius.circular(22),
            ),
            child: const Text(
              'Тесты помогают лучше понять свои предпочтения. '
              'Это подсказки для размышления, а не диагноз и не решение за вас.',
              style: TextStyle(height: 1.5, fontWeight: FontWeight.w600),
            ),
          ),
          const SizedBox(height: 18),
          if (_error != null)
            _ErrorPanel(error: _error!, onRetry: _load)
          else if (_isLoading && _tests.isEmpty)
            const Padding(
              padding: EdgeInsets.only(top: 100),
              child: Center(child: CircularProgressIndicator()),
            )
          else if (_tests.isEmpty)
            const _EmptyCatalog()
          else
            ..._tests.map(_testCard),
        ],
      ),
    ),
  );

  Widget _testCard(OnboardingTest test) {
    final title = _localized(test.title);
    final description = _localized(test.description);
    final submission = _submissions[test.slug];
    final isCurrentVersion = submission?.testVersion == test.version;

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(18),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    title,
                    style: const TextStyle(
                      fontSize: 17,
                      fontWeight: FontWeight.w900,
                    ),
                  ),
                ),
                if (isCurrentVersion)
                  const Icon(
                    Icons.check_circle_rounded,
                    color: Color(0xFF72A987),
                  ),
              ],
            ),
            if (description.isNotEmpty) ...[
              const SizedBox(height: 7),
              Text(
                description,
                style: const TextStyle(
                  height: 1.4,
                  color: AppColors.secondaryText,
                ),
              ),
            ],
            const SizedBox(height: 10),
            Text(
              '${test.questions.length} вопросов · версия ${test.version}'
              '${isCurrentVersion ? ' · пройдено' : ''}',
              style: const TextStyle(
                fontSize: 12,
                color: AppColors.secondaryText,
              ),
            ),
            const SizedBox(height: 12),
            SizedBox(
              width: double.infinity,
              child: FilledButton(
                onPressed: test.questions.isEmpty
                    ? null
                    : () => _openTest(test),
                child: Text(isCurrentVersion ? 'Изменить ответы' : 'Начать'),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class OnboardingTestRunner extends ConsumerStatefulWidget {
  final OnboardingTest test;
  final Map<String, dynamic> initialAnswers;

  const OnboardingTestRunner({
    super.key,
    required this.test,
    required this.initialAnswers,
  });

  @override
  ConsumerState<OnboardingTestRunner> createState() =>
      _OnboardingTestRunnerState();
}

class _OnboardingTestRunnerState extends ConsumerState<OnboardingTestRunner> {
  final Map<String, dynamic> _answers = {};
  final Map<String, TextEditingController> _controllers = {};
  int _currentQuestion = 0;
  bool _isSubmitting = false;

  List<OnboardingQuestion> get _questions => widget.test.questions;

  @override
  void initState() {
    super.initState();
    _answers.addAll(widget.initialAnswers);
    for (final question in _questions) {
      if (question.type == 'text' || question.type == 'number') {
        final value = _answers[question.key];
        _controllers[question.key] = TextEditingController(
          text: value?.toString() ?? '',
        );
      }
    }
  }

  @override
  void dispose() {
    for (final controller in _controllers.values) {
      controller.dispose();
    }
    super.dispose();
  }

  bool _isAnswered(OnboardingQuestion question) {
    final answer = _answers[question.key];
    if (answer == null) return !question.required;

    switch (question.type) {
      case 'single_choice':
        return question.options.any((option) => option.value == answer);
      case 'multiple_choice':
        return answer is List &&
            (answer.isEmpty
                ? !question.required
                : answer.every(
                    (value) =>
                        question.options.any((option) => option.value == value),
                  ));
      case 'text':
        if (answer is! String) return false;
        if (question.required && answer.trim().isEmpty) return false;
        return answer.runes.length <= (question.maxLength ?? 2000);
      case 'scale':
        return _validNumber(question, answer, integral: true);
      case 'number':
        return _validNumber(question, answer);
      case 'boolean':
        return answer is bool;
      default:
        return false;
    }
  }

  bool _validNumber(
    OnboardingQuestion question,
    Object? answer, {
    bool integral = false,
  }) {
    final number = answer is num ? answer.toDouble() : null;
    if (number == null || !number.isFinite) return false;
    if (integral && number.truncateToDouble() != number) return false;
    final minimum = question.min ?? (integral ? 1 : double.negativeInfinity);
    final maximum = question.max ?? (integral ? 5 : double.infinity);
    return number >= minimum && number <= maximum;
  }

  void _next() {
    final question = _questions[_currentQuestion];
    if (!_isAnswered(question)) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            question.required
                ? 'Ответьте на вопрос, чтобы продолжить.'
                : 'Проверьте выбранный ответ.',
          ),
        ),
      );
      return;
    }
    setState(() => _currentQuestion++);
  }

  void _back() => setState(() => _currentQuestion--);

  Future<void> _submit() async {
    final question = _questions[_currentQuestion];
    if (!_isAnswered(question)) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Ответьте на обязательный вопрос.')),
      );
      return;
    }

    final answers = <String, dynamic>{
      for (final item in _questions)
        if (_answers.containsKey(item.key)) item.key: _answers[item.key],
    };
    for (final item in _questions) {
      if (item.type == 'text' && answers[item.key] is String) {
        final value = (answers[item.key] as String).trim();
        if (value.isEmpty) {
          answers.remove(item.key);
        } else {
          answers[item.key] = value;
        }
      } else if (item.type == 'multiple_choice' &&
          answers[item.key] is List &&
          (answers[item.key] as List).isEmpty) {
        answers.remove(item.key);
      }
    }

    if (answers.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Добавьте хотя бы один ответ.')),
      );
      return;
    }

    setState(() => _isSubmitting = true);
    try {
      await ref
          .read(onboardingApiProvider)
          .submitAnswers(
            testSlug: widget.test.slug,
            version: widget.test.version,
            answers: answers,
          );
      if (mounted) Navigator.pop(context, true);
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(apiErrorMessage(error))));
      }
    } finally {
      if (mounted) setState(() => _isSubmitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_questions.isEmpty) {
      return Scaffold(
        appBar: AppBar(title: Text(_localized(widget.test.title))),
        body: const Center(child: Text('В этом тесте пока нет вопросов.')),
      );
    }

    final question = _questions[_currentQuestion];
    final isLastQuestion = _currentQuestion == _questions.length - 1;
    final progress = (_currentQuestion + 1) / _questions.length;

    return Scaffold(
      appBar: AppBar(title: Text(_localized(widget.test.title))),
      body: SafeArea(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 8, 20, 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  LinearProgressIndicator(value: progress),
                  const SizedBox(height: 8),
                  Text(
                    'Вопрос ${_currentQuestion + 1} из ${_questions.length}',
                    style: const TextStyle(color: AppColors.secondaryText),
                  ),
                ],
              ),
            ),
            Expanded(
              child: SingleChildScrollView(
                padding: const EdgeInsets.fromLTRB(20, 8, 20, 20),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      _localized(question.prompt),
                      style: const TextStyle(
                        fontSize: 23,
                        fontWeight: FontWeight.w900,
                        height: 1.3,
                      ),
                    ),
                    if (question.required) ...[
                      const SizedBox(height: 6),
                      const Text(
                        'Обязательный вопрос',
                        style: TextStyle(
                          color: AppColors.secondaryText,
                          fontSize: 12,
                        ),
                      ),
                    ],
                    const SizedBox(height: 22),
                    _buildAnswerInput(question),
                  ],
                ),
              ),
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 10, 20, 16),
              child: Row(
                children: [
                  if (_currentQuestion > 0)
                    OutlinedButton(
                      onPressed: _isSubmitting ? null : _back,
                      child: const Text('Назад'),
                    ),
                  const Spacer(),
                  FilledButton(
                    onPressed: _isSubmitting
                        ? null
                        : isLastQuestion
                        ? _submit
                        : _next,
                    child: _isSubmitting
                        ? const SizedBox(
                            width: 20,
                            height: 20,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              color: Colors.white,
                            ),
                          )
                        : Text(isLastQuestion ? 'Сохранить ответы' : 'Далее'),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildAnswerInput(OnboardingQuestion question) {
    switch (question.type) {
      case 'single_choice':
        return RadioGroup<String>(
          groupValue: _answers[question.key] as String?,
          onChanged: (value) {
            if (value != null) {
              setState(() => _answers[question.key] = value);
            }
          },
          child: Column(
            children: question.options.map((option) {
              return Card(
                child: RadioListTile<String>(
                  value: option.value,
                  title: Text(_localized(option.label)),
                ),
              );
            }).toList(),
          ),
        );
      case 'multiple_choice':
        final current = _answers[question.key] is List
            ? List<String>.from(_answers[question.key] as List)
            : <String>[];
        return Column(
          children: question.options.map((option) {
            return Card(
              child: CheckboxListTile(
                value: current.contains(option.value),
                title: Text(_localized(option.label)),
                onChanged: (selected) {
                  setState(() {
                    final updated = List<String>.from(current);
                    if (selected == true) {
                      updated.add(option.value);
                    } else {
                      updated.remove(option.value);
                    }
                    _answers[question.key] = updated;
                  });
                },
              ),
            );
          }).toList(),
        );
      case 'text':
        return TextField(
          controller: _controllers[question.key],
          maxLength: question.maxLength ?? 2000,
          minLines: 3,
          maxLines: 6,
          decoration: const InputDecoration(
            hintText: 'Напишите свой ответ',
            alignLabelWithHint: true,
          ),
          onChanged: (value) => _answers[question.key] = value,
        );
      case 'scale':
        final minimum = (question.min ?? 1).ceilToDouble();
        final maximum = (question.max ?? 5).floorToDouble();
        if (minimum > maximum) {
          return const Text('Для этого вопроса задана некорректная шкала.');
        }
        final selected = _answers[question.key] as num?;
        final value = (selected?.toDouble() ?? minimum).clamp(minimum, maximum);
        if (minimum == maximum) {
          return ChoiceChip(
            label: Text(minimum.toInt().toString()),
            selected: selected?.toInt() == minimum.toInt(),
            onSelected: (_) =>
                setState(() => _answers[question.key] = minimum.toInt()),
          );
        }
        final divisions = (maximum - minimum).round().clamp(1, 100);
        return Column(
          children: [
            Text(
              value.toInt().toString(),
              style: const TextStyle(
                fontSize: 42,
                fontWeight: FontWeight.w900,
                color: AppColors.lavenderDark,
              ),
            ),
            Slider(
              value: value,
              min: minimum,
              max: maximum,
              divisions: divisions,
              label: value.toInt().toString(),
              onChanged: (next) =>
                  setState(() => _answers[question.key] = next.round()),
            ),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(minimum.toInt().toString()),
                Text(maximum.toInt().toString()),
              ],
            ),
          ],
        );
      case 'number':
        return TextField(
          controller: _controllers[question.key],
          keyboardType: const TextInputType.numberWithOptions(
            decimal: true,
            signed: true,
          ),
          decoration: const InputDecoration(hintText: 'Введите число'),
          onChanged: (value) {
            final parsed = double.tryParse(value);
            setState(() {
              if (parsed == null || !parsed.isFinite) {
                _answers.remove(question.key);
              } else {
                _answers[question.key] = parsed;
              }
            });
          },
        );
      case 'boolean':
        final selected = _answers[question.key] as bool?;
        return Wrap(
          spacing: 10,
          children: [
            ChoiceChip(
              label: const Text('Да'),
              selected: selected == true,
              onSelected: (_) => setState(() => _answers[question.key] = true),
            ),
            ChoiceChip(
              label: const Text('Нет'),
              selected: selected == false,
              onSelected: (_) => setState(() => _answers[question.key] = false),
            ),
          ],
        );
      default:
        return const Text('Этот тип вопроса пока не поддерживается.');
    }
  }
}

class _EmptyCatalog extends StatelessWidget {
  const _EmptyCatalog();

  @override
  Widget build(BuildContext context) => const Padding(
    padding: EdgeInsets.only(top: 55),
    child: Column(
      children: [
        Text('🌱', style: TextStyle(fontSize: 48)),
        SizedBox(height: 12),
        Text(
          'Тесты появятся позже',
          style: TextStyle(fontSize: 18, fontWeight: FontWeight.w900),
        ),
        SizedBox(height: 6),
        Text(
          'Пока нет опубликованных тестов. Загляните сюда позже.',
          textAlign: TextAlign.center,
        ),
      ],
    ),
  );
}

class _ErrorPanel extends StatelessWidget {
  final Object error;
  final VoidCallback onRetry;

  const _ErrorPanel({required this.error, required this.onRetry});

  @override
  Widget build(BuildContext context) => Column(
    children: [
      Text(apiErrorMessage(error), textAlign: TextAlign.center),
      const SizedBox(height: 10),
      FilledButton(onPressed: onRetry, child: const Text('Повторить')),
    ],
  );
}

String _localized(Map<String, String> translations) =>
    translations['ru'] ??
    translations['en'] ??
    (translations.values.isEmpty ? '' : translations.values.first);
