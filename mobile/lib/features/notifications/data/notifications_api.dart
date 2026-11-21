import '../../../core/network/api_client.dart';

class NotificationItem {
  final int id;
  final String userId;
  final String category;
  final String title;
  final String body;
  final Map<String, dynamic> data;
  final bool isRead;
  final DateTime createdAt;

  const NotificationItem({
    required this.id,
    required this.userId,
    required this.category,
    required this.title,
    required this.body,
    required this.data,
    required this.isRead,
    required this.createdAt,
  });

  factory NotificationItem.fromJson(Map<String, dynamic> json) {
    return NotificationItem(
      id: (json['id'] as num).toInt(),
      userId: json['user_id'] as String,
      category: json['category'] as String,
      title: json['title'] as String,
      body: json['body'] as String,
      data: Map<String, dynamic>.from(json['data'] as Map? ?? {}),
      isRead: json['is_read'] as bool? ?? false,
      createdAt: DateTime.parse(json['created_at'] as String),
    );
  }
}

class NotificationSettings {
  final String userId;
  final List<String> enabledCategories;
  final int quietHoursStart;
  final int quietHoursEnd;
  final DateTime updatedAt;

  const NotificationSettings({
    required this.userId,
    required this.enabledCategories,
    required this.quietHoursStart,
    required this.quietHoursEnd,
    required this.updatedAt,
  });

  factory NotificationSettings.fromJson(Map<String, dynamic> json) {
    return NotificationSettings(
      userId: json['user_id'] as String,
      enabledCategories: List<String>.from(
        (json['enabled_categories'] as List).cast<String>(),
      ),
      quietHoursStart: (json['quiet_hours_start'] as num).toInt(),
      quietHoursEnd: (json['quiet_hours_end'] as num).toInt(),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }
}

class NotificationsApi {
  final ApiClient _apiClient;

  NotificationsApi(this._apiClient);

  Future<List<NotificationItem>> list({bool unreadOnly = false, int limit = 50}) async {
    final response = await _apiClient.dio.get(
      '/notifications',
      queryParameters: {'unread': unreadOnly ? 'true' : 'false', 'limit': limit},
    );
    final data = response.data as List;
    return data
        .whereType<Map>()
        .map((item) => NotificationItem.fromJson(Map<String, dynamic>.from(item)))
        .toList();
  }

  Future<NotificationSettings> getSettings() async {
    final response = await _apiClient.dio.get('/notifications/settings');
    return NotificationSettings.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }

  Future<NotificationSettings> updateSettings({
    List<String>? enabledCategories,
    int? quietHoursStart,
    int? quietHoursEnd,
  }) async {
    final response = await _apiClient.dio.patch(
      '/notifications/settings',
      data: {
        if (enabledCategories != null) 'enabled_categories': enabledCategories,
        if (quietHoursStart != null) 'quiet_hours_start': quietHoursStart,
        if (quietHoursEnd != null) 'quiet_hours_end': quietHoursEnd,
      },
    );
    return NotificationSettings.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }

  Future<void> markRead(int id) async {
    await _apiClient.dio.post('/notifications/$id/read');
  }

  Future<void> markAllRead() async {
    await _apiClient.dio.post('/notifications/read-all');
  }
}
