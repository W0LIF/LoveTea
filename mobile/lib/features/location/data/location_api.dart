import '../../../core/network/api_client.dart';

class LocationPosition {
  final int id;
  final String userId;
  final String coupleId;
  final double latitude;
  final double longitude;
  final double? accuracy;
  final DateTime capturedAt;

  const LocationPosition({
    required this.id,
    required this.userId,
    required this.coupleId,
    required this.latitude,
    required this.longitude,
    this.accuracy,
    required this.capturedAt,
  });

  factory LocationPosition.fromJson(Map<String, dynamic> json) {
    return LocationPosition(
      id: (json['id'] as num).toInt(),
      userId: json['user_id'] as String,
      coupleId: json['couple_id'] as String,
      latitude: (json['latitude'] as num).toDouble(),
      longitude: (json['longitude'] as num).toDouble(),
      accuracy: json['accuracy'] != null
          ? (json['accuracy'] as num).toDouble()
          : null,
      capturedAt: DateTime.parse(json['captured_at'] as String),
    );
  }
}

class LocationSharing {
  final String userId;
  final String coupleId;
  final bool isActive;
  final DateTime? lastSharingAt;

  const LocationSharing({
    required this.userId,
    required this.coupleId,
    required this.isActive,
    this.lastSharingAt,
  });

  factory LocationSharing.fromJson(Map<String, dynamic> json) {
    return LocationSharing(
      userId: json['user_id'] as String,
      coupleId: json['couple_id'] as String,
      isActive: json['is_active'] as bool,
      lastSharingAt: json['last_sharing_at'] != null
          ? DateTime.parse(json['last_sharing_at'] as String)
          : null,
    );
  }
}

class LocationGeofence {
  final String id;
  final String coupleId;
  final String userId;
  final String name;
  final double latitude;
  final double longitude;
  final double radiusMeters;
  final DateTime createdAt;

  const LocationGeofence({
    required this.id,
    required this.coupleId,
    required this.userId,
    required this.name,
    required this.latitude,
    required this.longitude,
    required this.radiusMeters,
    required this.createdAt,
  });

  factory LocationGeofence.fromJson(Map<String, dynamic> json) {
    return LocationGeofence(
      id: json['id'] as String,
      coupleId: json['couple_id'] as String,
      userId: json['user_id'] as String,
      name: json['name'] as String,
      latitude: (json['latitude'] as num).toDouble(),
      longitude: (json['longitude'] as num).toDouble(),
      radiusMeters: (json['radius_meters'] as num).toDouble(),
      createdAt: DateTime.parse(json['created_at'] as String),
    );
  }
}

class LocationApi {
  final ApiClient _apiClient;

  LocationApi(this._apiClient);

  Future<LocationSharing> getSharing() async {
    final response = await _apiClient.dio.get('/location/sharing');
    return LocationSharing.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }

  Future<LocationSharing> toggleSharing({required bool active}) async {
    final response = await _apiClient.dio.post(
      '/location/sharing',
      data: {'is_active': active},
    );
    return LocationSharing.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }

  Future<void> updatePosition({
    required double latitude,
    required double longitude,
    double? accuracy,
  }) async {
    await _apiClient.dio.post(
      '/location/position',
      data: {
        'latitude': latitude,
        'longitude': longitude,
        if (accuracy != null) 'accuracy': accuracy,
      },
    );
  }

  Future<LocationPosition?> getPartnerPosition() async {
    final response = await _apiClient.dio.get('/location/partner-position');
    if (response.data == null) return null;
    return LocationPosition.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }

  Future<List<LocationGeofence>> listGeofences() async {
    final response = await _apiClient.dio.get('/location/geofences');
    final data = response.data as List;
    return data
        .whereType<Map>()
        .map((item) => LocationGeofence.fromJson(Map<String, dynamic>.from(item)))
        .toList();
  }

  Future<LocationGeofence> createGeofence({
    required String name,
    required double latitude,
    required double longitude,
    required double radiusMeters,
  }) async {
    final response = await _apiClient.dio.post(
      '/location/geofences',
      data: {
        'name': name,
        'latitude': latitude,
        'longitude': longitude,
        'radius_meters': radiusMeters,
      },
    );
    return LocationGeofence.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }
}
