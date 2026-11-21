class PlanningDayItems {
  final List<Map<String, dynamic>> tasks;
  final List<Map<String, dynamic>> events;
  final List<Map<String, dynamic>> importantDates;

  const PlanningDayItems({
    required this.tasks,
    required this.events,
    required this.importantDates,
  });
}

PlanningDayItems planningItemsForDate({
  required DateTime date,
  required List<Map<String, dynamic>> tasks,
  required List<Map<String, dynamic>> events,
  required List<Map<String, dynamic>> importantDates,
}) {
  final selectedDate = _dateKey(date);
  return PlanningDayItems(
    tasks: tasks.where((task) {
      final dueAt = task['due_at'];
      if (dueAt is! String || task['status'] == 'cancelled') return false;
      final dueDate = DateTime.tryParse(dueAt)?.toLocal();
      return dueDate != null && _dateKey(dueDate) == selectedDate;
    }).toList(),
    events: events.where((event) {
      final startsAt = event['starts_at'];
      if (startsAt is! String) return false;
      final startDate = DateTime.tryParse(startsAt)?.toLocal();
      return startDate != null && _dateKey(startDate) == selectedDate;
    }).toList(),
    importantDates: importantDates.where((item) {
      final rawDate = item['date'];
      if (rawDate is! String || rawDate.length < 10) return false;
      final itemDate = DateTime.tryParse(rawDate);
      if (itemDate == null) return false;
      if (item['repeats_yearly'] == true) {
        return itemDate.month == date.month && itemDate.day == date.day;
      }
      return _dateKey(itemDate) == selectedDate;
    }).toList(),
  );
}

String _dateKey(DateTime date) =>
    '${date.year.toString().padLeft(4, '0')}-'
    '${date.month.toString().padLeft(2, '0')}-'
    '${date.day.toString().padLeft(2, '0')}';
