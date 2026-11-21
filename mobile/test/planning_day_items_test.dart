import 'package:flutter_test/flutter_test.dart';
import 'package:lovetea/features/planning/data/planning_day_items.dart';

void main() {
  test('matches due tasks, events, and yearly dates on the selected day', () {
    final items = planningItemsForDate(
      date: DateTime(2026, 11, 21),
      tasks: [
        {'id': 'due', 'due_at': '2026-11-21T18:00:00Z', 'status': 'open'},
        {'id': 'without-date', 'due_at': null, 'status': 'open'},
        {
          'id': 'cancelled',
          'due_at': '2026-11-21T12:00:00Z',
          'status': 'cancelled',
        },
      ],
      events: [
        {'id': 'event', 'starts_at': '2026-11-21T10:00:00Z'},
        {'id': 'other-day', 'starts_at': '2026-11-22T10:00:00Z'},
      ],
      importantDates: [
        {'id': 'annual', 'date': '2020-11-21', 'repeats_yearly': true},
        {'id': 'single', 'date': '2026-11-21', 'repeats_yearly': false},
        {'id': 'other-day', 'date': '2026-11-22', 'repeats_yearly': false},
      ],
    );

    expect(items.tasks.map((item) => item['id']), ['due']);
    expect(items.events.map((item) => item['id']), ['event']);
    expect(items.importantDates.map((item) => item['id']), [
      'annual',
      'single',
    ]);
  });
}
