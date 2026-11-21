import 'package:flutter_test/flutter_test.dart';
import 'package:lovetea/features/home/data/home_upcoming.dart';

void main() {
  test('sorts next plans and skips completed and expired items', () {
    final items = buildHomeUpcomingItems(
      today: DateTime(2026, 11, 21),
      tasks: [
        {
          'title': 'Задача завтра',
          'due_at': '2026-11-22T12:00:00Z',
          'status': 'open',
        },
        {'title': 'Готово', 'due_at': '2026-11-21T12:00:00Z', 'status': 'done'},
      ],
      events: [
        {'title': 'Встреча', 'starts_at': '2026-11-23T12:00:00Z'},
      ],
      importantDates: [
        {'title': 'Сегодня', 'date': '2020-11-21', 'repeats_yearly': true},
        {
          'title': 'Прошлая дата',
          'date': '2020-11-20',
          'repeats_yearly': false,
        },
      ],
    );

    expect(items.map((item) => item.title), [
      'Сегодня',
      'Задача завтра',
      'Встреча',
    ]);
    expect(items.first.date, DateTime(2026, 11, 21));
  });
}
