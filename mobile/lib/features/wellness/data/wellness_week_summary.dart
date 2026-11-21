class WellnessDaySummary {
  final DateTime date;
  final String? mood;
  final int? sleepQuality;

  const WellnessDaySummary({
    required this.date,
    required this.mood,
    required this.sleepQuality,
  });
}

List<WellnessDaySummary> buildWellnessWeekSummary({
  required DateTime today,
  required List<Map<String, dynamic>> moodEntries,
  required List<Map<String, dynamic>> sleepEntries,
}) {
  final moodsByDate = <String, String>{};
  for (final entry in moodEntries) {
    if (entry['is_mine'] != true) continue;
    final date = entry['entry_date'];
    final mood = entry['mood'];
    if (date is String && mood is String) moodsByDate[date] = mood;
  }

  final sleepByDate = <String, int>{};
  for (final entry in sleepEntries) {
    if (entry['is_mine'] != true) continue;
    final date = entry['entry_date'];
    final quality = entry['quality'];
    if (date is String && quality is num) {
      sleepByDate[date] = quality.toInt();
    }
  }

  final start = DateTime(
    today.year,
    today.month,
    today.day,
  ).subtract(const Duration(days: 6));
  return List<WellnessDaySummary>.generate(7, (index) {
    final date = start.add(Duration(days: index));
    final key =
        '${date.year.toString().padLeft(4, '0')}-'
        '${date.month.toString().padLeft(2, '0')}-'
        '${date.day.toString().padLeft(2, '0')}';
    return WellnessDaySummary(
      date: date,
      mood: moodsByDate[key],
      sleepQuality: sleepByDate[key],
    );
  });
}
