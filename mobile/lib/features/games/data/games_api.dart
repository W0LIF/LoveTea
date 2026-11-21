import '../../../core/network/api_client.dart';

class GameSession {
  final String id;
  final String coupleId;
  final String playerUserId;
  final String opponentUserId;
  final String type;
  final int totalQuestions;
  final int correctAnswers;
  final int score;
  final String status;
  final DateTime startedAt;
  final DateTime? finishedAt;

  const GameSession({
    required this.id,
    required this.coupleId,
    required this.playerUserId,
    required this.opponentUserId,
    required this.type,
    required this.totalQuestions,
    required this.correctAnswers,
    required this.score,
    required this.status,
    required this.startedAt,
    this.finishedAt,
  });

  factory GameSession.fromJson(Map<String, dynamic> json) {
    return GameSession(
      id: json['id'] as String,
      coupleId: json['couple_id'] as String,
      playerUserId: json['player_user_id'] as String,
      opponentUserId: json['opponent_user_id'] as String,
      type: json['type'] as String,
      totalQuestions: (json['total_questions'] as num).toInt(),
      correctAnswers: (json['correct_answers'] as num).toInt(),
      score: (json['score'] as num).toInt(),
      status: json['status'] as String,
      startedAt: DateTime.parse(json['started_at'] as String),
      finishedAt: json['finished_at'] != null
          ? DateTime.parse(json['finished_at'] as String)
          : null,
    );
  }
}

class GamesApi {
  final ApiClient _apiClient;

  GamesApi(this._apiClient);

  Future<List<GameSession>> listSessions() async {
    final response = await _apiClient.dio.get('/games/sessions');
    final data = response.data as List;
    return data
        .whereType<Map>()
        .map((item) => GameSession.fromJson(Map<String, dynamic>.from(item)))
        .toList();
  }

  Future<GameSession> startSession({
    required String opponentUserId,
    required String type,
  }) async {
    final response = await _apiClient.dio.post(
      '/games/sessions',
      data: {'opponent_user_id': opponentUserId, 'type': type},
    );
    return GameSession.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }

  Future<GameSession> submitAnswer({
    required String sessionId,
    required Map<String, dynamic> answer,
  }) async {
    final response = await _apiClient.dio.post(
      '/games/sessions/$sessionId/answer',
      data: answer,
    );
    return GameSession.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }

  Future<GameSession> finishSession(String sessionId) async {
    final response = await _apiClient.dio.post(
      '/games/sessions/$sessionId/finish',
    );
    return GameSession.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }
}
