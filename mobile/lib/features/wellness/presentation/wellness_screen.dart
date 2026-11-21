import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_error_message.dart';
import '../../../core/providers.dart';
import '../../../core/theme/app_theme.dart';
import '../../planning/data/planning_api.dart';
import '../data/wellness_week_summary.dart';

class WellnessScreen extends ConsumerStatefulWidget {
  const WellnessScreen({super.key});

  @override
  ConsumerState<WellnessScreen> createState() => _WellnessScreenState();
}

class _WellnessScreenState extends ConsumerState<WellnessScreen> {
  static const _moods = [
    ('😊', 'радостно'),
    ('🙂', 'хорошо'),
    ('😐', 'нейтрально'),
    ('😔', 'грустно'),
    ('😣', 'тяжело'),
  ];

  final _noteController = TextEditingController();
  List<Map<String, dynamic>> _entries = [];
  List<Map<String, dynamic>> _sleepEntries = [];
  String _selectedMood = 'хорошо';
  bool _shareWithCouple = false;
  bool _isLoading = true;
  bool _isSaving = false;
  Object? _error;

  @override
  void initState() {
    super.initState();
    _reload();
  }

  @override
  void dispose() {
    _noteController.dispose();
    super.dispose();
  }

  Future<void> _reload() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });
    try {
      final api = ref.read(wellnessApiProvider);
      final results = await Future.wait([api.listMood(), api.listSleep()]);
      if (!mounted) return;
      final entries = results[0];
      final today = PlanningApi.formatDate(DateTime.now());
      final myEntry = entries.cast<Map<String, dynamic>?>().firstWhere(
        (entry) => entry?['entry_date'] == today && entry?['is_mine'] == true,
        orElse: () => null,
      );
      setState(() {
        _entries = entries;
        _sleepEntries = results[1];
        if (myEntry != null) {
          _selectedMood = myEntry['mood'] as String? ?? _selectedMood;
          _noteController.text = myEntry['note'] as String? ?? '';
          _shareWithCouple = myEntry['shared_with_couple'] as bool? ?? false;
        }
      });
    } catch (error) {
      if (mounted) setState(() => _error = error);
    } finally {
      if (mounted) setState(() => _isLoading = false);
    }
  }

  Future<void> _saveSleep() async {
    var sleepTime = const TimeOfDay(hour: 23, minute: 0);
    var wakeTime = const TimeOfDay(hour: 7, minute: 0);
    var quality = 3;
    var shareWithCouple = false;
    final now = DateTime.now();

    await showDialog<void>(
      context: context,
      builder: (dialogContext) => StatefulBuilder(
        builder: (_, setDialogState) => AlertDialog(
          title: const Text('Записать сон'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              _TimeChoice(
                title: 'Время отхода ко сну',
                time: sleepTime,
                onPressed: () async {
                  final selected = await showTimePicker(
                    context: dialogContext,
                    initialTime: sleepTime,
                  );
                  if (selected != null) {
                    setDialogState(() => sleepTime = selected);
                  }
                },
              ),
              _TimeChoice(
                title: 'Время пробуждения',
                time: wakeTime,
                onPressed: () async {
                  final selected = await showTimePicker(
                    context: dialogContext,
                    initialTime: wakeTime,
                  );
                  if (selected != null) {
                    setDialogState(() => wakeTime = selected);
                  }
                },
              ),
              const SizedBox(height: 8),
              Text('Качество сна: $quality из 5'),
              Slider(
                value: quality.toDouble(),
                min: 1,
                max: 5,
                divisions: 4,
                label: '$quality',
                onChanged: (value) =>
                    setDialogState(() => quality = value.round()),
              ),
              SwitchListTile(
                contentPadding: EdgeInsets.zero,
                title: const Text('Поделиться с партнёром'),
                value: shareWithCouple,
                onChanged: (value) =>
                    setDialogState(() => shareWithCouple = value),
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
                final wakeAt = DateTime(
                  now.year,
                  now.month,
                  now.day,
                  wakeTime.hour,
                  wakeTime.minute,
                );
                var sleepDay = DateTime(now.year, now.month, now.day);
                final sleepAt = DateTime(
                  sleepDay.year,
                  sleepDay.month,
                  sleepDay.day,
                  sleepTime.hour,
                  sleepTime.minute,
                );
                if (!sleepAt.isBefore(wakeAt)) {
                  sleepDay = sleepDay.subtract(const Duration(days: 1));
                }
                final adjustedSleepAt = DateTime(
                  sleepDay.year,
                  sleepDay.month,
                  sleepDay.day,
                  sleepTime.hour,
                  sleepTime.minute,
                );
                final today = PlanningApi.formatDate(now);
                final existing = _sleepEntries.where(
                  (entry) =>
                      entry['entry_date'] == today && entry['is_mine'] == true,
                );
                final version = existing.isEmpty
                    ? null
                    : (existing.first['version'] as num).toInt();
                try {
                  await ref
                      .read(wellnessApiProvider)
                      .saveSleep(
                        sleptAt: adjustedSleepAt,
                        wokeAt: wakeAt,
                        quality: quality,
                        shareWithCouple: shareWithCouple,
                        version: version,
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
  }

  Future<void> _save() async {
    setState(() => _isSaving = true);
    try {
      final today = PlanningApi.formatDate(DateTime.now());
      final currentEntry = _entries.where(
        (entry) => entry['entry_date'] == today && entry['is_mine'] == true,
      );
      final version = currentEntry.isEmpty
          ? null
          : (currentEntry.first['version'] as num).toInt();
      await ref
          .read(wellnessApiProvider)
          .saveMood(
            mood: _selectedMood,
            note: _noteController.text,
            shareWithCouple: _shareWithCouple,
            version: version,
          );
      await _reload();
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Отметка настроения сохранена.')),
        );
      }
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(apiErrorMessage(error))));
      }
    } finally {
      if (mounted) setState(() => _isSaving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Настроение')),
      body: RefreshIndicator(
        onRefresh: _reload,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(18, 8, 18, 30),
          children: [
            Container(
              padding: const EdgeInsets.all(20),
              decoration: BoxDecoration(
                color: const Color(0xFFF3EBF8),
                borderRadius: BorderRadius.circular(22),
              ),
              child: const Text(
                'Как ты сегодня? Твои заметки видишь только ты. '
                'Настроением можно поделиться отдельно.',
                style: TextStyle(height: 1.5, fontWeight: FontWeight.w600),
              ),
            ),
            const SizedBox(height: 20),
            const Text(
              'Выбери настроение',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w900),
            ),
            const SizedBox(height: 12),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: _moods.map((mood) {
                final selected = _selectedMood == mood.$2;
                return ChoiceChip(
                  selected: selected,
                  label: Text('${mood.$1}  ${mood.$2}'),
                  onSelected: (_) => setState(() => _selectedMood = mood.$2),
                );
              }).toList(),
            ),
            const SizedBox(height: 18),
            TextField(
              controller: _noteController,
              maxLines: 4,
              maxLength: 2000,
              decoration: const InputDecoration(
                labelText: 'Личная заметка',
                hintText: 'Что повлияло на твоё состояние?',
                alignLabelWithHint: true,
              ),
            ),
            SwitchListTile(
              contentPadding: EdgeInsets.zero,
              value: _shareWithCouple,
              onChanged: (value) => setState(() => _shareWithCouple = value),
              title: const Text('Поделиться настроением с партнёром'),
              subtitle: const Text('Заметка останется личной'),
            ),
            const SizedBox(height: 8),
            SizedBox(
              height: 52,
              child: FilledButton(
                onPressed: _isSaving ? null : _save,
                child: _isSaving
                    ? const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                    : const Text('Сохранить отметку'),
              ),
            ),
            const SizedBox(height: 26),
            Row(
              children: [
                const Expanded(
                  child: Text(
                    'Сон',
                    style: TextStyle(fontSize: 18, fontWeight: FontWeight.w900),
                  ),
                ),
                TextButton.icon(
                  onPressed: _saveSleep,
                  icon: const Icon(Icons.bedtime_outlined),
                  label: const Text('Добавить'),
                ),
              ],
            ),
            if (_sleepEntries.isEmpty)
              const Text('Записей сна пока нет.')
            else
              ..._sleepEntries.map(_sleepTile),
            const SizedBox(height: 26),
            const Text(
              'Последние 7 дней',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w900),
            ),
            const SizedBox(height: 10),
            _weeklySummary(),
            const SizedBox(height: 26),
            const Text(
              'Недавние отметки',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w900),
            ),
            const SizedBox(height: 10),
            if (_error != null)
              Text(apiErrorMessage(_error!))
            else if (_isLoading && _entries.isEmpty)
              const Center(child: CircularProgressIndicator())
            else if (_entries.isEmpty)
              const Text('Записей пока нет.')
            else
              ..._entries.map(_entryTile),
          ],
        ),
      ),
    );
  }

  Widget _weeklySummary() {
    final days = buildWellnessWeekSummary(
      today: DateTime.now(),
      moodEntries: _entries,
      sleepEntries: _sleepEntries,
    );
    return Card(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 16),
        child: Column(
          children: [
            Row(
              children: days.map((day) {
                final mood = _moods.where((option) => option.$2 == day.mood);
                final emoji = mood.isEmpty ? '·' : mood.first.$1;
                return Expanded(
                  child: Column(
                    children: [
                      Text(
                        '${day.date.day}',
                        style: TextStyle(
                          color: Colors.grey.shade600,
                          fontSize: 11,
                        ),
                      ),
                      const SizedBox(height: 6),
                      Text(emoji, style: const TextStyle(fontSize: 20)),
                      const SizedBox(height: 6),
                      Text(
                        day.sleepQuality == null
                            ? '—'
                            : '${day.sleepQuality}/5',
                        style: const TextStyle(
                          fontSize: 10,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ],
                  ),
                );
              }).toList(),
            ),
            const SizedBox(height: 12),
            const Text(
              'Настроение и качество сна · только твои записи',
              textAlign: TextAlign.center,
              style: TextStyle(fontSize: 12, color: AppColors.secondaryText),
            ),
          ],
        ),
      ),
    );
  }

  Widget _sleepTile(Map<String, dynamic> entry) {
    final isMine = entry['is_mine'] == true;
    final start = DateTime.parse(entry['sleep_started_at'] as String).toLocal();
    final wake = DateTime.parse(entry['woke_at'] as String).toLocal();
    return Card(
      child: ListTile(
        leading: Icon(
          Icons.bedtime_rounded,
          color: isMine ? AppColors.lavenderDark : const Color(0xFF72A987),
        ),
        title: Text(
          '${_formatDate(entry['entry_date'] as String?)} · '
          '${_timeLabel(start)} – ${_timeLabel(wake)}',
          style: const TextStyle(fontWeight: FontWeight.w800),
        ),
        subtitle: Text(
          isMine
              ? 'Качество: ${entry['quality']}/5'
              : 'Сон партнёра · ${entry['quality']}/5',
        ),
        trailing: isMine ? null : const Icon(Icons.people_alt_outlined),
      ),
    );
  }

  Widget _entryTile(Map<String, dynamic> entry) {
    final isMine = entry['is_mine'] == true;
    final mood = entry['mood'] as String? ?? 'настроение';
    final matchingMoods = _moods.where((option) => option.$2 == mood).toList();
    final emoji = matchingMoods.isEmpty ? null : matchingMoods.first.$1;
    return Card(
      child: ListTile(
        leading: Text(emoji ?? '💜', style: const TextStyle(fontSize: 26)),
        title: Text(
          '${_formatDate(entry['entry_date'] as String?)} · $mood',
          style: const TextStyle(fontWeight: FontWeight.w800),
        ),
        subtitle: Text(
          isMine ? (entry['note'] as String? ?? '') : 'Отметка партнёра',
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
        ),
        trailing: isMine
            ? null
            : const Icon(Icons.people_alt_outlined, size: 20),
      ),
    );
  }
}

class _TimeChoice extends StatelessWidget {
  final String title;
  final TimeOfDay time;
  final VoidCallback onPressed;

  const _TimeChoice({
    required this.title,
    required this.time,
    required this.onPressed,
  });

  @override
  Widget build(BuildContext context) => ListTile(
    contentPadding: EdgeInsets.zero,
    title: Text(title),
    trailing: TextButton(
      onPressed: onPressed,
      child: Text(time.format(context)),
    ),
  );
}

String _formatDate(String? value) {
  if (value == null) return '';
  final date = DateTime.parse(value);
  return '${date.day.toString().padLeft(2, '0')}.'
      '${date.month.toString().padLeft(2, '0')}';
}

String _timeLabel(DateTime date) =>
    '${date.hour.toString().padLeft(2, '0')}:'
    '${date.minute.toString().padLeft(2, '0')}';
