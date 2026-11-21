import 'package:flutter_test/flutter_test.dart';
import 'package:lovetea/features/home/data/home_api.dart';
import 'package:lovetea/features/wellness/data/wellness_week_summary.dart';

void main() {
  test('parses a home snapshot with an active couple', () {
    final snapshot = HomeSnapshot.fromJson({
      'profile': {
        'user_id': 'user-1',
        'email': 'kitty@example.com',
        'display_name': 'Kitty',
      },
      'couple': {
        'status': 'active',
        'members': [
          {'email': 'kitty@example.com', 'nickname': 'Мурка'},
          {'email': 'wolf@example.com', 'nickname': 'Волчонок'},
        ],
      },
      'pet': {
        'couple_id': 'couple-1',
        'appearance': {'color': 'lavender'},
        'version': 2,
        'updated_at': '2026-11-20T12:00:00Z',
      },
    });

    expect(snapshot.displayName, 'Kitty');
    expect(snapshot.coupleStatus, 'active');
    expect(snapshot.members, hasLength(2));
    expect(snapshot.members.last.email, 'wolf@example.com');
    expect(snapshot.pet?.appearance['color'], 'lavender');
    expect(snapshot.pet?.version, 2);
  });

  test('parses a home snapshot when no couple exists', () {
    final snapshot = HomeSnapshot.fromJson({
      'profile': {
        'user_id': 'user-2',
        'email': 'user@example.com',
        'display_name': null,
      },
      'couple': null,
      'pet': null,
    });

    expect(snapshot.coupleStatus, isNull);
    expect(snapshot.members, isEmpty);
    expect(snapshot.userId, 'user-2');
  });

  test(
    'weekly wellness summary includes only the user’s shared-safe records',
    () {
      final days = buildWellnessWeekSummary(
        today: DateTime(2026, 11, 21),
        moodEntries: [
          {'entry_date': '2026-11-20', 'mood': 'радостно', 'is_mine': true},
          {'entry_date': '2026-11-21', 'mood': 'грустно', 'is_mine': false},
        ],
        sleepEntries: [
          {'entry_date': '2026-11-20', 'quality': 4, 'is_mine': true},
          {'entry_date': '2026-11-21', 'quality': 1, 'is_mine': false},
        ],
      );

      expect(days, hasLength(7));
      expect(days.first.date, DateTime(2026, 11, 15));
      expect(days[5].mood, 'радостно');
      expect(days[5].sleepQuality, 4);
      expect(days.last.mood, isNull);
      expect(days.last.sleepQuality, isNull);
    },
  );
}
