class HomeUpcomingItem {
  final String title;
  final DateTime date;
  final String kind;

  const HomeUpcomingItem({
    required this.title,
    required this.date,
    required this.kind,
  });
}

List<HomeUpcomingItem> buildHomeUpcomingItems({
  required DateTime today,
  required List<Map<String, dynamic>> tasks,
  required List<Map<String, dynamic>> events,
  required List<Map<String, dynamic>> importantDates,
  int limit = 3,
}) {
  final startOfToday = DateTime(today.year, today.month, today.day);
  final items = <HomeUpcomingItem>[];

  for (final task in tasks) {
    final dueAt = task['due_at'];
    if (dueAt is! String ||
        task['status'] == 'done' ||
        task['status'] == 'cancelled') {
      continue;
    }
    final date = DateTime.tryParse(dueAt)?.toLocal();
    if (date != null) {
      items.add(
        HomeUpcomingItem(
          title: task['title'] as String? ?? 'Задача',
          date: date,
          kind: 'task',
        ),
      );
    }
  }

  for (final event in events) {
    final startsAt = event['starts_at'];
    if (startsAt is! String) continue;
    final date = DateTime.tryParse(startsAt)?.toLocal();
    if (date != null && !date.isBefore(startOfToday)) {
      items.add(
        HomeUpcomingItem(
          title: event['title'] as String? ?? 'Событие',
          date: date,
          kind: 'event',
        ),
      );
    }
  }

  for (final importantDate in importantDates) {
    final rawDate = importantDate['date'];
    if (rawDate is! String) continue;
    final parsed = DateTime.tryParse(rawDate);
    if (parsed == null) continue;

    final storedDate = DateTime(parsed.year, parsed.month, parsed.day);
    var date = DateTime(startOfToday.year, parsed.month, parsed.day);
    if (importantDate['repeats_yearly'] == true) {
      if (date.isBefore(startOfToday)) {
        date = DateTime(startOfToday.year + 1, parsed.month, parsed.day);
      }
    } else {
      date = storedDate;
      if (date.isBefore(startOfToday)) continue;
    }
    items.add(
      HomeUpcomingItem(
        title: importantDate['title'] as String? ?? 'Важная дата',
        date: date,
        kind: 'importantDate',
      ),
    );
  }

  items.sort((first, second) => first.date.compareTo(second.date));
  return items.take(limit).toList();
}
