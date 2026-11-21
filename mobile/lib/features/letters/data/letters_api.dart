import '../../../core/network/api_client.dart';

class Letter {
  final String id;
  final String senderUserId;
  final String recipientUserId;
  final String coupleId;
  final String content;
  final DateTime cooldownUntil;
  final String status;
  final int version;
  final DateTime createdAt;
  final DateTime? sentAt;
  final DateTime? readAt;

  const Letter({
    required this.id,
    required this.senderUserId,
    required this.recipientUserId,
    required this.coupleId,
    required this.content,
    required this.cooldownUntil,
    required this.status,
    required this.version,
    required this.createdAt,
    this.sentAt,
    this.readAt,
  });

  factory Letter.fromJson(Map<String, dynamic> json) {
    return Letter(
      id: json['id'] as String,
      senderUserId: json['sender_user_id'] as String,
      recipientUserId: json['recipient_user_id'] as String,
      coupleId: json['couple_id'] as String,
      content: json['content'] as String,
      cooldownUntil: DateTime.parse(json['cooldown_until'] as String),
      status: json['status'] as String,
      version: (json['version'] as num).toInt(),
      createdAt: DateTime.parse(json['created_at'] as String),
      sentAt: json['sent_at'] != null
          ? DateTime.parse(json['sent_at'] as String)
          : null,
      readAt: json['read_at'] != null
          ? DateTime.parse(json['read_at'] as String)
          : null,
    );
  }
}

class LettersApi {
  final ApiClient _apiClient;

  LettersApi(this._apiClient);

  Future<List<Letter>> listMyLetters() async {
    final response = await _apiClient.dio.get('/letters');
    final data = response.data as List;
    return data
        .whereType<Map>()
        .map((item) => Letter.fromJson(Map<String, dynamic>.from(item)))
        .toList();
  }

  Future<List<Letter>> listForMe() async {
    final response = await _apiClient.dio.get('/letters/for-me');
    final data = response.data as List;
    return data
        .whereType<Map>()
        .map((item) => Letter.fromJson(Map<String, dynamic>.from(item)))
        .toList();
  }

  Future<Letter> createLetter({
    required String recipientUserId,
    required String coupleId,
    required String content,
  }) async {
    final response = await _apiClient.dio.post(
      '/letters',
      data: {
        'recipient_user_id': recipientUserId,
        'couple_id': coupleId,
        'content': content,
      },
    );
    return Letter.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }

  Future<Letter> getLetter(String id) async {
    final response = await _apiClient.dio.get('/letters/$id');
    return Letter.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }

  Future<void> markRead(String id) async {
    await _apiClient.dio.post('/letters/$id/read');
  }

  Future<void> archive(String id) async {
    await _apiClient.dio.post('/letters/$id/archive');
  }

  Future<Map<String, dynamic>> cooldownRemaining() async {
    final response = await _apiClient.dio.get('/letters/cooldown');
    return Map<String, dynamic>.from(response.data as Map);
  }
}
