import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_error_message.dart';
import '../../../core/providers.dart';
import '../../../core/theme/app_theme.dart';
import '../data/planning_day_items.dart';

enum _PlanningItemType { task, event, importantDate }

enum _PlanningView { list, week, month }

class PlanningScreen extends ConsumerStatefulWidget {
  const PlanningScreen({super.key});

  @override
  ConsumerState<PlanningScreen> createState() => _PlanningScreenState();
}

class _PlanningScreenState extends ConsumerState<PlanningScreen> {
  List<Map<String, dynamic>> _tasks = [];
  List<Map<String, dynamic>> _events = [];
  List<Map<String, dynamic>> _dates = [];
  DateTime _selectedDate = DateTime.now();
  _PlanningView _view = _PlanningView.list;
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
      final api = ref.read(planningApiProvider);
      final userId = ref.read(currentUserIdProvider);
      final results = await Future.wait([
        api.listTasks(userId: userId),
        api.listEvents(userId: userId),
        api.listImportantDates(userId: userId),
      ]);
      if (!mounted) return;
      setState(() {
        _tasks = results[0];
        _events = results[1];
        _dates = results[2];
      });
    } catch (error) {
      if (mounted) setState(() => _error = error);
    } finally {
      if (mounted) setState(() => _isLoading = false);
    }
  }

  Future<void> _toggleTask(Map<String, dynamic> task) async {
    try {
      await ref
          .read(planningApiProvider)
          .setTaskDone(
            id: task['id'] as String,
            version: (task['version'] as num).toInt(),
            done: task['status'] != 'done',
          );
      await _reload();
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(apiErrorMessage(error))));
      }
    }
  }

  Future<void> _chooseCreateType() async {
    final type = await showModalBottomSheet<_PlanningItemType>(
      context: context,
      showDragHandle: true,
      builder: (context) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.checklist_rounded),
              title: const Text('Задача'),
              onTap: () => Navigator.pop(context, _PlanningItemType.task),
            ),
            ListTile(
              leading: const Icon(Icons.event_rounded),
              title: const Text('Событие календаря'),
              onTap: () => Navigator.pop(context, _PlanningItemType.event),
            ),
            ListTile(
              leading: const Icon(Icons.cake_outlined),
              title: const Text('Важная дата'),
              onTap: () =>
                  Navigator.pop(context, _PlanningItemType.importantDate),
            ),
          ],
        ),
      ),
    );
    if (type != null && mounted) await _showCreateDialog(type);
  }

  Future<void> _showCreateDialog(_PlanningItemType type) async {
    final titleController = TextEditingController();
    var scope = 'personal';
    var repeatsYearly = false;
    var selectedDate = DateTime.now();
    DateTime? taskDueDate;
    final title = switch (type) {
      _PlanningItemType.task => 'Новая задача',
      _PlanningItemType.event => 'Новое событие',
      _PlanningItemType.importantDate => 'Важная дата',
    };

    await showDialog<void>(
      context: context,
      builder: (dialogContext) => StatefulBuilder(
        builder: (_, setDialogState) => AlertDialog(
          title: Text(title),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(
                  controller: titleController,
                  autofocus: true,
                  maxLength: 200,
                  decoration: const InputDecoration(labelText: 'Название'),
                ),
                const SizedBox(height: 8),
                DropdownButtonFormField<String>(
                  initialValue: scope,
                  decoration: const InputDecoration(labelText: 'Видимость'),
                  items: const [
                    DropdownMenuItem(
                      value: 'personal',
                      child: Text('Только для меня'),
                    ),
                    DropdownMenuItem(value: 'couple', child: Text('Для пары')),
                  ],
                  onChanged: (value) {
                    if (value != null) setDialogState(() => scope = value);
                  },
                ),
                if (type == _PlanningItemType.task)
                  Row(
                    children: [
                      Expanded(
                        child: TextButton.icon(
                          onPressed: () async {
                            final date = await showDatePicker(
                              context: context,
                              initialDate: taskDueDate ?? DateTime.now(),
                              firstDate: DateTime(2000),
                              lastDate: DateTime(2100),
                            );
                            if (date != null) {
                              setDialogState(() => taskDueDate = date);
                            }
                          },
                          icon: const Icon(Icons.calendar_today_rounded),
                          label: Text(
                            taskDueDate == null
                                ? 'Добавить срок'
                                : 'Срок: ${_formatDate(taskDueDate!)}',
                          ),
                        ),
                      ),
                      if (taskDueDate != null)
                        IconButton(
                          tooltip: 'Убрать срок',
                          onPressed: () =>
                              setDialogState(() => taskDueDate = null),
                          icon: const Icon(Icons.close_rounded),
                        ),
                    ],
                  ),
                if (type != _PlanningItemType.task) ...[
                  const SizedBox(height: 8),
                  TextButton.icon(
                    onPressed: () async {
                      final date = await showDatePicker(
                        context: context,
                        initialDate: selectedDate,
                        firstDate: DateTime(2000),
                        lastDate: DateTime(2100),
                      );
                      if (date != null) {
                        setDialogState(() => selectedDate = date);
                      }
                    },
                    icon: const Icon(Icons.calendar_today_rounded),
                    label: Text(_formatDate(selectedDate)),
                  ),
                ],
                if (type == _PlanningItemType.importantDate)
                  CheckboxListTile(
                    value: repeatsYearly,
                    contentPadding: EdgeInsets.zero,
                    title: const Text('Повторять каждый год'),
                    onChanged: (value) =>
                        setDialogState(() => repeatsYearly = value ?? false),
                  ),
              ],
            ),
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
                  final api = ref.read(planningApiProvider);
                  switch (type) {
                    case _PlanningItemType.task:
                      await api.createTask(
                        title: titleController.text,
                        scope: scope,
                        dueAt: taskDueDate == null
                            ? null
                            : DateTime(
                                taskDueDate!.year,
                                taskDueDate!.month,
                                taskDueDate!.day,
                                23,
                                59,
                              ),
                      );
                      break;
                    case _PlanningItemType.event:
                      final startsAt = DateTime(
                        selectedDate.year,
                        selectedDate.month,
                        selectedDate.day,
                        18,
                      );
                      await api.createEvent(
                        title: titleController.text,
                        scope: scope,
                        startsAt: startsAt,
                        endsAt: startsAt.add(const Duration(hours: 1)),
                      );
                      break;
                    case _PlanningItemType.importantDate:
                      await api.createImportantDate(
                        title: titleController.text,
                        scope: scope,
                        date: selectedDate,
                        repeatsYearly: repeatsYearly,
                      );
                      break;
                  }
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
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Планирование'),
        actions: [
          IconButton(
            onPressed: _chooseCreateType,
            tooltip: 'Добавить',
            icon: const Icon(Icons.add_rounded),
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _reload,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(18, 8, 18, 28),
          children: [
            if (_error != null)
              _InlineError(error: _error!, onRetry: _reload)
            else if (_isLoading &&
                _tasks.isEmpty &&
                _events.isEmpty &&
                _dates.isEmpty)
              const Padding(
                padding: EdgeInsets.only(top: 100),
                child: Center(child: CircularProgressIndicator()),
              )
            else ...[
              _viewSelector(),
              const SizedBox(height: 12),
              if (_view == _PlanningView.list)
                ..._listViewItems()
              else
                ..._calendarViewItems(),
            ],
          ],
        ),
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: _chooseCreateType,
        child: const Icon(Icons.add_rounded),
      ),
    );
  }

  Widget _viewSelector() => SegmentedButton<_PlanningView>(
    segments: const [
      ButtonSegment(value: _PlanningView.list, label: Text('Список')),
      ButtonSegment(value: _PlanningView.week, label: Text('Неделя')),
      ButtonSegment(value: _PlanningView.month, label: Text('Месяц')),
    ],
    selected: {_view},
    onSelectionChanged: (selection) => setState(() => _view = selection.first),
  );

  List<Widget> _listViewItems() => [
    const _GroupTitle(title: 'Задачи'),
    if (_tasks.isEmpty)
      const _EmptyCard(
        icon: Icons.checklist_rounded,
        text: 'Пока нет задач. Добавьте первую маленькую цель.',
      )
    else
      ..._tasks.map(_buildTask),
    const SizedBox(height: 22),
    const _GroupTitle(title: 'События'),
    if (_events.isEmpty)
      const _EmptyCard(
        icon: Icons.event_rounded,
        text: 'Здесь появятся ваши общие и личные события.',
      )
    else
      ..._events.map(
        (event) => _PlanningCard(
          icon: Icons.event_rounded,
          title: event['title'] as String? ?? 'Событие',
          subtitle: _dateTimeLabel(event['starts_at'] as String?),
          scope: event['scope'] as String? ?? 'personal',
        ),
      ),
    const SizedBox(height: 22),
    const _GroupTitle(title: 'Важные даты'),
    if (_dates.isEmpty)
      const _EmptyCard(
        icon: Icons.cake_outlined,
        text: 'Сохраняйте даты, которые хочется помнить.',
      )
    else
      ..._dates.map(_buildImportantDate),
  ];

  List<Widget> _calendarViewItems() {
    final selectedItems = planningItemsForDate(
      date: _selectedDate,
      tasks: _tasks,
      events: _events,
      importantDates: _dates,
    );
    return [
      if (_view == _PlanningView.month)
        CalendarDatePicker(
          initialDate: _selectedDate,
          firstDate: DateTime(2000),
          lastDate: DateTime(2100),
          onDateChanged: (date) => setState(() => _selectedDate = date),
        )
      else
        _weekSelector(),
      const SizedBox(height: 12),
      _GroupTitle(title: _formatDate(_selectedDate)),
      if (selectedItems.tasks.isEmpty &&
          selectedItems.events.isEmpty &&
          selectedItems.importantDates.isEmpty)
        const _EmptyCard(
          icon: Icons.event_available_rounded,
          text: 'На эту дату ничего не запланировано.',
        ),
      ...selectedItems.tasks.map(_buildTask),
      ...selectedItems.events.map(
        (event) => _PlanningCard(
          icon: Icons.event_rounded,
          title: event['title'] as String? ?? 'Событие',
          subtitle: _dateTimeLabel(event['starts_at'] as String?),
          scope: event['scope'] as String? ?? 'personal',
        ),
      ),
      ...selectedItems.importantDates.map(_buildImportantDate),
    ];
  }

  Widget _weekSelector() {
    final monday = DateTime(
      _selectedDate.year,
      _selectedDate.month,
      _selectedDate.day,
    ).subtract(Duration(days: _selectedDate.weekday - 1));
    const weekdays = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];
    return Row(
      children: List<Widget>.generate(7, (index) {
        final date = monday.add(Duration(days: index));
        final selected = _sameDate(date, _selectedDate);
        return Expanded(
          child: InkWell(
            borderRadius: BorderRadius.circular(14),
            onTap: () => setState(() => _selectedDate = date),
            child: Container(
              padding: const EdgeInsets.symmetric(vertical: 10),
              decoration: BoxDecoration(
                color: selected
                    ? Theme.of(context).colorScheme.primaryContainer
                    : null,
                borderRadius: BorderRadius.circular(14),
              ),
              child: Column(
                children: [
                  Text(weekdays[index], style: const TextStyle(fontSize: 11)),
                  const SizedBox(height: 4),
                  Text(
                    '${date.day}',
                    style: const TextStyle(fontWeight: FontWeight.w800),
                  ),
                ],
              ),
            ),
          ),
        );
      }),
    );
  }

  Widget _buildImportantDate(Map<String, dynamic> date) => _PlanningCard(
    icon: Icons.cake_outlined,
    title: date['title'] as String? ?? 'Важная дата',
    subtitle:
        '${_formatDate(DateTime.parse(date['date'] as String))}'
        '${date['repeats_yearly'] == true ? ' · каждый год' : ''}',
    scope: date['scope'] as String? ?? 'personal',
  );

  Widget _buildTask(Map<String, dynamic> task) {
    final done = task['status'] == 'done';
    final dueAt = task['due_at'] as String?;
    return Card(
      child: ListTile(
        leading: IconButton(
          onPressed: () => _toggleTask(task),
          tooltip: done ? 'Вернуть в активные' : 'Отметить выполненной',
          icon: Icon(
            done ? Icons.check_circle_rounded : Icons.circle_outlined,
            color: done ? const Color(0xFF72A987) : AppColors.lavenderDark,
          ),
        ),
        title: Text(
          task['title'] as String? ?? 'Задача',
          style: TextStyle(
            decoration: done ? TextDecoration.lineThrough : null,
            fontWeight: FontWeight.w700,
          ),
        ),
        subtitle: Text(
          [
            if ((task['description'] as String? ?? '').isNotEmpty)
              task['description'] as String,
            if (dueAt != null) _dateTimeLabel(dueAt),
            _scopeLabel(task['scope'] as String? ?? 'personal'),
          ].join(' · '),
        ),
      ),
    );
  }
}

class _PlanningCard extends StatelessWidget {
  final IconData icon;
  final String title;
  final String subtitle;
  final String scope;

  const _PlanningCard({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.scope,
  });

  @override
  Widget build(BuildContext context) => Card(
    child: ListTile(
      leading: Icon(icon, color: AppColors.lavenderDark),
      title: Text(title, style: const TextStyle(fontWeight: FontWeight.w700)),
      subtitle: Text('$subtitle · ${_scopeLabel(scope)}'),
    ),
  );
}

class _GroupTitle extends StatelessWidget {
  final String title;

  const _GroupTitle({required this.title});

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(bottom: 10),
    child: Text(
      title,
      style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w900),
    ),
  );
}

class _EmptyCard extends StatelessWidget {
  final IconData icon;
  final String text;

  const _EmptyCard({required this.icon, required this.text});

  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.all(18),
    decoration: BoxDecoration(
      color: const Color(0xFFF9F6FB),
      borderRadius: BorderRadius.circular(18),
    ),
    child: Row(
      children: [
        Icon(icon, color: AppColors.lavenderDark),
        const SizedBox(width: 12),
        Expanded(child: Text(text, style: const TextStyle(height: 1.4))),
      ],
    ),
  );
}

class _InlineError extends StatelessWidget {
  final Object error;
  final VoidCallback onRetry;

  const _InlineError({required this.error, required this.onRetry});

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(bottom: 16),
    child: Row(
      children: [
        Expanded(child: Text(apiErrorMessage(error))),
        IconButton(onPressed: onRetry, icon: const Icon(Icons.refresh_rounded)),
      ],
    ),
  );
}

String _scopeLabel(String scope) => scope == 'couple' ? 'Для пары' : 'Личное';

String _formatDate(DateTime date) =>
    '${date.day.toString().padLeft(2, '0')}.'
    '${date.month.toString().padLeft(2, '0')}.'
    '${date.year}';

bool _sameDate(DateTime first, DateTime second) =>
    first.year == second.year &&
    first.month == second.month &&
    first.day == second.day;

String _dateTimeLabel(String? value) {
  if (value == null) return '';
  final date = DateTime.parse(value).toLocal();
  return '${_formatDate(date)} · '
      '${date.hour.toString().padLeft(2, '0')}:'
      '${date.minute.toString().padLeft(2, '0')}';
}
