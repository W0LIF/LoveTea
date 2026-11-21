import '../../../core/network/api_client.dart';

class WheelSector {
  final String id;
  final String coupleId;
  final String title;
  final String description;
  final bool requiresConfirmation;
  final int position;
  final DateTime createdAt;

  const WheelSector({
    required this.id,
    required this.coupleId,
    required this.title,
    required this.description,
    required this.requiresConfirmation,
    required this.position,
    required this.createdAt,
  });

  factory WheelSector.fromJson(Map<String, dynamic> json) {
    return WheelSector(
      id: json['id'] as String,
      coupleId: json['couple_id'] as String,
      title: json['title'] as String,
      description: json['description'] as String? ?? '',
      requiresConfirmation: json['requires_confirmation'] as bool? ?? false,
      position: (json['position'] as num).toInt(),
      createdAt: DateTime.parse(json['created_at'] as String),
    );
  }
}

class WheelSpin {
  final String id;
  final String coupleId;
  final String spunByUserId;
  final String? selectedSectorId;
  final bool confirmedByBoth;
  final bool cancelled;
  final DateTime? confirmedAt;
  final DateTime createdAt;

  const WheelSpin({
    required this.id,
    required this.coupleId,
    required this.spunByUserId,
    this.selectedSectorId,
    required this.confirmedByBoth,
    required this.cancelled,
    this.confirmedAt,
    required this.createdAt,
  });

  factory WheelSpin.fromJson(Map<String, dynamic> json) {
    return WheelSpin(
      id: json['id'] as String,
      coupleId: json['couple_id'] as String,
      spunByUserId: json['spun_by_user_id'] as String,
      selectedSectorId: json['selected_sector_id'] as String?,
      confirmedByBoth: json['confirmed_by_both'] as bool? ?? false,
      cancelled: json['cancelled'] as bool? ?? false,
      confirmedAt: json['confirmed_at'] != null
          ? DateTime.parse(json['confirmed_at'] as String)
          : null,
      createdAt: DateTime.parse(json['created_at'] as String),
    );
  }
}

class WheelApi {
  final ApiClient _apiClient;

  WheelApi(this._apiClient);

  Future<List<WheelSector>> listSectors() async {
    final response = await _apiClient.dio.get('/wheel/sectors');
    final data = response.data as List;
    return data
        .whereType<Map>()
        .map((item) => WheelSector.fromJson(Map<String, dynamic>.from(item)))
        .toList();
  }

  Future<WheelSector> createSector({
    required String title,
    required String description,
    required bool requiresConfirmation,
    required int position,
  }) async {
    final response = await _apiClient.dio.post(
      '/wheel/sectors',
      data: {
        'title': title,
        'description': description,
        'requires_confirmation': requiresConfirmation,
        'position': position,
      },
    );
    return WheelSector.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }

  Future<void> deleteSector(String id) async {
    await _apiClient.dio.delete('/wheel/sectors/$id');
  }

  Future<Map<String, dynamic>> spin() async {
    final response = await _apiClient.dio.post('/wheel/spin');
    return Map<String, dynamic>.from(response.data as Map);
  }

  Future<Map<String, dynamic>> confirmSpin(String spinId) async {
    final response = await _apiClient.dio.post('/wheel/spin/$spinId/confirm');
    return Map<String, dynamic>.from(response.data as Map);
  }

  Future<List<WheelSpin>> history() async {
    final response = await _apiClient.dio.get('/wheel/history');
    final data = response.data as List;
    return data
        .whereType<Map>()
        .map((item) => WheelSpin.fromJson(Map<String, dynamic>.from(item)))
        .toList();
  }
}
