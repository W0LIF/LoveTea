import '../../../core/network/api_client.dart';

class DateIdea {
  final String id;
  final String coupleId;
  final String proposerUserId;
  final String title;
  final String description;
  final String category;
  final int? budgetMin;
  final int? budgetMax;
  final String status;
  final DateTime createdAt;
  final DateTime? acceptedAt;
  final DateTime? completedAt;

  const DateIdea({
    required this.id,
    required this.coupleId,
    required this.proposerUserId,
    required this.title,
    required this.description,
    required this.category,
    this.budgetMin,
    this.budgetMax,
    required this.status,
    required this.createdAt,
    this.acceptedAt,
    this.completedAt,
  });

  factory DateIdea.fromJson(Map<String, dynamic> json) {
    return DateIdea(
      id: json['id'] as String,
      coupleId: json['couple_id'] as String,
      proposerUserId: json['proposer_user_id'] as String,
      title: json['title'] as String,
      description: json['description'] as String? ?? '',
      category: json['category'] as String? ?? 'general',
      budgetMin: json['budget_min'] != null
          ? (json['budget_min'] as num).toInt()
          : null,
      budgetMax: json['budget_max'] != null
          ? (json['budget_max'] as num).toInt()
          : null,
      status: json['status'] as String,
      createdAt: DateTime.parse(json['created_at'] as String),
      acceptedAt: json['accepted_at'] != null
          ? DateTime.parse(json['accepted_at'] as String)
          : null,
      completedAt: json['completed_at'] != null
          ? DateTime.parse(json['completed_at'] as String)
          : null,
    );
  }
}

class DatesApi {
  final ApiClient _apiClient;

  DatesApi(this._apiClient);

  Future<List<DateIdea>> listIdeas() async {
    final response = await _apiClient.dio.get('/dates/ideas');
    final data = response.data as List;
    return data
        .whereType<Map>()
        .map((item) => DateIdea.fromJson(Map<String, dynamic>.from(item)))
        .toList();
  }

  Future<List<DateIdea>> listRecommendations() async {
    final response = await _apiClient.dio.get('/dates/recommendations');
    final data = response.data as List;
    return data
        .whereType<Map>()
        .map((item) => DateIdea.fromJson(Map<String, dynamic>.from(item)))
        .toList();
  }

  Future<DateIdea> createIdea({
    required String title,
    required String description,
    required String category,
    int? budgetMin,
    int? budgetMax,
  }) async {
    final response = await _apiClient.dio.post(
      '/dates/ideas',
      data: {
        'title': title,
        'description': description,
        'category': category,
        if (budgetMin != null) 'budget_min': budgetMin,
        if (budgetMax != null) 'budget_max': budgetMax,
      },
    );
    return DateIdea.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }

  Future<DateIdea> getIdea(String id) async {
    final response = await _apiClient.dio.get('/dates/ideas/$id');
    return DateIdea.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }

  Future<void> updateIdeaStatus(String id, String status) async {
    await _apiClient.dio.patch('/dates/ideas/$id/status', data: {'status': status});
  }
}
