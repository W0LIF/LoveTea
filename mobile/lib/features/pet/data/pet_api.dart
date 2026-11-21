import '../../../core/network/api_client.dart';

class PetAppearance {
  final Map<String, dynamic> appearance;
  final int version;
  final DateTime updatedAt;

  const PetAppearance({
    required this.appearance,
    required this.version,
    required this.updatedAt,
  });

  factory PetAppearance.fromJson(Map<String, dynamic> json) {
    return PetAppearance(
      appearance: Map<String, dynamic>.from(json['appearance'] as Map),
      version: (json['version'] as num).toInt(),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }
}

class PetApi {
  final ApiClient _apiClient;

  PetApi(this._apiClient);

  Future<PetAppearance> getPetState() async {
    final response = await _apiClient.dio.get('/pet');
    return PetAppearance.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }

  Future<PetAppearance> updateAppearance(
    Map<String, dynamic> appearance,
  ) async {
    final response = await _apiClient.dio.patch(
      '/pet/appearance',
      data: {'appearance': appearance},
    );
    return PetAppearance.fromJson(
      Map<String, dynamic>.from(response.data as Map),
    );
  }
}
