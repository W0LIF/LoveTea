import '../../../core/network/api_client.dart';

class ChatMessage {
  final String id;
  final String conversationId;
  final String senderUserId;
  final String content;
  final String? ciphertext;
  final int version;
  final DateTime createdAt;

  const ChatMessage({
    required this.id,
    required this.conversationId,
    required this.senderUserId,
    required this.content,
    this.ciphertext,
    required this.version,
    required this.createdAt,
  });

  factory ChatMessage.fromJson(Map<String, dynamic> json) {
    return ChatMessage(
      id: json['id'] as String,
      conversationId: json['conversation_id'] as String,
      senderUserId: json['sender_user_id'] as String,
      content: json['content'] as String,
      ciphertext: json['ciphertext'] as String?,
      version: (json['version'] as num).toInt(),
      createdAt: DateTime.parse(json['created_at'] as String),
    );
  }
}

class ChatReaction {
  final String id;
  final String messageId;
  final String userId;
  final String emoji;
  final DateTime createdAt;

  const ChatReaction({
    required this.id,
    required this.messageId,
    required this.userId,
    required this.emoji,
    required this.createdAt,
  });

  factory ChatReaction.fromJson(Map<String, dynamic> json) {
    return ChatReaction(
      id: json['id'] as String,
      messageId: json['message_id'] as String,
      userId: json['user_id'] as String,
      emoji: json['emoji'] as String,
      createdAt: DateTime.parse(json['created_at'] as String),
    );
  }
}

class ChatApi {
  final ApiClient _apiClient;

  ChatApi(this._apiClient);

  Future<List<ChatMessage>> listMessages(String conversationId) async {
    final response = await _apiClient.dio.get(
      '/chat/messages',
      queryParameters: {'conversation_id': conversationId},
    );
    final data = response.data as List;
    return data
        .whereType<Map>()
        .map((item) => ChatMessage.fromJson(Map<String, dynamic>.from(item)))
        .toList();
  }

  Future<ChatMessage> sendMessage({
    required String conversationId,
    required String content,
  }) async {
    final response = await _apiClient.dio.post(
      '/chat/messages',
      data: {
        'conversation_id': conversationId,
        'content': content,
      },
    );
    return ChatMessage.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }

  Future<void> addReaction(String messageId, String emoji) async {
    await _apiClient.dio.post('/chat/messages/$messageId/reaction',
        data: {'emoji': emoji});
  }

  Future<void> removeReaction(String messageId, String emoji) async {
    await _apiClient.dio.delete('/chat/messages/$messageId/reaction',
        data: {'emoji': emoji});
  }

  Future<List<ChatReaction>> listReactions(String messageId) async {
    final response = await _apiClient.dio.get('/chat/messages/$messageId/reactions');
    final data = response.data as List;
    return data
        .whereType<Map>()
        .map((item) => ChatReaction.fromJson(Map<String, dynamic>.from(item)))
        .toList();
  }
}
