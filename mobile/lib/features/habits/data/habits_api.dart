import 'package:dio/dio.dart';

import '../../../core/network/api_client.dart';
import '../../planning/data/planning_api.dart';

class HabitMemberStats {
  final String userId;
  final int completionCount;
  final int successfulWeeks;
  final int currentStreakWeeks;

  const HabitMemberStats({
    required this.userId,
    required this.completionCount,
    required this.successfulWeeks,
    required this.currentStreakWeeks,
  });

  factory HabitMemberStats.fromJson(Map<String, dynamic> json) =>
      HabitMemberStats(
        userId: json['user_id'] as String,
        completionCount: (json['completion_count'] as num).toInt(),
        successfulWeeks: (json['successful_weeks'] as num).toInt(),
        currentStreakWeeks: (json['current_streak_weeks'] as num).toInt(),
      );
}

class HabitStats {
  final int targetPerWeek;
  final int totalCompletions;
  final List<HabitMemberStats> members;

  const HabitStats({
    required this.targetPerWeek,
    required this.totalCompletions,
    required this.members,
  });

  int get successfulWeeks =>
      members.fold(0, (total, member) => total + member.successfulWeeks);

  int get bestCurrentStreakWeeks => members.fold(
    0,
    (best, member) =>
        member.currentStreakWeeks > best ? member.currentStreakWeeks : best,
  );

  factory HabitStats.fromJson(Map<String, dynamic> json) {
    final rawMembers = json['members'];
    return HabitStats(
      targetPerWeek: (json['target_per_week'] as num).toInt(),
      totalCompletions: (json['total_completions'] as num).toInt(),
      members: rawMembers is List
          ? rawMembers
                .whereType<Map>()
                .map(
                  (member) => HabitMemberStats.fromJson(
                    Map<String, dynamic>.from(member),
                  ),
                )
                .toList()
          : const [],
    );
  }
}

class HabitsApi {
  final ApiClient _apiClient;

  HabitsApi(this._apiClient);

  Dio get _dio => _apiClient.dio;

  Future<List<Map<String, dynamic>>> listHabits() async {
    final response = await _dio.get('/habits');
    final data = response.data;
    if (data is! List) {
      throw const FormatException('Ожидался список привычек от сервера.');
    }
    return data
        .whereType<Map>()
        .map((item) => Map<String, dynamic>.from(item))
        .toList();
  }

  Future<List<Map<String, dynamic>>> listCompletions(String habitId) async {
    final now = DateTime.now();
    final response = await _dio.get(
      '/habits/$habitId/completions',
      queryParameters: {
        'from': PlanningApi.formatDate(now.subtract(const Duration(days: 6))),
        'to': PlanningApi.formatDate(now),
      },
    );
    final data = response.data;
    if (data is! List) {
      throw const FormatException('Ожидался список отметок от сервера.');
    }
    return data
        .whereType<Map>()
        .map((item) => Map<String, dynamic>.from(item))
        .toList();
  }

  Future<HabitStats> getStats(String habitId) async {
    final response = await _dio.get('/habits/$habitId/stats');
    return HabitStats.fromJson(Map<String, dynamic>.from(response.data as Map));
  }

  Future<void> createHabit({
    required String title,
    required String scope,
    required int targetPerWeek,
  }) async {
    await _dio.post(
      '/habits',
      data: {
        'scope': scope,
        'title': title.trim(),
        'target_per_week': targetPerWeek,
      },
    );
  }

  Future<void> completeToday(String habitId) async {
    await _dio.post(
      '/habits/$habitId/completions',
      data: {'completed_on': PlanningApi.formatDate(DateTime.now())},
    );
  }
}
