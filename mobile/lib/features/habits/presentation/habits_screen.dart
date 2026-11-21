import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_error_message.dart';
import '../../../core/providers.dart';
import '../../../core/theme/app_theme.dart';

class HabitsScreen extends ConsumerStatefulWidget {
  const HabitsScreen({super.key});

  @override
  ConsumerState<HabitsScreen> createState() => _HabitsScreenState();
}

class _HabitsScreenState extends ConsumerState<HabitsScreen> {
  List<Map<String, dynamic>> _habits = [];
  final Set<String> _completedToday = {};
  Object? _error;
  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _reload();
  }

  Future<void> _reload() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });
    try {
      final api = ref.read(habitsApiProvider);
      final habits = await api.listHabits();
      final completions = await Future.wait(
        habits.map((habit) => api.listCompletions(habit['id'] as String)),
      );
      final today = _dateOnly(DateTime.now());
      final completed = <String>{};
      for (var index = 0; index < habits.length; index++) {
        final hasCompleted = completions[index].any(
          (entry) => entry['completed_on'] == today && entry['is_mine'] == true,
        );
        if (hasCompleted) completed.add(habits[index]['id'] as String);
      }
      if (!mounted) return;
      setState(() {
        _habits = habits;
        _completedToday
          ..clear()
          ..addAll(completed);
      });
    } catch (error) {
      if (mounted) setState(() => _error = error);
    } finally {
      if (mounted) setState(() => _isLoading = false);
    }
  }

  Future<void> _completeToday(String id) async {
    try {
      await ref.read(habitsApiProvider).completeToday(id);
      await _reload();
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(apiErrorMessage(error))));
      }
    }
  }

  Future<void> _showStats(Map<String, dynamic> habit) async {
    try {
      final stats = await ref
          .read(habitsApiProvider)
          .getStats(habit['id'] as String);
      if (!mounted) return;
      await showDialog<void>(
        context: context,
        builder: (context) => AlertDialog(
          title: Text(habit['title'] as String? ?? 'Статистика привычки'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('Выполнений за 30 дней: ${stats.totalCompletions}'),
              const SizedBox(height: 8),
              Text('Успешных недель у участников: ${stats.successfulWeeks}'),
              const SizedBox(height: 8),
              Text(
                'Лучшая текущая серия: '
                '${stats.bestCurrentStreakWeeks} нед.',
              ),
              const SizedBox(height: 12),
              Text(
                'Цель: ${stats.targetPerWeek} отметок в неделю. '
                'Для серии неделя должна быть завершена.',
                style: TextStyle(color: Colors.grey.shade700, fontSize: 12),
              ),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('Закрыть'),
            ),
          ],
        ),
      );
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(apiErrorMessage(error))));
      }
    }
  }

  Future<void> _createHabit() async {
    final titleController = TextEditingController();
    var scope = 'personal';
    var target = 3;
    await showDialog<void>(
      context: context,
      builder: (dialogContext) => StatefulBuilder(
        builder: (_, setDialogState) => AlertDialog(
          title: const Text('Новая привычка'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: titleController,
                autofocus: true,
                maxLength: 200,
                decoration: const InputDecoration(
                  labelText: 'Например, вечерняя прогулка',
                ),
              ),
              DropdownButtonFormField<String>(
                initialValue: scope,
                decoration: const InputDecoration(labelText: 'Видимость'),
                items: const [
                  DropdownMenuItem(value: 'personal', child: Text('Личная')),
                  DropdownMenuItem(value: 'couple', child: Text('Совместная')),
                ],
                onChanged: (value) {
                  if (value != null) setDialogState(() => scope = value);
                },
              ),
              const SizedBox(height: 12),
              DropdownButtonFormField<int>(
                initialValue: target,
                decoration: const InputDecoration(labelText: 'Цель в неделю'),
                items: List.generate(
                  7,
                  (index) => DropdownMenuItem(
                    value: index + 1,
                    child: Text('${index + 1} ${index == 0 ? 'раз' : 'раза'}'),
                  ),
                ),
                onChanged: (value) {
                  if (value != null) setDialogState(() => target = value);
                },
              ),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(dialogContext),
              child: const Text('Отмена'),
            ),
            FilledButton(
              onPressed: () async {
                if (titleController.text.trim().isEmpty) return;
                try {
                  await ref
                      .read(habitsApiProvider)
                      .createHabit(
                        title: titleController.text,
                        scope: scope,
                        targetPerWeek: target,
                      );
                  if (dialogContext.mounted) Navigator.pop(dialogContext);
                  await _reload();
                } catch (error) {
                  if (mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(content: Text(apiErrorMessage(error))),
                    );
                  }
                }
              },
              child: const Text('Сохранить'),
            ),
          ],
        ),
      ),
    );
    titleController.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(
      title: const Text('Привычки'),
      actions: [
        IconButton(
          onPressed: _createHabit,
          tooltip: 'Добавить привычку',
          icon: const Icon(Icons.add_rounded),
        ),
      ],
    ),
    body: RefreshIndicator(
      onRefresh: _reload,
      child: ListView(
        padding: const EdgeInsets.fromLTRB(18, 8, 18, 30),
        children: [
          Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              color: const Color(0xFFEAF5EF),
              borderRadius: BorderRadius.circular(22),
            ),
            child: const Text(
              'Небольшие привычки помогают заботиться друг о друге без давления.',
              style: TextStyle(height: 1.5, fontWeight: FontWeight.w600),
            ),
          ),
          const SizedBox(height: 18),
          if (_error != null)
            _ErrorMessage(error: _error!, onRetry: _reload)
          else if (_isLoading && _habits.isEmpty)
            const Padding(
              padding: EdgeInsets.only(top: 100),
              child: Center(child: CircularProgressIndicator()),
            )
          else if (_habits.isEmpty)
            const _EmptyHabits()
          else
            ..._habits.map(_habitTile),
        ],
      ),
    ),
    floatingActionButton: FloatingActionButton(
      onPressed: _createHabit,
      child: const Icon(Icons.add_rounded),
    ),
  );

  Widget _habitTile(Map<String, dynamic> habit) {
    final id = habit['id'] as String;
    final completed = _completedToday.contains(id);
    final target = (habit['target_per_week'] as num?)?.toInt() ?? 1;
    final description = habit['description'] as String? ?? '';
    return Card(
      margin: const EdgeInsets.only(bottom: 10),
      child: ListTile(
        leading: Icon(
          completed ? Icons.check_circle_rounded : Icons.spa_outlined,
          color: completed ? const Color(0xFF72A987) : AppColors.lavenderDark,
          size: 30,
        ),
        title: Text(
          habit['title'] as String? ?? 'Привычка',
          style: const TextStyle(fontWeight: FontWeight.w800),
        ),
        subtitle: Text(
          [
            if (description.isNotEmpty) description,
            'Цель: $target в неделю',
            habit['scope'] == 'couple' ? 'Совместная' : 'Личная',
          ].join(' · '),
        ),
        trailing: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            IconButton(
              onPressed: () => _showStats(habit),
              tooltip: 'Статистика и серии',
              icon: const Icon(Icons.insights_rounded),
            ),
            IconButton(
              onPressed: completed ? null : () => _completeToday(id),
              tooltip: completed ? 'Уже отмечено сегодня' : 'Отметить сегодня',
              icon: Icon(
                completed ? Icons.done_all_rounded : Icons.add_task_rounded,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _EmptyHabits extends StatelessWidget {
  const _EmptyHabits();

  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.all(24),
    decoration: BoxDecoration(
      color: AppColors.white,
      borderRadius: BorderRadius.circular(22),
    ),
    child: const Column(
      children: [
        Text('🌱', style: TextStyle(fontSize: 44)),
        SizedBox(height: 12),
        Text(
          'Начните с маленького шага',
          style: TextStyle(fontSize: 18, fontWeight: FontWeight.w900),
        ),
        SizedBox(height: 6),
        Text(
          'Создайте личную или совместную привычку.',
          textAlign: TextAlign.center,
        ),
      ],
    ),
  );
}

class _ErrorMessage extends StatelessWidget {
  final Object error;
  final VoidCallback onRetry;

  const _ErrorMessage({required this.error, required this.onRetry});

  @override
  Widget build(BuildContext context) => Column(
    children: [
      Text(apiErrorMessage(error), textAlign: TextAlign.center),
      const SizedBox(height: 10),
      FilledButton(onPressed: onRetry, child: const Text('Повторить')),
    ],
  );
}

String _dateOnly(DateTime date) =>
    '${date.year.toString().padLeft(4, '0')}-'
    '${date.month.toString().padLeft(2, '0')}-'
    '${date.day.toString().padLeft(2, '0')}';
