import 'package:flutter_test/flutter_test.dart';
import 'package:lovetea/features/habits/data/habits_api.dart';

void main() {
  test('parses server habit totals and participant streaks', () {
    final stats = HabitStats.fromJson({
      'habit_id': 'habit-1',
      'from': '2026-10-22',
      'to': '2026-11-21',
      'target_per_week': 3,
      'total_completions': 12,
      'members': [
        {
          'user_id': 'user-1',
          'completion_count': 7,
          'successful_weeks': 2,
          'current_streak_weeks': 1,
        },
        {
          'user_id': 'user-2',
          'completion_count': 5,
          'successful_weeks': 1,
          'current_streak_weeks': 3,
        },
      ],
    });

    expect(stats.totalCompletions, 12);
    expect(stats.successfulWeeks, 3);
    expect(stats.bestCurrentStreakWeeks, 3);
    expect(stats.targetPerWeek, 3);
  });
}
